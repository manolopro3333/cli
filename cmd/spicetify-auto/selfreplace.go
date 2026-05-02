package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const selfReplaceArg = "--self-replace"

func handleSelfReplace(args []string) bool {
	if len(args) == 0 || args[0] != selfReplaceArg {
		return false
	}

	targetPath := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--target" && i+1 < len(args) {
			targetPath = args[i+1]
			break
		}
	}

	if targetPath == "" {
		os.Exit(1)
	}

	if err := selfReplace(targetPath); err != nil {
		os.Exit(1)
	}

	return true
}

func startSelfReplace(downloadPath, targetPath string) error {
	cmd := exec.Command(downloadPath, selfReplaceArg, "--target", targetPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

func selfReplace(targetPath string) error {
	sourcePath, err := currentExecutable()
	if err != nil {
		return err
	}

	if err := replaceWithRetries(sourcePath, targetPath, 60, 500*time.Millisecond); err != nil {
		return err
	}

	cmd := exec.Command(targetPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

func replaceWithRetries(sourcePath, targetPath string, attempts int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := replaceBinary(sourcePath, targetPath); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(delay)
	}
	if lastErr == nil {
		lastErr = errors.New("replace failed")
	}
	return lastErr
}

func replaceBinary(sourcePath, targetPath string) error {
	tmpPath := targetPath + ".new"
	if err := copyFile(sourcePath, tmpPath); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0755)
	}

	if runtime.GOOS == "windows" {
		_ = os.Remove(targetPath + ".old")
		_ = os.Rename(targetPath, targetPath+".old")
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
		_ = os.Remove(targetPath + ".old")
		return nil
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	return err
}

func currentExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		return resolved, nil
	}
	return exePath, nil
}
