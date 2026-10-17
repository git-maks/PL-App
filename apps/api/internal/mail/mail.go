// Package mail sends transactional e-mail over SMTP (STARTTLS when the server offers it).
package mail

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type SMTP struct {
	Host, User, Password, From string
	Port                       int
}

var ErrNotConfigured = errors.New("e-mail delivery is not configured (UQ_SMTP_HOST)")

func (m SMTP) Send(ctx context.Context, to, subject, body string) error {
	if m.Host == "" {
		return ErrNotConfigured
	}
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return fmt.Errorf("UQ_SMTP_FROM: %w", err)
	}
	msg := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(m.Host+":"+strconv.Itoa(m.Port), auth, from.Address, []string{to}, []byte(msg))
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
		return errors.New("smtp timeout")
	}
}
