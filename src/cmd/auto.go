package cmd

import (
	"os"

	backupstatus "github.com/manolopro3333/cli/src/status/backup"
	spotifystatus "github.com/manolopro3333/cli/src/status/spotify"
	"github.com/manolopro3333/cli/src/utils"
)

// Auto checks Spotify state, re-backup and apply if needed, then launch
// Spotify client normally.
func Auto(spicetifyVersion string) {
	backupVersion := backupSection.Key("version").MustString("")
	spotStat := spotifystatus.Get(appPath)
	backStat := backupstatus.Get(prefsPath, backupFolder, backupVersion)

	if spotStat.IsBackupable() && (backStat.IsEmpty() || backStat.IsOutdated()) {
		Backup(spicetifyVersion, true)
		backupVersion := backupSection.Key("version").MustString("")
		backStat = backupstatus.Get(prefsPath, backupFolder, backupVersion)
	}

	if !spotStat.IsBackupable() && backStat.IsOutdated() {
		RestoreAuto()
		if err := utils.SetAutoApplyPaused(false); err != nil {
			utils.PrintWarning("Failed to clear auto-apply pause after restore: " + err.Error())
		}
		Backup(spicetifyVersion, true)
		backupVersion := backupSection.Key("version").MustString("")
		backStat = backupstatus.Get(prefsPath, backupFolder, backupVersion)
		spotStat = spotifystatus.Get(appPath)
	}

	if !backStat.IsBackuped() {
		os.Exit(1)
	}

	if isAppX {
		spotStat = spotifystatus.Get(appDestPath)
	}

	if !spotStat.IsApplied() && backStat.IsBackuped() {
		CheckStates()
		InitSetting()
		Apply(spicetifyVersion)
	}
}
