package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ini/ini"
	"github.com/manolopro3333/cli/src/utils"
)

type spicetifyPaths struct {
	SpotifyPath string
	PrefsPath   string
	AppsPath    string
	AppDestPath string
	IsAppX      bool
}

type currentState struct {
	SpotifyVersion  string
	AppsModTimeUnix int64
	PatchApplied    bool
}

func resolvePaths() (spicetifyPaths, error) {
	configPath := filepath.Join(utils.GetSpicetifyFolder(), "config-xpui.ini")
	spotifyPath := ""
	prefsPath := ""

	cfg, err := ini.LoadSources(ini.LoadOptions{
		IgnoreContinuation:  true,
		IgnoreInlineComment: true,
	}, configPath)
	if err == nil {
		setting := cfg.Section("Setting")
		spotifyPath = utils.ReplaceEnvVarsInString(setting.Key("spotify_path").String())
		prefsPath = utils.ReplaceEnvVarsInString(setting.Key("prefs_path").String())
	}

	if spotifyPath == "" {
		spotifyPath = findSpotifyPath()
	}
	if prefsPath == "" {
		prefsPath = findPrefsPath()
	}

	if spotifyPath == "" || prefsPath == "" {
		return spicetifyPaths{}, errors.New("spotify paths not found")
	}

	isAppX := false
	if runtime.GOOS == "windows" {
		if strings.Contains(spotifyPath, "SpotifyAB.SpotifyMusic") || strings.Contains(prefsPath, "SpotifyAB.SpotifyMusic") {
			isAppX = true
		}
	}

	appsPath := filepath.Join(spotifyPath, "Apps")
	appDestPath := appsPath
	if isAppX {
		appDestPath = filepath.Join(utils.GetSpicetifyFolder(), "AppX")
	}

	return spicetifyPaths{
		SpotifyPath: spotifyPath,
		PrefsPath:   prefsPath,
		AppsPath:    appsPath,
		AppDestPath: appDestPath,
		IsAppX:      isAppX,
	}, nil
}

func detectCurrent(paths spicetifyPaths) currentState {
	cur := currentState{}

	if version, err := readSpotifyVersion(paths.PrefsPath); err == nil {
		cur.SpotifyVersion = version
	}
	if modTime, err := appsModTime(paths.AppsPath); err == nil {
		cur.AppsModTimeUnix = modTime
	}
	if applied, err := isPatchApplied(paths.AppDestPath); err == nil {
		cur.PatchApplied = applied
	}

	return cur
}

func hasSpotifyChanged(prev autoState, cur currentState) bool {
	if prev.SpotifyVersion != "" && cur.SpotifyVersion != "" && prev.SpotifyVersion != cur.SpotifyVersion {
		return true
	}
	if prev.AppsModTimeUnix != 0 && cur.AppsModTimeUnix != 0 && prev.AppsModTimeUnix != cur.AppsModTimeUnix {
		return true
	}
	return false
}

func readSpotifyVersion(prefsPath string) (string, error) {
	cfg, err := ini.Load(prefsPath)
	if err != nil {
		return "", err
	}
	section, err := cfg.GetSection("")
	if err != nil {
		return "", err
	}
	return section.Key("app.last-launched-version").String(), nil
}

func appsModTime(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.ModTime().Unix(), nil
}

func isPatchApplied(appsPath string) (bool, error) {
	entries, err := os.ReadDir(appsPath)
	if err != nil {
		return false, err
	}

	spaCount := 0
	dirCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			dirCount++
			continue
		}
		if strings.HasSuffix(entry.Name(), ".spa") {
			spaCount++
		}
	}

	return dirCount > 0 && spaCount == 0, nil
}

func findSpotifyPath() string {
	switch runtime.GOOS {
	case "windows":
		path := filepath.Join(os.Getenv("APPDATA"), "Spotify")
		if _, err := os.Stat(filepath.Join(path, "Spotify.exe")); err == nil {
			return path
		}
		return utils.WinXApp()
	default:
		return utils.FindAppPath()
	}
}

func findPrefsPath() string {
	switch runtime.GOOS {
	case "windows":
		path := filepath.Join(os.Getenv("APPDATA"), "Spotify", "prefs")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if utils.WinXApp() != "" {
			return utils.WinXPrefs()
		}
		return ""
	default:
		return utils.FindPrefFilePath()
	}
}
