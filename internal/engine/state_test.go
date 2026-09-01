package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/keyboard"
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

// Настройки, которых не было в прежних версиях, приходят из старого файла
// нулями. Ноль насыщенности и чувствительности — не то, что имел в виду
// пользователь: интерфейс показал бы ноль, а эффект рисовал бы по-своему.
func TestLoadFillsSettingsMissingFromOlderConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "hyperx-studio"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Файл в том виде, в каком его писала версия без этих полей.
	old := `{"effect":"colorwave","fps":60,"lang":"ru","params":{"speed":1,"brightness":1}}`
	if err := os.WriteFile(filepath.Join(dir, "hyperx-studio", "config.json"),
		[]byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	st := Load()
	if st.Params.Saturation != 1 {
		t.Errorf("насыщенность %v, ожидалась 1", st.Params.Saturation)
	}
	if st.Params.Sensitivity != 1 {
		t.Errorf("чувствительность %v, ожидалась 1", st.Params.Sensitivity)
	}
	if st.Effect != "colorwave" {
		t.Errorf("прежние настройки потерялись: эффект %q", st.Effect)
	}
}

// Погашение обязано дойти и до предпросмотра. Иначе цикл рендера встаёт,
// последний кадр остаётся висеть в окне, и программа показывает горящую
// клавиатуру, когда та уже тёмная.
func TestBlackoutDarkensThePreview(t *testing.T) {
	e := &Engine{
		st:    Load(),
		frame: make([]keyboard.RGB, keyboard.LEDCount),
		stop:  make(chan struct{}),
		subs:  map[chan []keyboard.RGB]struct{}{},
		t0:    time.Now(),
	}
	e.rebuild()
	e.effect = effects.New("static")

	// кадр, будто только что нарисованный ярким эффектом
	lit := make([]keyboard.RGB, keyboard.LEDCount)
	for i := range lit {
		lit[i] = keyboard.RGB{R: 200, G: 40, B: 90}
	}
	e.mu.Lock()
	e.frame = lit
	e.mu.Unlock()

	ch := e.Subscribe()
	defer e.Unsubscribe(ch)

	// устройства нет — ошибку ждём, но предпросмотр всё равно обязан погаснуть
	if err := e.Blackout(); err == nil {
		t.Fatal("без устройства Blackout должен вернуть ошибку")
	}

	select {
	case got := <-ch:
		for i, c := range got {
			if c != (keyboard.RGB{}) {
				t.Fatalf("в предпросмотр ушёл незачернённый кадр: светодиод %d = %+v", i, c)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("после погашения предпросмотр не получил ни одного кадра")
	}

	for i, c := range e.Frame() {
		if c != (keyboard.RGB{}) {
			t.Fatalf("сохранённый кадр остался цветным: светодиод %d = %+v", i, c)
		}
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	st := DefaultState()
	// Пути к несуществующему узлу довольно, чтобы цикл не нашёл настоящую
	// клавиатуру: иначе тест гасит подсветку тому, кто его запустил.
	st.Device = filepath.Join(t.TempDir(), "not-a-keyboard")

	e := &Engine{
		st:    st,
		frame: make([]keyboard.RGB, keyboard.LEDCount),
		stop:  make(chan struct{}),
		subs:  map[chan []keyboard.RGB]struct{}{},
		t0:    time.Now(),
	}
	e.rebuild()
	e.effect = effects.New(e.st.Effect)
	return e
}

// Погашение и пауза не должны заставлять цикл замолчать.
//
// Клавиатура держит прямой режим, только пока ей шлют кадры: перестанешь —
// и через несколько секунд она включит собственный эффект из прошивки.
// Раньше цикл на паузе просто засыпал, и погашенная подсветка загоралась
// заново сама по себе.
func TestPausedLoopKeepsSendingFrames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stop  func(*Engine)
		black bool
	}{
		{"погашение", func(e *Engine) { e.Blackout() }, true},
		{"пауза", func(e *Engine) { e.SetPaused(true) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEngine(t)
			go e.Run()
			defer e.Close()

			ch := e.Subscribe()
			defer e.Unsubscribe(ch)

			tc.stop(e)

			seen := 0
			deadline := time.After(600 * time.Millisecond)
			for {
				select {
				case frame := <-ch:
					seen++
					if !tc.black {
						continue
					}
					for i, c := range frame {
						if c != (keyboard.RGB{}) {
							t.Fatalf("после погашения пришёл цветной кадр: светодиод %d = %+v", i, c)
						}
					}
				case <-deadline:
					// на 60 кадрах в секунду за 0,6 с их должны быть десятки
					if seen < 5 {
						t.Fatalf("цикл замолчал: за 0,6 с всего %d кадров", seen)
					}
					return
				}
			}
		})
	}
}
