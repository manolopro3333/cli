package main

import (
	"bytes"
	"io"
	"os/exec"
	"runtime"
)

func runSpicetifyApply() error {
	spicetifyPath, err := exec.LookPath("spicetify")
	if err != nil {
		return err
	}

	cmd := exec.Command(spicetifyPath, "-q", "backup", "apply")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func runSpicetifyUpdate() error {
	spicetifyPath, err := exec.LookPath("spicetify")
	if err != nil {
		return err
	}

	cmd := exec.Command(spicetifyPath, "-q", "update")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func isSpotifyRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq spotify.exe")
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return bytes.Contains(bytes.ToLower(out), []byte("spotify.exe"))
	case "linux":
		err := exec.Command("pgrep", "-x", "spotify").Run()
		return err == nil
	case "darwin":
		err := exec.Command("pgrep", "-x", "Spotify").Run()
		return err == nil
	default:
		return false
	}
}
