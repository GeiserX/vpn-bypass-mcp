package version

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// The release-binary example in the getting-started guide names a concrete version.
// It must be the one package.json and the next tag carry, or a copied command fetches
// an older binary.
func TestGettingStartedNamesThePackageVersion(t *testing.T) {
	raw, err := os.ReadFile("../package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct{ Version string }
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile("../docs/getting-started.md")
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`download/v([0-9][^/]*)/|vpn-bypass-mcp_([0-9][^_]*)_darwin`).FindAllStringSubmatch(string(doc), -1)
	if len(found) == 0 {
		t.Fatal("no versioned release link in docs/getting-started.md")
	}
	for _, m := range found {
		if got := m[1] + m[2]; got != pkg.Version {
			t.Errorf("docs/getting-started.md names %s in %q, package.json is %s", got, m[0], pkg.Version)
		}
	}
}
