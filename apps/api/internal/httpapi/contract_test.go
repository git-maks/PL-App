package httpapi_test

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/git-maks/pl-app/apps/api/internal/config"
	"github.com/git-maks/pl-app/apps/api/internal/httpapi"
)

// TestRoutesMatchOpenAPI keeps the Go router and packages/contracts/openapi.yaml in sync.
func TestRoutesMatchOpenAPI(t *testing.T) {
	f, err := os.Open("../../../../packages/contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pathRe := regexp.MustCompile(`^  (/\S+):\s*$`)
	methodRe := regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	spec := map[string]bool{}
	inPaths, current := false, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "paths:":
			inPaths = true
		case inPaths && len(line) > 0 && line[0] != ' ' && line[0] != '#':
			inPaths = false
		case inPaths && pathRe.MatchString(line):
			current = pathRe.FindStringSubmatch(line)[1]
		case inPaths && current != "" && methodRe.MatchString(line):
			spec[strings.ToUpper(methodRe.FindStringSubmatch(line)[1])+" "+current] = true
		}
	}
	impl := map[string]bool{}
	for _, r := range httpapi.New(nil, nil, testConfig(), nil, nil).Routes() {
		impl[r] = true
	}
	var missing, undocumented []string
	for r := range spec {
		if !impl[r] {
			missing = append(missing, r)
		}
	}
	for r := range impl {
		if !spec[r] {
			undocumented = append(undocumented, r)
		}
	}
	sort.Strings(missing)
	sort.Strings(undocumented)
	if len(missing) > 0 {
		t.Errorf("in openapi.yaml but not implemented: %v", missing)
	}
	if len(undocumented) > 0 {
		t.Errorf("implemented but missing from openapi.yaml: %v", undocumented)
	}
	if len(spec) < 40 {
		t.Fatalf("parsed only %d operations – parser broken?", len(spec))
	}
}

func testConfig() config.Config { return config.Config{BaseURL: "http://uniquest.test"} }
