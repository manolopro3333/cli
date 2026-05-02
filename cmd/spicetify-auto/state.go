package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/manolopro3333/cli/src/utils"
)

type autoState struct {
	SpotifyVersion  string `json:"spotify_version"`
	AppsModTimeUnix int64  `json:"apps_mod_time_unix"`
	PendingApply    bool   `json:"pending_apply"`
}

func statePath() string {
	return filepath.Join(utils.GetStateFolder("Auto"), "state.json")
}

func loadState() (autoState, error) {
	path := statePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return autoState{}, nil
		}
		return autoState{}, err
	}

	var state autoState
	if err := json.Unmarshal(data, &state); err != nil {
		return autoState{}, err
	}

	return state, nil
}

func saveState(state autoState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0600)
}
