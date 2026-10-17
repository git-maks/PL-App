// Package qr implements the station and team QR payload formats (spec §7.4). The web app's
// src/lib/qr/payload.ts implements the same formats; shared vectors live in
// packages/contracts/fixtures/qr.
//
//	rotating station code : {base}/q/1.{stationHex}.{window}.{mac}
//	static station poster : {base}/q/s.{stationHex}.{mac}
//	team invite           : {base}/t/{code}
//	manual station code   : 6 characters from ManualAlphabet
package qr

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PeriodSeconds       = 30
	ManualPeriodSeconds = 300
	MacLength           = 12
	ManualAlphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type Kind int

const (
	Invalid Kind = iota
	Rotating
	Static
	Team
	Manual
)

type Payload struct {
	Kind      Kind
	StationID string // dashed UUID
	Window    int64
	MAC       string
	Code      string // team or manual code
}

var (
	hex32  = regexp.MustCompile(`^[0-9a-f]{32}$`)
	macRe  = regexp.MustCompile(`^[A-Za-z0-9_-]{12}$`)
	teamRe = regexp.MustCompile(`^[A-Za-z0-9]{6,16}$`)
	qPath  = regexp.MustCompile(`/q/([^/]+)/?$`)
	tPath  = regexp.MustCompile(`/t/([^/]+)/?$`)
)

func HexToUUID(h string) string {
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func UUIDToHex(id string) string { return strings.ToLower(strings.ReplaceAll(id, "-", "")) }

// NormalizeManual returns the canonical 6-character manual code or "".
func NormalizeManual(in string) string {
	r := strings.NewReplacer(" ", "", "-", "", "\t", "")
	code := strings.ToUpper(r.Replace(strings.TrimSpace(in)))
	if len(code) != 6 {
		return ""
	}
	for _, ch := range code {
		if !strings.ContainsRune(ManualAlphabet, ch) {
			return ""
		}
	}
	return code
}

// Parse recognises our formats; anything else is Invalid.
func Parse(raw string) Payload {
	text := strings.TrimSpace(raw)
	if m := NormalizeManual(text); m != "" && !strings.Contains(text, "/") {
		return Payload{Kind: Manual, Code: m}
	}
	u, err := url.Parse(text)
	if err != nil || u.Scheme == "" {
		return Payload{}
	}
	if m := qPath.FindStringSubmatch(u.Path); m != nil {
		parts := strings.Split(m[1], ".")
		if parts[0] == "1" && len(parts) == 4 {
			w, err := strconv.ParseInt(parts[2], 10, 64)
			if err == nil && w > 0 && hex32.MatchString(parts[1]) && macRe.MatchString(parts[3]) {
				return Payload{Kind: Rotating, StationID: HexToUUID(parts[1]), Window: w, MAC: parts[3]}
			}
		}
		if parts[0] == "s" && len(parts) == 3 && hex32.MatchString(parts[1]) && macRe.MatchString(parts[2]) {
			return Payload{Kind: Static, StationID: HexToUUID(parts[1]), MAC: parts[2]}
		}
		return Payload{}
	}
	if m := tPath.FindStringSubmatch(u.Path); m != nil && teamRe.MatchString(m[1]) {
		return Payload{Kind: Team, Code: strings.ToUpper(m[1])}
	}
	return Payload{}
}

func mac(secret []byte, msg string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(msg))
	return h.Sum(nil)
}

func RotatingMAC(secret []byte, stationID string, window int64) string {
	return base64.RawURLEncoding.EncodeToString(mac(secret, "1|"+UUIDToHex(stationID)+"|"+strconv.FormatInt(window, 10)))[:MacLength]
}

func StaticMAC(secret []byte, stationID string) string {
	return base64.RawURLEncoding.EncodeToString(mac(secret, "s|"+UUIDToHex(stationID)))[:MacLength]
}

// ManualCode maps 30 bits of the HMAC onto ManualAlphabet.
func ManualCode(secret []byte, stationID string, window int64) string {
	d := mac(secret, "m|"+UUIDToHex(stationID)+"|"+strconv.FormatInt(window, 10))
	bits := binary.BigEndian.Uint32(d[:4])
	out := make([]byte, 6)
	for i := range out {
		out[i] = ManualAlphabet[bits&31]
		bits >>= 5
	}
	return string(out)
}

// Equal compares MACs in constant time.
func Equal(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }

func Window(t time.Time, period int64) int64 { return t.Unix() / period }

func RotatingURL(base, stationID string, window int64, m string) string {
	return base + "/q/1." + UUIDToHex(stationID) + "." + strconv.FormatInt(window, 10) + "." + m
}

func StaticURL(base, stationID, m string) string {
	return base + "/q/s." + UUIDToHex(stationID) + "." + m
}

func TeamURL(base, code string) string { return base + "/t/" + code }
