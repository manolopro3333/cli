package service

import (
	"path/filepath"
	"time"

	"github.com/manolopro3333/cli/src/cmd"
	backupstatus "github.com/manolopro3333/cli/src/status/backup"
	spotifystatus "github.com/manolopro3333/cli/src/status/spotify"
	"github.com/manolopro3333/cli/src/utils"
)

// Start runs the background service loop which monitors Spotify and reapplies Spicetify when needed.
func Start(spicetifyVersion string) {
	cmd.InitConfig(true)
	cmd.InitPaths()
	cmd.InitSetting()

	cfg := utils.ParseConfig(cmd.GetConfigPath())
	setting := cfg.GetSection("Setting")
	serviceSection := cfg.GetSection("Service")

	if !serviceSection.Key("auto_update").MustBool(true) {
		utils.PrintInfo("Service is disabled in config (Service.auto_update=0).")
		return
	}

	spotifyPath := utils.ReplaceEnvVarsInString(setting.Key("spotify_path").String())
	prefsPath := utils.ReplaceEnvVarsInString(setting.Key("prefs_path").String())
	appsPath := filepath.Join(spotifyPath, "Apps")
	backupPath := utils.GetStateFolder("Backup")
	backupVersion := cfg.GetSection("Backup").Key("version").MustString("")
	interval := time.Duration(serviceSection.Key("interval_seconds").MustInt(60)) * time.Second

	// Initialize with old timestamp so first iteration triggers update check on startup.
	lastState := serviceState{
		applied:         spotifystatus.Get(appsPath).IsApplied(),
		lastUpdateCheck: time.Now().Unix() - 6*3600 - 1,
	}

	for {
		cfg = utils.ParseConfig(cmd.GetConfigPath())
		serviceSection = cfg.GetSection("Service")
		if !serviceSection.Key("auto_update").MustBool(true) {
			utils.PrintInfo("Service disabled, stopping background monitor.")
			return
		}

		setting = cfg.GetSection("Setting")
		spotifyPath = utils.ReplaceEnvVarsInString(setting.Key("spotify_path").String())
		prefsPath = utils.ReplaceEnvVarsInString(setting.Key("prefs_path").String())
		appsPath = filepath.Join(spotifyPath, "Apps")
		backupPath = utils.GetStateFolder("Backup")
		backupVersion = cfg.GetSection("Backup").Key("version").MustString("")
		interval = time.Duration(serviceSection.Key("interval_seconds").MustInt(60)) * time.Second

		currentState := snapshotState(appsPath)
		if currentState == lastState {
			continue
		}

		if needsHeal(lastState, currentState) {
			utils.PrintInfo("Checking for Spicetify updates or reapplying patch...")
			heal(spicetifyVersion, prefsPath, backupPath, backupVersion)
			lastState = snapshotState(appsPath)
			time.Sleep(interval)
			continue
		}

		lastState = currentState
		time.Sleep(interval)
	}
}

type serviceState struct {
	applied         bool
	lastUpdateCheck int64
}

func snapshotState(appsPath string) serviceState {
	state := serviceState{
		applied:         spotifystatus.Get(appsPath).IsApplied(),
		lastUpdateCheck: time.Now().Unix(),
	}
	return state
}

func needsHeal(previous, current serviceState) bool {
	// Check if patch was lost.
	if !current.applied {
		return true
	}

	// Check if enough time has passed since last update check (6 hours).
	if previous.lastUpdateCheck > 0 && (current.lastUpdateCheck-previous.lastUpdateCheck) > 6*3600 {
		return true
	}

	return false
}

func heal(spicetifyVersion, prefsPath, backupPath, backupVersion string) {
	cmd.InitConfig(true)
	cmd.InitPaths()
	cmd.InitSetting()

	// Try to update Spicetify CLI if new version is available.
	cmd.Update(spicetifyVersion)
	utils.PrintSuccess("Update check completed")

	// If patch is not applied, reapply it.
	cfg := utils.ParseConfig(cmd.GetConfigPath())
	setting := cfg.GetSection("Setting")
	spotifyPath := utils.ReplaceEnvVarsInString(setting.Key("spotify_path").String())
	appsPath := filepath.Join(spotifyPath, "Apps")

	backStat := backupstatus.Get(prefsPath, backupPath, backupVersion)
	if !backStat.IsBackuped() {
		utils.PrintWarning("Backup is not ready; will reapply on next cycle.")
		return
	}

	if utils.IsAutoApplyPaused() {
		utils.PrintInfo("Auto-apply is paused because user restored Spotify.")
		return
	}

	if !spotifystatus.Get(appsPath).IsApplied() {
		cmd.SpotifyKill()
		cmd.Apply(spicetifyVersion)
		utils.PrintSuccess("Reapplied Spicetify")
	}
}
