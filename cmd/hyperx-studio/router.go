package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/webui"
)

func newRouter(eng *engine.Engine, quit chan<- struct{}) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(webui.Files)))

	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, eng.Snapshot())
	})

	post := func(path string, fn func(*json.Decoder) error) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "нужен POST", http.StatusMethodNotAllowed)
				return
			}
			if err := fn(json.NewDecoder(r.Body)); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}

	post("/api/effect", func(d *json.Decoder) error {
		var b struct {
			ID string `json:"id"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetEffect(b.ID)
		return nil
	})

	post("/api/overlay", func(d *json.Decoder) error {
		var b struct {
			ID string `json:"id"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetOverlay(b.ID)
		return nil
	})

	post("/api/params", func(d *json.Decoder) error {
		var p effects.Params
		if err := d.Decode(&p); err != nil {
			return err
		}
		eng.SetParams(p)
		return nil
	})

	post("/api/variant", func(d *json.Decoder) error {
		var b struct {
			Variant string `json:"variant"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetVariant(b.Variant)
		return nil
	})

	post("/api/fps", func(d *json.Decoder) error {
		var b struct {
			FPS int `json:"fps"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetFPS(b.FPS)
		return nil
	})

	post("/api/mask", func(d *json.Decoder) error {
		var b struct {
			On bool `json:"on"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetMask(b.On)
		return nil
	})

	post("/api/selection", func(d *json.Decoder) error {
		var b struct {
			Indices []int `json:"indices"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetSelection(b.Indices)
		return nil
	})

	post("/api/paint", func(d *json.Decoder) error {
		var b struct {
			Indices []int  `json:"indices"`
			Color   string `json:"color"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.PaintKeys(b.Indices, b.Color)
		return nil
	})

	post("/api/clear-paint", func(*json.Decoder) error {
		eng.ClearPainted()
		return nil
	})

	post("/api/pause", func(d *json.Decoder) error {
		var b struct {
			On bool `json:"on"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetPaused(b.On)
		return nil
	})

	post("/api/blackout", func(*json.Decoder) error {
		return eng.Blackout()
	})

	// Вызывается хуком systemd вокруг сна системы. Клавиатуру нужно отпустить
	// до заморозки процессов, иначе она останется в прямом режиме и компьютер
	// не проснётся от нажатия клавиши.
	post("/api/power/sleep", func(*json.Decoder) error {
		eng.Sleep()
		return nil
	})

	post("/api/power/wake", func(*json.Decoder) error {
		return eng.Wake()
	})

	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "нужен POST", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go func() {
			time.Sleep(200 * time.Millisecond) // дать ответу уйти
			quit <- struct{}{}
		}()
	})

	post("/api/lang", func(d *json.Decoder) error {
		var b struct {
			Lang string `json:"lang"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetLang(b.Lang)
		return nil
	})

	post("/api/autostart", func(d *json.Decoder) error {
		var b struct {
			On bool `json:"on"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		if err := engine.ApplyAutostart(b.On); err != nil {
			return err
		}
		eng.SetAutostart(b.On)
		return nil
	})

	// Поток кадров для предпросмотра. Обычный SSE: отдельная библиотека
	// для веб-сокетов ради односторонней передачи не нужна.
	mux.HandleFunc("/api/frames", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "поток не поддерживается", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := eng.Subscribe()
		defer eng.Unsubscribe(ch)

		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case frame := <-ch:
				snap := eng.Snapshot()
				conn := 0
				if snap.Connected {
					conn = 1
				}
				fmt.Fprintf(w, "data: %s|%.0f|%d\n\n",
					keyboard.Hex(frame), snap.FPSReal, conn)
				flusher.Flush()
			case <-ping.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	})

	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
