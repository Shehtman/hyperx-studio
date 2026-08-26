// Package engine связывает раскладку, эффекты, ввод и устройство.
package engine

import (
	"encoding/json"
	"os"
	"path/filepath"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/i18n"
)

// State — всё, что пользователь настроил. Сохраняется на диск.
type State struct {
	Effect    string            `json:"effect"`
	Overlay   string            `json:"overlay"`
	Variant   string            `json:"variant"`
	FPS       int               `json:"fps"`
	MaskSel   bool              `json:"maskSelection"`
	Params    effects.Params    `json:"params"`
	PerKey    map[string]string `json:"perKey"` // индекс -> "#rrggbb"
	Selection []int             `json:"selection"`
	Autostart bool              `json:"autostart"`
	Lang      string            `json:"lang"`
}

func DefaultState() State {
	return State{
		Effect: "rainbow", Overlay: "", Variant: "ansi", FPS: 30,
		Params: effects.DefaultParams(),
		PerKey: map[string]string{},
		Lang:   i18n.EN,
	}
}

func ConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "hyperx-studio")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hyperx-studio")
}

func ConfigPath() string { return filepath.Join(ConfigDir(), "config.json") }

// Load читает сохранённое состояние. Если файла нет — вернёт значения по
// умолчанию, а не ошибку: первый запуск это норма.
func Load() State {
	st := DefaultState()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return st
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return DefaultState()
	}
	if st.PerKey == nil {
		st.PerKey = map[string]string{}
	}
	if st.FPS <= 0 {
		st.FPS = 30
	}
	if !i18n.Valid(st.Lang) {
		st.Lang = i18n.EN
	}
	return st
}

// Save пишет состояние атомарно: сначала во временный файл, потом
// переименование. Сбой посреди записи не оставит обрезанный конфиг.
func Save(st State) error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", " ")
	if err != nil {
		return err
	}
	tmp := ConfigPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ConfigPath())
}
