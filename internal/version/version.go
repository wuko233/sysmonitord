package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	GitCommit = "none"
	BuildTime = "unknown"
)

func Info() string {
	return fmt.Sprintf("Sysmonitord %s\nGit Commit: %s\nBuild Time: %s\nGo Version: %s",
		Version, GitCommit, BuildTime, runtime.Version())
}
