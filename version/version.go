// Package version holds the build identity, set by GoReleaser through -ldflags.
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns "version (commit) date".
func String() string {
	return Version + " (" + Commit + ") " + Date
}
