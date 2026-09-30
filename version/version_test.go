package version

import "testing"

func TestString(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })

	Version, Commit, Date = "0.1.0", "abc1234", "2026-09-30T00:00:00Z"
	if got, want := String(), "0.1.0 (abc1234) 2026-09-30T00:00:00Z"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
