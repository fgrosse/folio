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
	err    error // why the configuration could not be loaded, in which case there are no values
}

// loadConfigCmd returns a command that loads the configuration from store.
func loadConfigCmd(store Store) tea.Cmd {
	return func() tea.Msg {
		values, err := loadConfig(store)
		return ConfigLoadedMsg{values: values, err: err}
	}
}

// ConfigSavedMsg reports that the configuration was changed, with what its keys are set to since,
// as ConfigLoadedMsg has them.
type ConfigSavedMsg struct {
	values map[string]string

	// err is why the configuration could not be changed, or not be loaded afterwards. If it is
	// the latter, there are no values.
	err error
}

// setConfigCmd returns a command that sets key of the configuration in store to value and then
// loads the configuration, so that the dialog shows what the store has rather than what it asked
// for.
func setConfigCmd(store Store, key, value string) tea.Cmd {
	return func() tea.Msg {
		if err := store.SetConfig(key, value); err != nil {
			return configSaved(store, fmt.Errorf("set %s: %w", key, err))
		}

		return configSaved(store, nil)
	}
}

// unsetConfigCmd returns a command that takes the value of key of the configuration in store back
// and then loads the configuration, as setConfigCmd does.
func unsetConfigCmd(store Store, key string) tea.Cmd {
	return func() tea.Msg {
		if err := store.UnsetConfig(key); err != nil {
			return configSaved(store, fmt.Errorf("unset %s: %w", key, err))
		}

		return configSaved(store, nil)
	}
}

// configSaved loads the configuration from store and reports it together with err, which is why
// it could not be changed, if it could not. The configuration is loaded even then: the dialog keeps
// showing what the store has, which is what it had before.
func configSaved(store Store, err error) ConfigSavedMsg {
	values, loadErr := loadConfig(store)
	if err == nil {
		err = loadErr
	}

	return ConfigSavedMsg{values: values, err: err}
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
