// Package catalog loads the shared knowledge-base content (content/*.json) embedded in the binary.
package catalog

import (
	"encoding/json"
	"fmt"

	"github.com/git-maks/pl-app/apps/api/internal/game"
	"github.com/git-maks/pl-app/content"
)

type Stat struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Archetype   string `json:"archetype"`
	Description string `json:"description"`
}

type Archetypes struct {
	Stats   []Stat `json:"stats"`
	Hybrids map[string]struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"hybrids"`
	Defaults struct {
		game.Config
		BasePoints int `json:"basePoints"`
	} `json:"defaults"`
}

type Program struct {
	Slug    string      `json:"slug"`
	Name    string      `json:"name"`
	Faculty string      `json:"faculty"`
	Vector  game.Vector `json:"vector"`
}

type Club struct {
	Slug    string      `json:"slug"`
	Name    string      `json:"name"`
	Faculty string      `json:"faculty"`
	Vector  game.Vector `json:"vector"`
}

type DemoStation struct {
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Kind        string      `json:"kind"`
	Faculty     *string     `json:"faculty"`
	Poi         string      `json:"poi"`
	Description string      `json:"description"`
	Weights     game.Vector `json:"weights"`
	Team        *struct {
		Mode       bool `json:"mode"`
		MinMembers int  `json:"minMembers"`
	} `json:"team"`
	Sessions []struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Room  string `json:"room"`
	} `json:"sessions"`
}

type DemoEvent struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Timezone        string `json:"timezone"`
	MinScansForCard int    `json:"minScansForCard"`
	Day             struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"day"`
	Stations []DemoStation `json:"stations"`
}

type Catalog struct {
	Archetypes Archetypes
	Programs   []Program
	Clubs      []Club
	DemoEvent  DemoEvent
}

func read(name string, v any) error {
	b, err := content.FS.ReadFile(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Load parses the embedded content files.
func Load() (*Catalog, error) {
	c := &Catalog{}
	for name, v := range map[string]any{
		"archetypes.json": &c.Archetypes,
		"programs.json":   &c.Programs,
		"clubs.json":      &c.Clubs,
		"demo-event.json": &c.DemoEvent,
	} {
		if err := read(name, v); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Catalog) ProgramItems() []game.Item {
	out := make([]game.Item, len(c.Programs))
	for i, p := range c.Programs {
		out[i] = game.Item{Slug: p.Slug, Vector: p.Vector}
	}
	return out
}

func (c *Catalog) ClubItems() []game.Item {
	out := make([]game.Item, len(c.Clubs))
	for i, p := range c.Clubs {
		out[i] = game.Item{Slug: p.Slug, Vector: p.Vector}
	}
	return out
}
