package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/engine"
	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/webui"
)

func newRouter(eng *engine.Engine, quit chan<- struct{}, show chan<- struct{}) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", noCache(http.FileServer(http.FS(webui.Files))))

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

	post("/api/preset/apply", func(d *json.Decoder) error {
		var b struct {
			ID string `json:"id"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		if !eng.ApplyPreset(b.ID) {
			return fmt.Errorf("схема %q не найдена", b.ID)
		}
		return nil
	})

	post("/api/preset/save", func(d *json.Decoder) error {
		var b struct {
			Name string `json:"name"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		name := strings.TrimSpace(b.Name)
		if name == "" {
			return errors.New("нужно имя схемы")
		}
		if len(name) > 40 {
			name = name[:40]
		}
		// Встроенные схемы адресуются по идентификатору: одноимённая
		// пользовательская сделала бы обращение к ним неоднозначным.
		for _, p := range engine.Builtin() {
			if p.ID == name {
				return fmt.Errorf("имя %q занято встроенной схемой", name)
			}
		}
		eng.SavePreset(name)
		return nil
	})

	post("/api/preset/delete", func(d *json.Decoder) error {
		var b struct {
			Name string `json:"name"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		if !eng.DeletePreset(b.Name) {
			return fmt.Errorf("схема %q не найдена", b.Name)
		}
		return nil
	})

	// Выбор устройства вручную. Пустой путь возвращает автоопределение.
	post("/api/device", func(d *json.Decoder) error {
		var b struct {
			Path string `json:"path"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		return eng.SetDevice(strings.TrimSpace(b.Path))
	})

	post("/api/audio/source", func(d *json.Decoder) error {
		var b struct {
			Name string `json:"name"`
		}
		if err := d.Decode(&b); err != nil {
			return err
		}
		eng.SetAudioSource(b.Name)
		return nil
	})

	// Список источников ходит в pactl, поэтому отдаётся отдельно, а не с
	// каждым опросом состояния.
	mux.HandleFunc("/api/audio/sources", func(w http.ResponseWriter, r *http.Request) {
		src, err := engine.AudioSources()
		if err != nil {
			writeJSON(w, []engine.AudioSource{})
			return
		}
		writeJSON(w, src)
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

	// Просьба показать окно. Её шлёт второй запуск программы: интерфейс
	// рисует работающий экземпляр, а не тот, который только что запустили.
	mux.HandleFunc("/api/window", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "нужен POST", http.StatusMethodNotAllowed)
			return
		}
		select {
		case show <- struct{}{}:
		default: // просьба уже в очереди — второй раз незачем
		}
		w.WriteHeader(http.StatusNoContent)
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

		// Устройству кадры идут с полной частотой, а предпросмотру столько
		// не нужно: сотня клавиш в окне стоит дороже самой отправки на
		// клавиатуру. Глазу хватает двадцати кадров в секунду.
		//
		// Лишние кадры при этом откладываются, а не выбрасываются. Выброшенный
		// кадр не беда, пока за ним идёт следующий, — но последний кадр перед
		// остановкой рендера следующего не имеет. Так пропадало погашение:
		// клавиатура гасла, а окно до конца сеанса показывало прежнюю картинку.
		tick := time.NewTicker(previewInterval)
		defer tick.Stop()

		var pending []keyboard.RGB
		for {
			select {
			case <-r.Context().Done():
				return
			case frame := <-ch:
				pending = frame
			case <-tick.C:
				if pending == nil {
					continue
				}
				frame := pending
				pending = nil

				st := eng.Status()
				conn := 0
				if st.Connected {
					conn = 1
				}
				fmt.Fprintf(w, "data: %s|%.0f|%d|%.3f\n\n",
					keyboard.Hex(frame), st.FPSReal, conn, st.Level)
				flusher.Flush()
			case <-ping.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	})

	return mux
}

// previewInterval — как часто кадр уходит в интерфейс.
const previewInterval = 50 * time.Millisecond

// noCache заставляет браузер каждый раз спрашивать сервер о свежести
// интерфейса.
//
// Файлы вшиты в исполняемый файл и отдаются без даты изменения, поэтому
// браузер кэширует их на своё усмотрение. После обновления программы это
// оборачивалось старым интерфейсом в уже открытом окне: сервер отдаёт новые
// эффекты, а страница остаётся прежней.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
