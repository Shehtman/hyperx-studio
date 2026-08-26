package main

import (
	"fmt"
	"os"
	"os/exec"

	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/i18n"
)

// openWindow показывает интерфейс отдельным окном.
//
// Браузеры на основе Chromium умеют режим приложения: окно без вкладок,
// адресной строки и кнопок. Вместе с --class его подхватывает оболочка
// рабочего стола и рисует нашу иконку, а не браузерную.
func openWindow(url string) {
	size := "--window-size=1180,760"

	for _, name := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
		"brave-browser", "microsoft-edge", "vivaldi",
	} {
		bin, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, "--app="+url, "--class=hyperx-studio", size)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Start(); err == nil {
			go cmd.Wait() // не оставляем зомби; закрытие окна нас не останавливает
			return
		}
	}

	// Firefox отдельного окна приложения не умеет — открываем как получится.
	if bin, err := exec.LookPath("firefox"); err == nil {
		cmd := exec.Command(bin, "--new-window", url)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
	}

	if err := exec.Command("xdg-open", url).Start(); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T(engine.Load().Lang, "app.openManually", url))
	}
}
