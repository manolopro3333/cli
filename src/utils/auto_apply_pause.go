package utils

import (
	"os"
	"path/filepath"
)

func autoApplyPausePath() string {
	return filepath.Join(GetStateFolder("Auto"), "pause-auto-apply")
}

func SetAutoApplyPaused(paused bool) error {
	path := autoApplyPausePath()
	if paused {
		return os.WriteFile(path, []byte("1"), 0600)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsAutoApplyPaused() bool {
	_, err := os.Stat(autoApplyPausePath())
	return err == nil
}
