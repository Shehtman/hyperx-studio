package engine

import (
	"os"
	"testing"

	"hyperx-studio/internal/effects"
)

// Состояние обязано переживать перезапуск, в том числе грубый.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	st := DefaultState()
	st.Effect = "fire"
	st.Overlay = "ripple"
	st.FPS = 45
	st.Params.Color1 = effects.RGB{R: 7, G: 200, B: 33}
	st.Params.ReactFade = 2.25
	st.PerKey["33"] = "#00ff88"
	st.Selection = []int{1, 2, 3}

	if err := Save(st); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ConfigPath() + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("остался временный файл после записи")
	}

	got := Load()
	if got.Effect != "fire" || got.Overlay != "ripple" || got.FPS != 45 {
		t.Fatalf("не восстановилось: %+v", got)
	}
	if got.Params.Color1 != (effects.RGB{R: 7, G: 200, B: 33}) {
		t.Fatalf("цвет не восстановился: %v", got.Params.Color1)
	}
	if got.Params.ReactFade != 2.25 {
		t.Fatalf("затухание не восстановилось: %v", got.Params.ReactFade)
	}
	if got.PerKey["33"] != "#00ff88" || len(got.Selection) != 3 {
		t.Fatalf("ручные цвета или выделение потеряны: %+v", got)
	}
}

// Отсутствие файла — это первый запуск, а не ошибка.
func TestLoadMissingGivesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got := Load()
	if got.FPS != DefaultState().FPS || got.Effect != DefaultState().Effect {
		t.Fatalf("значения по умолчанию не подставились: %+v", got)
	}
}

// Битый конфиг не должен ронять программу.
func TestLoadCorruptGivesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	os.MkdirAll(ConfigDir(), 0o755)
	os.WriteFile(ConfigPath(), []byte("{это не json"), 0o644)
	if got := Load(); got.Effect != DefaultState().Effect {
		t.Fatalf("битый конфиг не заменён умолчаниями: %+v", got)
	}
}

func TestAutostartToggle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if AutostartEnabled() {
		t.Fatal("автозапуск включён на пустом каталоге")
	}
	if err := ApplyAutostart(true); err != nil {
		t.Fatal(err)
	}
	if !AutostartEnabled() {
		t.Fatal("ярлык автозапуска не создан")
	}
	body, _ := os.ReadFile(autostartPath())
	if len(body) == 0 || !contains(string(body), "Exec=") {
		t.Fatalf("ярлык неполон: %s", body)
	}
	if err := ApplyAutostart(false); err != nil {
		t.Fatal(err)
	}
	if AutostartEnabled() {
		t.Fatal("ярлык не удалён")
	}
	// повторное выключение не должно быть ошибкой
	if err := ApplyAutostart(false); err != nil {
		t.Fatalf("повторное выключение вернуло ошибку: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
