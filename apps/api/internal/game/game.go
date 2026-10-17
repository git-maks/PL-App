// Package game implements the UniQuest scoring engine (spec §8).
//
// It mirrors apps/web/src/lib/game/scoring.ts operation by operation (same summation order and
// rounding) so both produce bit-identical results; the shared golden fixtures in
// packages/contracts/fixtures/scoring verify that. Change the rules in both places.
package game

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// Codes is the canonical stat order; ties are always broken by this order.
var Codes = [8]string{"STR", "INT", "MAN", "KRE", "ENE", "CHA", "WIS", "VIT"}

var codeIndex = func() map[string]int {
	m := make(map[string]int, len(Codes))
	for i, c := range Codes {
		m[c] = i
	}
	return m
}()

// IsCode reports whether s is a valid stat code.
func IsCode(s string) bool { _, ok := codeIndex[s]; return ok }

// Stats holds one value per stat in canonical order. It marshals as {"STR": …, …}.
type Stats [8]float64

func (s Stats) MarshalJSON() ([]byte, error) {
	m := make(map[string]float64, 8)
	for i, c := range Codes {
		m[c] = s[i]
	}
	return json.Marshal(m)
}

func (s *Stats) UnmarshalJSON(b []byte) error {
	var m map[string]float64
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*s = Stats{}
	for k, v := range m {
		i, ok := codeIndex[k]
		if !ok {
			return fmt.Errorf("unknown stat %q", k)
		}
		s[i] = v
	}
	return nil
}

// Get returns the value of a stat by code.
func (s Stats) Get(code string) float64 { return s[codeIndex[code]] }

// Vector is a partial map of stat weights (station weights, programme profiles).
type Vector map[string]float64

func (v Vector) at(i int) float64 { return v[Codes[i]] }

// Config mirrors the event game configuration.
type Config struct {
	RatingMultipliers  []float64 `json:"ratingMultipliers"`
	LevelDivisor       float64   `json:"levelDivisor"`
	TeamBonus          float64   `json:"teamBonus"`
	ArchetypeThreshold float64   `json:"archetypeThreshold"`
	MaxArchetypes      int       `json:"maxArchetypes"`
	RoundingStep       int       `json:"roundingStep"`
	MinScansForCard    int       `json:"minScansForCard"`
}

// Validate checks the configuration is usable.
func (c Config) Validate() error {
	if len(c.RatingMultipliers) != 5 {
		return fmt.Errorf("ratingMultipliers must have 5 entries")
	}
	if c.LevelDivisor <= 0 || c.RoundingStep <= 0 || 100%c.RoundingStep != 0 || c.MaxArchetypes < 1 {
		return fmt.Errorf("invalid game config")
	}
	return nil
}

// Input is one visit as the engine sees it.
type Input struct {
	BasePoints float64 `json:"basePoints"`
	Weights    Vector  `json:"weights"`
	Rating     *int    `json:"rating"`
	Team       bool    `json:"team"`
}

// roundHalfUp matches JavaScript's Math.round for the non-negative values used here.
func roundHalfUp(x float64) float64 { return math.Floor(x + 0.5) }

// Round2 rounds to two decimals exactly like the TypeScript engine.
func Round2(x float64) float64 { return roundHalfUp(x*100) / 100 }

// Deltas returns the points a visit adds: base × weight × f(rating) × teamBonus.
func Deltas(in Input, cfg Config) Stats {
	var out Stats
	if in.Rating == nil {
		return out
	}
	r := *in.Rating
	if r < 1 || r > len(cfg.RatingMultipliers) {
		return out
	}
	f := cfg.RatingMultipliers[r-1]
	if in.Team {
		f *= cfg.TeamBonus
	}
	for i := range Codes {
		if w := in.Weights.at(i); w > 0 {
			out[i] = Round2(in.BasePoints * w * f)
		}
	}
	return out
}

// Sum adds deltas, rounding after each addition like the TypeScript engine.
func Sum(all []Stats) Stats {
	var out Stats
	for _, d := range all {
		for i := range out {
			out[i] = Round2(out[i] + d[i])
		}
	}
	return out
}

// XP is the rounded total of all stats.
func XP(s Stats) float64 {
	t := 0.0
	for i := range s {
		t += s[i]
	}
	return Round2(t)
}

// Level = 1 + floor(sqrt(xp / divisor)).
func Level(xp, divisor float64) int {
	return 1 + int(math.Floor(math.Sqrt(math.Max(0, xp)/divisor)))
}

// LevelProgress describes the XP window of the current level.
type LevelProgress struct {
	Current float64 `json:"current"`
	Next    float64 `json:"next"`
	Ratio   float64 `json:"ratio"`
}

func Progress(xp, divisor float64) LevelProgress {
	l := float64(Level(xp, divisor))
	cur := divisor * (l - 1) * (l - 1)
	next := divisor * l * l
	return LevelProgress{Current: cur, Next: next, Ratio: (xp - cur) / (next - cur)}
}

// Rank returns stat indices sorted by value desc, ties by canonical order.
func Rank(s Stats) []int {
	idx := []int{0, 1, 2, 3, 4, 5, 6, 7}
	sort.SliceStable(idx, func(a, b int) bool {
		if s[idx[a]] != s[idx[b]] {
			return s[idx[a]] > s[idx[b]]
		}
		return idx[a] < idx[b]
	})
	return idx
}

// Apportion rounds shares to `step` percent with the largest-remainder method (sums to 100).
func Apportion(values []float64, step int) []int {
	units := int(roundHalfUp(100 / float64(step)))
	total := 0.0
	for _, v := range values {
		total += v
	}
	out := make([]int, len(values))
	if total <= 0 {
		return out
	}
	type rem struct {
		i int
		r float64
	}
	rems := make([]rem, len(values))
	left := units
	for i, v := range values {
		q := v / total * float64(units)
		f := math.Floor(q + 1e-9)
		out[i] = int(f)
		left -= int(f)
		rems[i] = rem{i, q - f}
	}
	sort.SliceStable(rems, func(a, b int) bool {
		if rems[a].r != rems[b].r {
			return rems[a].r > rems[b].r
		}
		return rems[a].i < rems[b].i
	})
	for _, r := range rems {
		if left <= 0 {
			break
		}
		out[r.i]++
		left--
	}
	for i := range out {
		out[i] *= step
	}
	return out
}

// Share is one archetype and its rounded percentage.
type Share struct {
	Code    string `json:"code"`
	Percent int    `json:"percent"`
}

// Archetypes picks up to MaxArchetypes stats with ≥ threshold share and renormalises them.
func Archetypes(s Stats, cfg Config) []Share {
	xp := XP(s)
	if xp <= 0 {
		return []Share{}
	}
	ranked := Rank(s)
	var picked []int
	for _, i := range ranked {
		if s[i]/xp >= cfg.ArchetypeThreshold && len(picked) < cfg.MaxArchetypes {
			picked = append(picked, i)
		}
	}
	if len(picked) == 0 {
		picked = ranked[:1]
	}
	vals := make([]float64, len(picked))
	for k, i := range picked {
		vals[k] = s[i]
	}
	pct := Apportion(vals, cfg.RoundingStep)
	out := []Share{}
	for k, i := range picked {
		if pct[k] > 0 {
			out = append(out, Share{Code: Codes[i], Percent: pct[k]})
		}
	}
	return out
}

// TitleKey is "A+B" (canonical order) for two or more archetypes, "A" for one, "" for none.
func TitleKey(a []Share) string {
	switch len(a) {
	case 0:
		return ""
	case 1:
		return a[0].Code
	}
	x, y := a[0].Code, a[1].Code
	if codeIndex[x] > codeIndex[y] {
		x, y = y, x
	}
	return x + "+" + y
}

// Profile is the computed state of a player.
type Profile struct {
	Stats         Stats         `json:"stats"`
	XP            float64       `json:"xp"`
	Level         int           `json:"level"`
	LevelProgress LevelProgress `json:"levelProgress"`
	RatedCount    int           `json:"ratedCount"`
	Archetypes    []Share       `json:"archetypes"`
	TitleKey      string        `json:"titleKey"`
}

// BuildProfile computes a profile from stored per-visit deltas.
func BuildProfile(deltas []Stats, rated int, cfg Config) Profile {
	s := Sum(deltas)
	xp := XP(s)
	a := Archetypes(s, cfg)
	return Profile{Stats: s, XP: xp, Level: Level(xp, cfg.LevelDivisor), LevelProgress: Progress(xp, cfg.LevelDivisor), RatedCount: rated, Archetypes: a, TitleKey: TitleKey(a)}
}

// BuildProfileFromInputs is the TypeScript buildProfile equivalent (used by fixtures).
func BuildProfileFromInputs(in []Input, cfg Config) Profile {
	ds := make([]Stats, len(in))
	rated := 0
	for i, x := range in {
		ds[i] = Deltas(x, cfg)
		if x.Rating != nil {
			rated++
		}
	}
	return BuildProfile(ds, rated, cfg)
}

// Item is anything with a stat profile that can be recommended.
type Item struct {
	Slug   string
	Vector Vector
}

// Recommendation is a cosine-similarity match with the two strongest contributing stats.
type Recommendation struct {
	Slug    string   `json:"slug"`
	Score   int      `json:"score"`
	Because []string `json:"because"`
}

// Recommend ranks items by cosine similarity to the player's stats (spec §8.5).
func Recommend(s Stats, items []Item, limit int) []Recommendation {
	sn := 0.0
	for i := range s {
		sn += s[i] * s[i]
	}
	sn = math.Sqrt(sn)
	if sn == 0 {
		return []Recommendation{}
	}
	out := make([]Recommendation, 0, len(items))
	for _, it := range items {
		vn, dot := 0.0, 0.0
		type c struct {
			i int
			v float64
		}
		contrib := make([]c, 0, 8)
		for i := range Codes {
			w := it.Vector.at(i)
			vn += w * w
			v := s[i] * w
			dot += v
			contrib = append(contrib, c{i, v})
		}
		vn = math.Sqrt(vn)
		score := 0
		if vn != 0 {
			score = int(roundHalfUp(dot / (sn * vn) * 100))
		}
		sort.SliceStable(contrib, func(a, b int) bool {
			if contrib[a].v != contrib[b].v {
				return contrib[a].v > contrib[b].v
			}
			return contrib[a].i < contrib[b].i
		})
		because := []string{}
		for _, x := range contrib {
			if x.v > 0 && len(because) < 2 {
				because = append(because, Codes[x.i])
			}
		}
		out = append(out, Recommendation{Slug: it.Slug, Score: score, Because: because})
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Score != out[b].Score {
			return out[a].Score > out[b].Score
		}
		return out[a].Slug < out[b].Slug
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
