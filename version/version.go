package version

import (
	"fmt"
	"runtime"
)

// Values pre-populated in build using linker flags
var (
	// Version
	Version   = "dev"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("Version: %s, built %s, go version: %s", Version, BuildDate, runtime.Version())
}
