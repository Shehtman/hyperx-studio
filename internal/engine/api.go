package engine

import (
	"strconv"
	"time"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/i18n"
	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/layout"
)

// ── трансляция кадров подписчикам (для предпросмотра в интерфейсе) ────

func (e *Engine) Subscribe() chan []keyboard.RGB {
	ch := make(chan []keyboard.RGB, 2)
	e.mu.Lock()
	e.subs[ch] = struct{}{}
	e.mu.Unlock()
	return ch
}

func (e *Engine) Unsubscribe(ch chan []keyboard.RGB) {
	e.mu.Lock()
	delete(e.subs, ch)
	e.mu.Unlock()
	close(ch)
}

func (e *Engine) publish(buf []keyboard.RGB) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for ch := range e.subs {
		select {
		case ch <- buf:
		default: // подписчик не успевает — пропускаем кадр, а не копим очередь
		}
	}
}

// ── чтение состояния ─────────────────────────────────────────────────

// Snapshot — состояние для интерфейса.
type Snapshot struct {
	State     State                `json:"state"`
	Keys      []layout.Key         `json:"keys"`
	Width     float64              `json:"width"`
	Height    float64              `json:"height"`
	Effects   []effects.Descriptor `json:"effects"`
	Device    string               `json:"device"`
	LEDs      int                  `json:"leds"`
	FPSReal   float64              `json:"fpsReal"`
	Paused    bool                 `json:"paused"`
	InputOK   bool                 `json:"inputOk"`
	InputErr  string               `json:"inputErr"`
	Devices   []string             `json:"inputDevices"`
	Connected bool                 `json:"connected"`
	DevErr    string               `json:"devErr"`
}

func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	var devs []string
	if e.reader != nil {
		devs = e.reader.Devices
	}
	return Snapshot{
		State: e.st, Keys: e.ctx.Keys, Width: e.ctx.W, Height: e.ctx.H,
		Effects: effects.All, Device: e.devPath(), LEDs: keyboard.LEDCount,
		FPSReal: e.FPSReal, Paused: e.paused,
		InputOK: e.InputOK, Devices: devs, Connected: e.connected,
	}
}

func (e *Engine) Frame() []keyboard.RGB {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.frame
}

// ── изменение состояния ──────────────────────────────────────────────

// touch откладывает сохранение: копим правки и пишем раз в полторы секунды,
// чтобы состояние пережило и грубое завершение процесса.
func (e *Engine) touch() {
	e.dirty = true
	e.saveAt = time.Now().Add(1500 * time.Millisecond)
}

func (e *Engine) SetEffect(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Effect = id
	e.effect = effects.New(id)
	e.touch()
}

func (e *Engine) SetOverlay(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Overlay = id
	if id == "" {
		e.overlay = nil
	} else {
		e.overlay = effects.New(id)
	}
	e.touch()
}

func (e *Engine) SetParams(p effects.Params) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Params = p
	e.ctx.P = p
	e.touch()
}

func (e *Engine) SetVariant(v string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if v != "ansi" && v != "iso" {
		return
	}
	e.st.Variant = v
	e.rebuild()
	e.touch()
}

func (e *Engine) SetFPS(fps int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if fps < 1 {
		fps = 1
	}
	if fps > 120 {
		fps = 120
	}
	e.st.FPS = fps
	e.touch()
}

func (e *Engine) SetMask(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.MaskSel = on
	e.touch()
}

func (e *Engine) SetSelection(idx []int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Selection = idx
	sel := map[int]bool{}
	for _, i := range idx {
		sel[i] = true
	}
	e.ctx.Selection = sel
	e.touch()
}

// PaintKeys закрашивает выбранные клавиши и переводит в «Статику»,
// иначе заливка была бы не видна под анимацией.
func (e *Engine) PaintKeys(idx []int, hex string) {
	c, ok := parseHex(hex)
	if !ok {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(idx) == 0 {
		for _, k := range e.ctx.Keys {
			idx = append(idx, k.Index)
		}
	}
	for _, i := range idx {
		e.ctx.PerKey[i] = c
		e.st.PerKey[strconv.Itoa(i)] = hexOf(c)
	}
	e.st.Effect = "static"
	e.effect = effects.New("static")
	e.touch()
}

func (e *Engine) ClearPainted() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ctx.PerKey = map[int]effects.RGB{}
	e.st.PerKey = map[string]string{}
	e.touch()
}

func (e *Engine) SetPaused(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.paused = on
}

// Blackout гасит подсветку и останавливает рендер.
func (e *Engine) Blackout() error {
	e.mu.Lock()
	e.paused = true
	dev := e.dev
	e.mu.Unlock()
	if dev == nil {
		return errNoDevice
	}
	return dev.Off()
}

// SetLang меняет язык сообщений командной строки; интерфейс переключается
// сам, без обращения к серверу.
func (e *Engine) SetLang(lang string) {
	if !i18n.Valid(lang) {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Lang = lang
	e.touch()
}

func (e *Engine) SetAutostart(on bool) {
	e.mu.Lock()
	e.st.Autostart = on
	e.touch()
	e.mu.Unlock()
}
