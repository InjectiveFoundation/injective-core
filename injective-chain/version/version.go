package version

import (
	"fmt"
	"runtime"
)

// AppName represents the application name as the 'user agent' on the larger Ethereum network.
const AppName = "injective"

// Version contains the application semantic version.
const Version = "1.6.0"

// GitCommit contains the git SHA1 short hash set by build flags.
var GitCommit = ""

// ClientVersion returns the full version string for identification on the larger Ethereum network.
func ClientVersion() string {
	return fmt.Sprintf("%s/%s+%s/%s/%s", AppName, Version, GitCommit, runtime.GOOS, runtime.Version())
}
