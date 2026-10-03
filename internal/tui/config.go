package tui

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// ConfigLoadedMsg reports what the keys of the configuration are set to, by name and without the
// keys that are not set, for the configuration dialog to open on.
type ConfigLoadedMsg struct {
	values map[string]string
}

// loadConfigCmd returns a command that loads the configuration from store.
func loadConfigCmd(store Store) tea.Cmd {
	return func() tea.Msg {
		values, _ := loadConfig(store)
		return ConfigLoadedMsg{values: values}
	}
}

// ConfigSavedMsg reports that the configuration was changed, with what its keys are set to since,
// as ConfigLoadedMsg has them.
type ConfigSavedMsg struct {
	values map[string]string
}

// setConfigCmd returns a command that sets key of the configuration in store to value and then
// loads the configuration, so that the dialog shows what the store has rather than what it asked
// for.
func setConfigCmd(store Store, key, value string) tea.Cmd {
	return func() tea.Msg {
		_ = store.SetConfig(key, value)
		values, _ := loadConfig(store)
		return ConfigSavedMsg{values: values}
	}
}

// loadConfig returns what every key of the configuration is set to in store, by the name of the
// key, as text the way the store keeps it. A key that is not set is not in it.
func loadConfig(store Store) (map[string]string, error) {
	values := make(map[string]string)
	for _, key := range portfolio.ConfigKeys {
		value, err := store.GetConfig(key.Name)
		switch {
		case errors.Is(err, portfolio.ErrNotSet):
			continue
		case err != nil:
			return nil, fmt.Errorf("get %s: %w", key.Name, err)
		}

		values[key.Name] = value
	}

	return values, nil
}
