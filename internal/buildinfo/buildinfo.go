package buildinfo

// Populated at build time via ldflags. Defaults allow local `go build` to work.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)
