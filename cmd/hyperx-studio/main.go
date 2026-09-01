// HyperX Studio — управление подсветкой клавиатуры HyperX Alloy Origins.
//
// Одна программа: сама говорит с устройством через hidraw, сама считает
// эффекты, сама отдаёт интерфейс. Ни OpenRGB, ни других служб не требуется.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/i18n"
	"hyperx-studio/internal/keyboard"
)

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:7423", "адрес интерфейса")
		noWindow   = flag.Bool("no-window", false, "не открывать окно")
		useBrowser = flag.Bool("browser", false, "показать интерфейс в браузере, а не своим окном")
		applyOnly  = flag.Bool("apply", false, "применить сохранённую схему и выйти")
		off        = flag.Bool("off", false, "погасить подсветку и выйти")
		showVer    = flag.Bool("version", false, "показать версию")
		instUdev   = flag.Bool("install-udev", false, "записать правило доступа к устройству (нужен root)")
		printUdev  = flag.Bool("print-udev", false, "вывести правило udev и выйти")
		quit       = flag.Bool("quit", false, "остановить работающий экземпляр")
		sleepCmd   = flag.Bool("sleep", false, "отпустить клавиатуру перед сном системы")
		wakeCmd    = flag.Bool("wake", false, "вернуть клавиатуру после пробуждения")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("hyperx-studio", version)
		return
	}

	if *quit {
		if err := quitRunning(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(i18n.T(engine.Load().Lang, "app.stopped"))
		return
	}

	if *sleepCmd {
		if err := suspendKeyboard(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *wakeCmd {
		if err := resumeKeyboard(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *printUdev {
		fmt.Print(engine.UdevRule)
		return
	}

	if *instUdev {
		if err := engine.InstallUdev(); err != nil {
			fmt.Fprintln(os.Stderr, "Ошибка:", err)
			os.Exit(1)
		}
		return
	}

	lang := engine.Load().Lang

	eng, err := engine.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.prefix"), err)
		os.Exit(1)
	}
	// Клавиатуры может не быть — это не повод не запускаться. Программа
	// работает предпросмотром и подхватит устройство, как только оно
	// появится; в окне об этом сказано отдельной строкой.
	if eng.DevicePath() == "" {
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.prefix"), eng.DeviceError())
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.permsHint"))
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.installUdev"))
	}

	switch {
	case *off:
		if err := eng.Blackout(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.T(lang, "err.blackout"), err)
			os.Exit(1)
		}
		eng.Close()
		return
	case *applyOnly:
		// статичная схема остаётся на клавиатуре и после выхода
		if err := eng.ApplyOnce(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.T(lang, "err.apply"), err)
			os.Exit(1)
		}
		eng.Close()
		return
	}

	go eng.Run()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		if alreadyRunning(*addr) {
			fmt.Println(i18n.T(lang, "app.alreadyOpen"))
			eng.Close()
			// Окно рисует работающий экземпляр — второму остаётся его
			// попросить. Раньше здесь открывалось второе окно поверх
			// первого, и закрытие любого из них ничего не значило.
			if !*noWindow && !showRunningWindow(*addr) {
				openBrowserWindow("http://" + *addr)
			}
			return
		}
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.portBusy", *addr, err))
		eng.Close()
		os.Exit(1)
	}

	quitCh := make(chan struct{}, 1)
	showWin := make(chan struct{}, 1)
	srv := &http.Server{Handler: newRouter(eng, quitCh, showWin)}
	url := "http://" + ln.Addr().String()
	fmt.Println(i18n.T(lang, "app.url", url))
	fmt.Println(i18n.T(lang, "app.device", eng.DevicePath()))

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "сервер остановлен:", err)
		}
	}()

	win := &window{url: url, lang: lang, browser: *useBrowser}
	if !*noWindow {
		win.show()
	}
	// Повторный запуск программы не поднимает второй экземпляр, а просит
	// этот показать окно.
	go func() {
		for range showWin {
			win.show()
		}
	}()

	fmt.Println(i18n.T(lang, "app.windowHint"))
	fmt.Println(i18n.T(lang, "app.quitHint"))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-quitCh:
	}
	win.close()

	fmt.Println("\n" + i18n.T(lang, "app.shuttingDown"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	eng.Close() // подсветка намеренно остаётся гореть последним кадром
}

// showRunningWindow просит уже работающий экземпляр показать своё окно.
func showRunningWindow(addr string) bool {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Post("http://"+addr+"/api/window", "application/json", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 300
}

// alreadyRunning проверяет, что порт занят именно нашим экземпляром.
func alreadyRunning(addr string) bool {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://" + addr + "/api/state")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// suspendKeyboard вызывается перед сном системы.
//
// Работающему экземпляру достаточно сказать «отпусти»: он сам остановит
// рендер и переинициализирует устройство. Если программа не запущена, режим
// --apply мог оставить клавиатуру в прямом режиме — снимаем его сами,
// иначе компьютер не проснётся от клавиатуры и мыши.
func suspendKeyboard(addr string) error {
	if alreadyRunning(addr) {
		c := http.Client{Timeout: 10 * time.Second}
		resp, err := c.Post("http://"+addr+"/api/power/sleep", "application/json", nil)
		if err == nil {
			resp.Body.Close()
			return nil
		}
	}
	return keyboard.ResetUSB()
}

// resumeKeyboard возвращает подсветку после пробуждения.
func resumeKeyboard(addr string) error {
	if alreadyRunning(addr) {
		c := http.Client{Timeout: 15 * time.Second}
		resp, err := c.Post("http://"+addr+"/api/power/wake", "application/json", nil)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("%s", strings.TrimSpace(string(body)))
			}
			return nil
		}
	}
	// Демона нет: восстанавливаем сохранённую схему так же, как --apply.
	eng, err := engine.New()
	if err != nil {
		return err
	}
	defer eng.Close()
	return eng.ApplyOnce()
}

// quitRunning просит работающий экземпляр завершиться.
func quitRunning(addr string) error {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Post("http://"+addr+"/api/quit", "application/json", nil)
	if err != nil {
		return fmt.Errorf("%s", i18n.T(engine.Load().Lang, "app.notRunning"))
	}
	resp.Body.Close()
	return nil
}
