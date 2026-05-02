package main

import (
	"os"
	"time"
)

var version string

const (
	repoOwner  = "manolopro3333"
	repoName   = "cli"
	binaryName = "spicetify"
)

func main() {
	if version == "" {
		version = "dev"
	}

	if handleSelfReplace(os.Args[1:]) {
		return
	}

	runLoop(loopConfig{
		Update: updateConfig{
			RepoOwner:      repoOwner,
			RepoName:       repoName,
			BinaryName:     binaryName,
			CurrentVersion: version,
		},
		Interval: 6 * time.Hour,
	})
}
