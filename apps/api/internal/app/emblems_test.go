package app

import (
	"os"
	"regexp"
	"testing"
)

// The web app offers exactly the emblems the server accepts.
func TestEmblemsMatchWebApp(t *testing.T) {
	src, err := os.ReadFile("../../../web/src/lib/game/emblems.ts")
	if err != nil {
		t.Fatal(err)
	}
	web := map[string]bool{}
	for _, m := range regexp.MustCompile(`icon: '([a-z_]+)'`).FindAllStringSubmatch(string(src), -1) {
		web[m[1]] = true
	}
	if len(web) != len(allowedEmblems) {
		t.Fatalf("web app has %d emblems, server allows %d", len(web), len(allowedEmblems))
	}
	for e := range web {
		if !allowedEmblems[e] {
			t.Errorf("emblem %q is offered by the web app but rejected by the server", e)
		}
	}
}
