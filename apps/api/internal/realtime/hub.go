// Package realtime delivers per-user server-sent events. Every publish goes through PostgreSQL
// NOTIFY, and every instance LISTENs and fans out to its own SSE clients, so horizontal scaling
// needs no extra broker (spec §7.6).
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const channel = "uq_events"

// pg_notify payloads are limited to 8000 bytes; larger events are delivered locally only.
const maxNotifyPayload = 7900

type Message struct {
	Type string
	Data json.RawMessage
}

type envelope struct {
	Users []string        `json:"u"`
	Type  string          `json:"t"`
	Data  json.RawMessage `json:"d"`
}

type Hub struct {
	db  *pgxpool.Pool
	log *slog.Logger

	mu      sync.RWMutex
	clients map[string]map[chan Message]struct{}
}

func NewHub(db *pgxpool.Pool, log *slog.Logger) *Hub {
	return &Hub{db: db, log: log, clients: map[string]map[chan Message]struct{}{}}
}

// Subscribe registers an SSE client for a user. The returned function unsubscribes.
func (h *Hub) Subscribe(userID string) (<-chan Message, func()) {
	ch := make(chan Message, 32)
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[chan Message]struct{}{}
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.clients[userID], ch)
		if len(h.clients[userID]) == 0 {
			delete(h.clients, userID)
		}
		h.mu.Unlock()
	}
}

// Connections returns the number of open SSE connections on this instance.
func (h *Hub) Connections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, set := range h.clients {
		n += len(set)
	}
	return n
}

func (h *Hub) deliver(users []string, m Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, u := range users {
		for ch := range h.clients[u] {
			select {
			case ch <- m:
			default: // slow client: drop rather than block the publisher; the app refreshes on reconnect
			}
		}
	}
}

// Publish sends an event to the given users on every instance.
func (h *Hub) Publish(ctx context.Context, users []string, eventType string, data any) {
	if len(users) == 0 {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		h.log.Error("realtime marshal", "err", err)
		return
	}
	payload, _ := json.Marshal(envelope{Users: users, Type: eventType, Data: raw})
	if h.db == nil || len(payload) > maxNotifyPayload {
		h.deliver(users, Message{Type: eventType, Data: raw})
		return
	}
	if _, err := h.db.Exec(ctx, `select pg_notify($1, $2)`, channel, string(payload)); err != nil {
		h.log.Warn("pg_notify failed, delivering locally", "err", err)
		h.deliver(users, Message{Type: eventType, Data: raw})
	}
}

// Run LISTENs for events until ctx is cancelled, reconnecting with backoff.
func (h *Hub) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := h.listen(ctx); err != nil && ctx.Err() == nil {
			h.log.Warn("realtime listener", "err", err, "retry_in", backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (h *Hub) listen(ctx context.Context) error {
	conn, err := h.db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "listen "+channel); err != nil {
		return err
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var env envelope
		if err := json.Unmarshal([]byte(n.Payload), &env); err != nil {
			continue
		}
		h.deliver(env.Users, Message{Type: env.Type, Data: env.Data})
	}
}
