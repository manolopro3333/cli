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
			return
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

func heal(spicetifyVersion, prefsPath, backupPath, backupVersion string) bool {
	cmd.InitConfig(true)
	cmd.InitPaths()
	cmd.InitSetting()

	// Try to update Spicetify CLI if new version is available.
	updated := cmd.Update(spicetifyVersion)

	// If patch is not applied, reapply it.
	cfg := utils.ParseConfig(cmd.GetConfigPath())
	setting := cfg.GetSection("Setting")
	spotifyPath := utils.ReplaceEnvVarsInString(setting.Key("spotify_path").String())
	appsPath := filepath.Join(spotifyPath, "Apps")

	if updated {
		// After updating the CLI, ensure Spotify is restored to stock,
		// re-backup (preprocess) and re-apply so preprocessing and assets
		// are regenerated for the new version.
		cmd.SpotifyKill()
		cmd.Restore()
		if err := utils.SetAutoApplyPaused(false); err != nil {
			utils.PrintWarning("Failed to clear auto-apply pause after restore: " + err.Error())
		}
		cmd.Backup(spicetifyVersion, true)
		cmd.Apply(spicetifyVersion)
		utils.PrintSuccess("Reapplied Spicetify after CLI update")
		utils.PrintSuccess("Update check completed")
		return true
	}

	backStat := backupstatus.Get(prefsPath, backupPath, backupVersion)
	if !backStat.IsBackuped() {
		if utils.IsAutoApplyPaused() {
			utils.PrintInfo("Auto-apply is paused because user restored Spotify.")
			return false
		}

		spotStat := spotifystatus.Get(appsPath)
		if backStat.IsOutdated() {
			utils.PrintInfo("Backup is outdated; rebuilding for the new Spotify version...")
			if spotStat.IsBackupable() {
				rebuildBackupAndApply(spicetifyVersion)
				return true
			}
			restoreBackupAndApply(spicetifyVersion)
			return true
		}

		if backStat.IsEmpty() {
			if spotStat.IsBackupable() {
				utils.PrintInfo("Backup is missing; creating a new backup...")
				rebuildBackupAndApply(spicetifyVersion)
				return true
			}
		}

		utils.PrintWarning("Backup is not ready; will reapply on next cycle.")
		return false
	}

	if utils.IsAutoApplyPaused() {
		utils.PrintInfo("Auto-apply is paused because user restored Spotify.")
		return false
	}

	if !spotifystatus.Get(appsPath).IsApplied() {
		cmd.SpotifyKill()
		cmd.Apply(spicetifyVersion)
		utils.PrintSuccess("Reapplied Spicetify")
	}
	return false
}

func rebuildBackupAndApply(spicetifyVersion string) {
	cmd.SpotifyKill()
	cmd.Backup(spicetifyVersion, true)
	cmd.Apply(spicetifyVersion)
	utils.PrintSuccess("Rebuilt backup and reapplied Spicetify")
}

func restoreBackupAndApply(spicetifyVersion string) {
	cmd.SpotifyKill()
	cmd.Restore()
	if err := utils.SetAutoApplyPaused(false); err != nil {
		utils.PrintWarning("Failed to clear auto-apply pause after restore: " + err.Error())
	}
	cmd.Backup(spicetifyVersion, true)
	cmd.Apply(spicetifyVersion)
	utils.PrintSuccess("Restored Spotify, rebuilt backup, and reapplied Spicetify")
}
