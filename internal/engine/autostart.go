package engine

import (
	"fmt"
	"os"
	"path/filepath"
)

func autostartPath() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "autostart", "hyperx-studio.desktop")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart", "hyperx-studio.desktop")
}

// ApplyAutostart создаёт или удаляет ярлык автозапуска.
//
// Подсветку нельзя записать в память клавиатуры, поэтому после перезагрузки
// она возвращается к заводской. Автозапуск — единственный способ вернуть свою.
func ApplyAutostart(on bool) error {
	path := autostartPath()
	if !on {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=HyperX Studio
Comment=Подсветка клавиатуры HyperX
Exec=%s --no-window
Icon=input-keyboard
Terminal=false
X-GNOME-Autostart-enabled=true
`, exe)
	return os.WriteFile(path, []byte(body), 0o644)
}

func AutostartEnabled() bool {
	_, err := os.Stat(autostartPath())
	return err == nil
}
