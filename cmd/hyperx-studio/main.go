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
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/i18n"
)

func main() {
	var (
		addr      = flag.String("addr", "127.0.0.1:7423", "адрес интерфейса")
		noWindow  = flag.Bool("no-window", false, "не открывать браузер")
		applyOnly = flag.Bool("apply", false, "применить сохранённую схему и выйти")
		off       = flag.Bool("off", false, "погасить подсветку и выйти")
		showVer   = flag.Bool("version", false, "показать версию")
		instUdev  = flag.Bool("install-udev", false, "записать правило доступа к устройству (нужен root)")
		printUdev = flag.Bool("print-udev", false, "вывести правило udev и выйти")
		quit      = flag.Bool("quit", false, "остановить работающий экземпляр")
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
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.permsHint"))
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.installUdev"))
		os.Exit(1)
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
			if !*noWindow {
				openWindow("http://" + *addr)
			}
			return
		}
		fmt.Fprintln(os.Stderr, i18n.T(lang, "err.portBusy", *addr, err))
		eng.Close()
		os.Exit(1)
	}

	quitCh := make(chan struct{}, 1)
	srv := &http.Server{Handler: newRouter(eng, quitCh)}
	url := "http://" + ln.Addr().String()
	fmt.Println(i18n.T(lang, "app.url", url))
	fmt.Println(i18n.T(lang, "app.device", eng.DevicePath()))

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "сервер остановлен:", err)
		}
	}()

	if !*noWindow {
		openWindow(url)
	}
	fmt.Println(i18n.T(lang, "app.windowHint"))
	fmt.Println(i18n.T(lang, "app.quitHint"))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-quitCh:
	}

	fmt.Println("\n" + i18n.T(lang, "app.shuttingDown"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	eng.Close() // подсветка намеренно остаётся гореть последним кадром
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
