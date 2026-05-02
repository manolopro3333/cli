package main

import (
	"time"

	"github.com/manolopro3333/cli/src/utils"
)

type loopConfig struct {
	Update   updateConfig
	Interval time.Duration
}

func runLoop(cfg loopConfig) {
	nextRun := time.Now()
	for {
		now := time.Now()
		if now.Before(nextRun) {
			time.Sleep(nextRun.Sub(now))
		}

		if runCycle(cfg.Update) {
			return
		}

		nextRun = nextRun.Add(cfg.Interval)
	}
}

func runCycle(updateCfg updateConfig) bool {
	updated, err := checkAndUpdate(updateCfg)
	if err != nil {
		_ = runSpicetifyUpdate()
	}
	if updated {
		return true
	}

	state, _ := loadState()

	paths, err := resolvePaths()
	if err != nil {
		return false
	}

	current := detectCurrent(paths)
	spotifyChanged := hasSpotifyChanged(state, current)
	patchMissing := !current.PatchApplied
	needsApply := spotifyChanged || patchMissing || state.PendingApply

	if needsApply {
		if utils.IsAutoApplyPaused() {
			state.PendingApply = false
		} else if isSpotifyRunning() {
			state.PendingApply = true
		} else if err := runSpicetifyApply(); err != nil {
			state.PendingApply = true
		} else {
			state.PendingApply = false
		}
	}

	if current.SpotifyVersion != "" {
		state.SpotifyVersion = current.SpotifyVersion
	}
	if current.AppsModTimeUnix != 0 {
		state.AppsModTimeUnix = current.AppsModTimeUnix
	}
	_ = saveState(state)

	return false
}
