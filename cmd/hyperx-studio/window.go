package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/i18n"
)

// Как называется окно и где оболочка ищет для него иконку. appID совпадает
// с именем файла /usr/share/applications/hyperx-studio.desktop — под Wayland
// оболочка сопоставляет окно с записью именно по нему.
const (
	windowTitle = "HyperX Studio"
	appID       = "hyperx-studio"
	windowW     = 1180
	windowH     = 760

	// helperName — программа, которая рисует окно. Отдельная, чтобы в самой
	// службе не было ни GTK, ни WebKit: она работает и без окна.
	helperName = "hyperx-studio-window"
)

// window заведует окном интерфейса.
//
// Окно живёт отдельным процессом, и это принципиально: закрытое окно не
// гасит подсветку. Служба продолжает работать, а окно возвращается по
// повторному запуску программы.
type window struct {
	url     string
	lang    string
	browser bool // показывать в браузере, даже если своё окно доступно

	mu      sync.Mutex
	proc    *os.Process
	closing bool
}

// show открывает окно. Уже открытое поднимает наверх, а не заводит второе.
func (w *window) show() {
	w.mu.Lock()
	if w.proc != nil {
		p := w.proc
		w.mu.Unlock()
		p.Signal(syscall.SIGUSR1)
		return
	}
	closing := w.closing
	w.mu.Unlock()
	if closing {
		return
	}

	if !w.browser && w.startHelper() {
		return
	}
	openBrowserWindow(w.url)
}

// startHelper запускает своё окно. false — окна нет, покажет браузер.
func (w *window) startHelper() bool {
	bin := helperPath()
	if bin == "" {
		return false
	}
	cmd := exec.Command(bin,
		"-title", windowTitle,
		"-app-id", appID,
		"-width", strconv.Itoa(windowW),
		"-height", strconv.Itoa(windowH),
		w.url)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return false
	}

	w.mu.Lock()
	w.proc = cmd.Process
	w.mu.Unlock()

	go func() {
		err := cmd.Wait()
		w.mu.Lock()
		w.proc = nil
		closing := w.closing
		w.mu.Unlock()
		// Окно закрыли — так и было задумано. А вот ненулевой код возврата
		// значит, что открыть его не удалось: нет графической сессии или
		// не хватает библиотек. Интерфейс тогда покажет браузер.
		if err != nil && !closing {
			fmt.Fprintln(os.Stderr, i18n.T(w.lang, "app.windowFailed"), err)
			openBrowserWindow(w.url)
		}
	}()
	return true
}

// close убирает окно с экрана перед завершением программы.
func (w *window) close() {
	w.mu.Lock()
	w.closing = true
	p := w.proc
	w.mu.Unlock()
	if p != nil {
		p.Signal(syscall.SIGTERM)
	}
}

// helperPath ищет программу окна рядом с собой, затем в PATH.
//
// Рядом — чтобы собранное в дереве исходников работало без установки;
// в PATH — чтобы работал установленный пакет.
func helperPath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), helperName)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if p, err := exec.LookPath(helperName); err == nil {
		return p
	}
	return ""
}

// openBrowserWindow — запасной путь, когда своего окна нет.
//
// Браузеры на основе Chromium умеют режим приложения: окно без вкладок,
// адресной строки и кнопок. Имя окну они дают своё; на X11 его перебивает
// --class, под Wayland этот ключ не действует, и окно останется подписанным
// по-браузерному. Ради этого случая и появилось своё окно.
func openBrowserWindow(url string) {
	size := fmt.Sprintf("--window-size=%d,%d", windowW, windowH)

	for _, name := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
		"brave-browser", "microsoft-edge", "vivaldi",
	} {
		bin, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, "--app="+url, "--class="+appID,
			"--no-first-run", "--no-default-browser-check", size)
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
