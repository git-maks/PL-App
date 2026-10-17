package game_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/git-maks/pl-app/apps/api/internal/catalog"
	"github.com/git-maks/pl-app/apps/api/internal/game"
)

type fixtureFile struct {
	Config game.Config `json:"config"`
	Cases  []struct {
		Name     string       `json:"name"`
		Scans    []game.Input `json:"scans"`
		Expected struct {
			Stats      game.Stats   `json:"stats"`
			XP         float64      `json:"xp"`
			Level      int          `json:"level"`
			Archetypes []game.Share `json:"archetypes"`
			TitleKey   *string      `json:"titleKey"`
			Programs   []string     `json:"programs"`
			Clubs      []string     `json:"clubs"`
		} `json:"expected"`
	} `json:"cases"`
}

// The same fixtures are asserted by apps/web (Vitest): both engines must agree exactly.
func TestGoldenFixtures(t *testing.T) {
	raw, err := os.ReadFile("../../../../packages/contracts/fixtures/scoring/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			p := game.BuildProfileFromInputs(c.Scans, f.Config)
			if p.Stats != c.Expected.Stats {
				t.Errorf("stats = %v, want %v", p.Stats, c.Expected.Stats)
			}
			if p.XP != c.Expected.XP || p.Level != c.Expected.Level {
				t.Errorf("xp/level = %v/%v, want %v/%v", p.XP, p.Level, c.Expected.XP, c.Expected.Level)
			}
			if !reflect.DeepEqual(p.Archetypes, c.Expected.Archetypes) {
				t.Errorf("archetypes = %v, want %v", p.Archetypes, c.Expected.Archetypes)
			}
			want := ""
			if c.Expected.TitleKey != nil {
				want = *c.Expected.TitleKey
			}
			if p.TitleKey != want {
				t.Errorf("titleKey = %q, want %q", p.TitleKey, want)
			}
			if got := slugs(game.Recommend(p.Stats, cat.ProgramItems(), 3)); !reflect.DeepEqual(got, c.Expected.Programs) {
				t.Errorf("programs = %v, want %v", got, c.Expected.Programs)
			}
			if got := slugs(game.Recommend(p.Stats, cat.ClubItems(), 3)); !reflect.DeepEqual(got, c.Expected.Clubs) {
				t.Errorf("clubs = %v, want %v", got, c.Expected.Clubs)
			}
		})
	}
}

func slugs(r []game.Recommendation) []string {
	out := []string{}
	for _, x := range r {
		out = append(out, x.Slug)
	}
	return out
}

func TestApportionSumsTo100(t *testing.T) {
	for _, v := range [][]float64{{1, 1, 1}, {5, 3, 1}, {0.7, 0.2}, {185, 125, 105}, {1}} {
		sum := 0
		for _, p := range game.Apportion(v, 5) {
			sum += p
		}
		if sum != 100 {
			t.Errorf("Apportion(%v) sums to %d", v, sum)
		}
	}
}

func TestStatsJSONRoundTrip(t *testing.T) {
	s := game.Stats{1, 2, 3, 4, 5, 6, 7, 8}
	b, _ := json.Marshal(s)
	var back game.Stats
	if err := json.Unmarshal(b, &back); err != nil || back != s {
		t.Fatalf("round trip failed: %s %v %v", b, back, err)
	}
}
