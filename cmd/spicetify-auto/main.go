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

	// Asegura que cualquier proceso spicetify lingering se termina
	// antes de dormir, para evitar que la ventana se quede colgada.
	terminateSpicetifyProcesses()

	// Espera unos segundos antes de salir para permitir que cualquier
	// proceso hijo o mensaje final se complete y la ventana se cierre.
	time.Sleep(5 * time.Second)
}
