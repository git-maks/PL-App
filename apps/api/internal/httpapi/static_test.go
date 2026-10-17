package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticHandler(t *testing.T) {
	s := &Server{static: fstest.MapFS{
		"index.html":                {Data: []byte("landing")},
		"200.html":                  {Data: []byte("spa")},
		"hero.html":                 {Data: []byte("hero")},
		"_app/immutable/entry/a.js": {Data: []byte("js")},
		"explore/kola/index.html":   {Data: []byte("kola")},
		"manifest.webmanifest":      {Data: []byte("{}")},
	}}
	h := s.staticHandler()
	cases := []struct {
		path, body string
		code       int
	}{
		{"/", "landing", 200},
		{"/hero", "hero", 200},
		{"/explore/kola", "kola", 200},
		{"/_app/immutable/entry/a.js", "js", 200},
		{"/q/1.0123456789abcdef0123456789abcdef.58723411.AbCdEf-_1234", "spa", 200},
		{"/q/s.0123456789abcdef0123456789abcdef.AbCdEf-_1234", "spa", 200},
		{"/t/ABC123", "spa", 200},
		{"/some/client/route", "spa", 200},
		{"/_app/immutable/missing.js", "", 404},
		{"/missing.png", "", 404},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.code {
			t.Errorf("%s: status %d, want %d", c.path, rec.Code, c.code)
			continue
		}
		if c.body != "" && rec.Body.String() != c.body {
			t.Errorf("%s: body %q, want %q", c.path, rec.Body.String(), c.body)
		}
	}
}
