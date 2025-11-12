package buildinfo

import "runtime/debug"

var (
	Version   = "dev"
	Commit    = ""
	BuildTime = ""
)

// ReadBuildInfo tries to read build info provided by Go toolchain.
func ReadBuildInfo() {
	if info, ok := debug.ReadBuildInfo(); ok && info != nil {
		Version = info.Main.Version
	}
}
