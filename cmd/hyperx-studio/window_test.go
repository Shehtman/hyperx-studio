package main

import (
	"os"
	"strings"
	"testing"
)

func buildScript(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../build-deb.sh")
	if err != nil {
		t.Fatalf("не прочитать build-deb.sh: %v", err)
	}
	return string(data)
}

// Под Wayland оболочка связывает окно с записью .desktop по имени программы.
// Разойдись appID с именем файла — и окно снова останется без подписи и без
// иконки, ровно как было с браузером.
func TestAppIDMatchesDesktopEntry(t *testing.T) {
	s := buildScript(t)
	for _, want := range []string{
		"/applications/" + appID + ".desktop",
		"Icon=" + appID,
		"StartupWMClass=" + appID,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("в build-deb.sh нет %q — окно останется без имени или иконки", want)
		}
	}
}

// Окно ищется рядом с самой программой и в PATH под этим именем. Если пакет
// кладёт его под другим, интерфейс молча уедет обратно в браузер.
func TestPackageInstallsTheWindowHelper(t *testing.T) {
	s := buildScript(t)
	if !strings.Contains(s, "/usr/bin/"+helperName) {
		t.Errorf("build-deb.sh не ставит %q в /usr/bin", helperName)
	}
}

// Запись .desktop для окна Chrome больше не нужна: окно своё. Пока она в
// пакете, оболочка может привязать к ней чужое окно браузера.
func TestChromeShimIsGone(t *testing.T) {
	if s := buildScript(t); strings.Contains(s, "chrome-127.0.0.1") {
		t.Error("в пакете осталась запись .desktop для окна Chrome")
	}
}
