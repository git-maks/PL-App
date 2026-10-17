package qr_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/git-maks/pl-app/apps/api/internal/qr"
)

type vectors struct {
	SecretHex string `json:"secretHex"`
	StationID string `json:"stationId"`
	Cases     []struct {
		Window int64  `json:"window"`
		MAC    string `json:"mac"`
		URL    string `json:"url"`
	} `json:"cases"`
	Static struct {
		MAC string `json:"mac"`
		URL string `json:"url"`
	} `json:"static"`
	Manual []struct {
		Window int64  `json:"window"`
		Code   string `json:"code"`
	} `json:"manual"`
}

// Same vectors as apps/web/src/lib/qr/payload.test.ts.
func TestSharedVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../../packages/contracts/fixtures/qr/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	secret, _ := hex.DecodeString(v.SecretHex)
	for _, c := range v.Cases {
		if got := qr.RotatingMAC(secret, v.StationID, c.Window); got != c.MAC {
			t.Errorf("rotating mac(%d) = %s, want %s", c.Window, got, c.MAC)
		}
	}
	if got := qr.RotatingURL("https://uniquest.example", v.StationID, v.Cases[0].Window, v.Cases[0].MAC); got != v.Cases[0].URL {
		t.Errorf("url = %s", got)
	}
	if got := qr.StaticMAC(secret, v.StationID); got != v.Static.MAC {
		t.Errorf("static mac = %s, want %s", got, v.Static.MAC)
	}
	for _, m := range v.Manual {
		if got := qr.ManualCode(secret, v.StationID, m.Window); got != m.Code {
			t.Errorf("manual(%d) = %s, want %s", m.Window, got, m.Code)
		}
	}
	p := qr.Parse(v.Cases[0].URL)
	if p.Kind != qr.Rotating || p.StationID != v.StationID || p.Window != v.Cases[0].Window || p.MAC != v.Cases[0].MAC {
		t.Errorf("parse rotating = %+v", p)
	}
	if qr.Parse(v.Static.URL).Kind != qr.Static {
		t.Error("parse static failed")
	}
}

func TestParse(t *testing.T) {
	cases := map[string]qr.Kind{
		" ukz-kcj ":                      qr.Manual,
		"ABCDE0":                         qr.Invalid,
		"https://x.pl/PL-App/t/ab12cd34": qr.Team,
		"https://example.com/menu":       qr.Invalid,
		"WIFI:S:eduroam;;":               qr.Invalid,
		"https://x.pl/q/1.zz.1.abc":      qr.Invalid,
	}
	for in, want := range cases {
		if got := qr.Parse(in).Kind; got != want {
			t.Errorf("Parse(%q) = %v, want %v", in, got, want)
		}
	}
	if c := qr.Parse("https://x.pl/t/ab12cd34").Code; c != "AB12CD34" {
		t.Errorf("team code = %s", c)
	}
}
