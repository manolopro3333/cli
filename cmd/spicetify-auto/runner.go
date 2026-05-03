package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
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

// runSpicetifyUpdateWithTimeout runs `spicetify -q update` with a timeout
// and forcefully terminates any lingering spicetify process after timeout.
func runSpicetifyUpdateWithTimeout(timeout time.Duration) error {
	spicetifyPath, err := exec.LookPath("spicetify")
	if err != nil {
		return err
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, spicetifyPath, "-q", "update")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	err = cmd.Run()
	
	// Always terminate lingering processes after running update,
	// regardless of success or timeout, to ensure the UI closes.
	_ = terminateUpdateProcesses()
	
	return err
}

// runSpicetifyUpdate kept for compatibility; uses a 5s timeout.
func runSpicetifyUpdate() error {
	return runSpicetifyUpdateWithTimeout(5 * time.Second)
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

// terminateSpicetifyProcesses forcefully kills any lingering spicetify
// processes (best-effort). Uses escalating approaches: soft kill first,
// then forceful kill (/F on Windows, -9 on Unix) to ensure closure.
func terminateSpicetifyProcesses() error {
	switch runtime.GOOS {
	case "windows":
		// Kill both current and self-replace renamed processes.
		_ = killWindowsImage("spicetify.exe")
		_ = killWindowsImage("spicetify.exe.old")
	case "linux", "darwin":
		// First attempt: graceful termination
		_ = exec.Command("pkill", "-f", "spicetify").Run()
		// Second attempt: forceful termination (kill -9)
		_ = exec.Command("pkill", "-9", "-f", "spicetify").Run()
	default:
		// unsupported OS; nothing to do
	}
	return nil
}

func killWindowsImage(image string) error {
	_ = exec.Command("taskkill", "/IM", image, "/T").Run()
	_ = exec.Command("taskkill", "/IM", image, "/T", "/F").Run()
	return nil
}

// terminateUpdateProcesses kills processes spawned during spicetify update.
func terminateUpdateProcesses() error {
	if runtime.GOOS == "windows" {
		_ = killWindowsImage("spicetify.exe")
		_ = killWindowsImage("cmd.exe")
		_ = killWindowsImage("powershell.exe")
	}
	return nil
}
