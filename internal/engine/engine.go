package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/i18n"
	"hyperx-studio/internal/input"
	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/layout"
)

// Engine держит устройство и крутит рендер.
type Engine struct {
	mu sync.Mutex

	dev    *keyboard.Device
	reader *input.Reader

	st      State
	ctx     *effects.Context
	effect  effects.Effect
	overlay effects.Effect
	evIndex map[string]int
	keyByIx map[int]layout.Key

	hits      []effects.Hit
	frame     []keyboard.RGB
	t0        time.Time
	paused    bool
	stop      chan struct{}
	dirty     bool // состояние изменилось, надо сохранить
	saveAt    time.Time
	subs      map[chan []keyboard.RGB]struct{}
	connected bool
	FPSReal   float64
	InputOK   bool
	InputErr  string
}

func New() (*Engine, error) {
	paths, err := keyboard.Candidates()
	if err != nil || len(paths) == 0 {
		return nil, fmt.Errorf(i18n.T(Load().Lang, "err.noDevice"),
			keyboard.VendorID, keyboard.ProductID)
	}
	var dev *keyboard.Device
	var lastErr error
	for _, p := range paths {
		dev, lastErr = keyboard.Open(p)
		if lastErr == nil {
			break
		}
	}
	if dev == nil {
		return nil, fmt.Errorf(i18n.T(Load().Lang, "err.noAccess"), lastErr)
	}

	e := &Engine{
		dev:   dev,
		st:    Load(),
		frame: make([]keyboard.RGB, keyboard.LEDCount),
		stop:  make(chan struct{}),
		subs:  map[chan []keyboard.RGB]struct{}{},
		t0:    time.Now(),
	}
	e.connected = true
	e.rebuild()
	e.effect = effects.New(e.st.Effect)
	if e.st.Overlay != "" {
		e.overlay = effects.New(e.st.Overlay)
	}

	e.reader = &input.Reader{OnKey: e.onKey}
	if e.reader.Start() {
		e.InputOK = true
	} else if e.reader.Err != nil {
		e.InputErr = e.reader.Err.Error()
	}
	return e, nil
}

func (e *Engine) DevicePath() string { return e.dev.Path() }

// rebuild пересобирает контекст под текущую раскладку.
func (e *Engine) rebuild() {
	keys := layout.Variant(e.st.Variant)
	w, h := layout.Bounds(keys)
	perKey := map[int]effects.RGB{}
	for k, v := range e.st.PerKey {
		i, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		if c, ok := parseHex(v); ok {
			perKey[i] = c
		}
	}
	sel := map[int]bool{}
	for _, i := range e.st.Selection {
		sel[i] = true
	}
	e.ctx = &effects.Context{
		Keys: keys, W: w, H: h, P: e.st.Params,
		PerKey: perKey, Selection: sel,
	}
	e.evIndex = layout.EvIndex(keys)
	e.keyByIx = map[int]layout.Key{}
	for _, k := range keys {
		e.keyByIx[k.Index] = k
	}
}

func parseHex(s string) (effects.RGB, bool) {
	if len(s) == 7 && s[0] == '#' {
		v, err := strconv.ParseUint(s[1:], 16, 32)
		if err == nil {
			return effects.RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, true
		}
	}
	return effects.RGB{}, false
}

func hexOf(c effects.RGB) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// onKey вызывается из потока чтения ввода.
func (e *Engine) onKey(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	idx, ok := e.evIndex[name]
	if !ok {
		return
	}
	k := e.keyByIx[idx]
	e.hits = append(e.hits, effects.Hit{
		Index: idx, CX: k.CX(), CY: k.CY(),
		Born: time.Since(e.t0).Seconds(), Seed: rand.Float64(),
	})
	if len(e.hits) > 64 {
		e.hits = e.hits[len(e.hits)-64:]
	}
}

// Run крутит рендер до вызова Close.
func (e *Engine) Run() {
	frames, tick := 0, time.Now()
	for {
		select {
		case <-e.stop:
			return
		default:
		}
		start := time.Now()
		t := start.Sub(e.t0).Seconds()

		e.mu.Lock()
		paused := e.paused
		fps := e.st.FPS
		eff, ovl := e.effect, e.overlay
		// хиты живут не дольше времени затухания
		keep := e.st.Params.ReactFade + 0.5
		alive := e.hits[:0]
		for _, h := range e.hits {
			if t-h.Born < keep {
				alive = append(alive, h)
			}
		}
		e.hits = alive
		e.ctx.Hits = append([]effects.Hit(nil), e.hits...)
		e.ctx.P = e.st.Params
		mask := e.st.MaskSel
		needSave := e.dirty && !e.saveAt.IsZero() && time.Now().After(e.saveAt)
		if needSave {
			e.dirty = false
			e.saveAt = time.Time{}
		}
		stCopy := e.st
		e.mu.Unlock()

		if needSave {
			if err := Save(stCopy); err != nil {
				fmt.Println(i18n.T(stCopy.Lang, "err.saveSettings"), err)
			}
		}

		if paused {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		buf := make([]keyboard.RGB, keyboard.LEDCount)
		eff.Render(t, buf, e.ctx, false)
		if ovl != nil {
			tmp := make([]keyboard.RGB, keyboard.LEDCount)
			ovl.Render(t, tmp, e.ctx, true)
			for _, k := range e.ctx.Keys {
				if tmp[k.Index] != (keyboard.RGB{}) {
					buf[k.Index] = effects.Add(buf[k.Index], tmp[k.Index])
				}
			}
		}
		if mask && len(e.ctx.Selection) > 0 {
			for _, k := range e.ctx.Keys {
				if !e.ctx.Selection[k.Index] {
					buf[k.Index] = keyboard.RGB{}
				}
			}
		}
		if b := e.ctx.P.Brightness; b < 0.999 {
			for i := range buf {
				buf[i] = effects.Scale(buf[i], b)
			}
		}

		showErr := e.dev.Show(buf)
		if showErr != nil {
			// после переподключения клавиатуры прежний hidraw мёртв,
			// а номер узла меняется — ищем устройство заново
			if !e.reopen() {
				e.setDisconnected(showErr)
				e.mu.Lock()
				e.frame = buf
				e.mu.Unlock()
				e.publish(buf)
				time.Sleep(time.Second)
				continue
			}
		}
		e.setConnected()

		e.mu.Lock()
		e.frame = buf
		e.mu.Unlock()
		e.publish(buf)

		frames++
		if now := time.Now(); now.Sub(tick) >= time.Second {
			e.mu.Lock()
			e.FPSReal = float64(frames) / now.Sub(tick).Seconds()
			e.mu.Unlock()
			frames, tick = 0, now
		}

		if fps < 1 {
			fps = 1
		}
		if d := time.Second/time.Duration(fps) - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}

// reopen пытается заново открыть устройство: номер узла hidraw после
// переподключения другой, поэтому ищем по идентификаторам, а не по пути.
func (e *Engine) reopen() bool {
	paths, err := keyboard.Candidates()
	if err != nil || len(paths) == 0 {
		return false
	}
	for _, p := range paths {
		dev, err := keyboard.Open(p)
		if err != nil {
			continue
		}
		e.mu.Lock()
		old := e.dev
		e.dev = dev
		e.mu.Unlock()
		if old != nil {
			old.Close()
		}
		return true
	}
	return false
}

func (e *Engine) setDisconnected(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.connected {
		fmt.Println(i18n.T(e.st.Lang, "dev.lost", err))
	}
	e.connected = false
}

func (e *Engine) setConnected() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.connected {
		fmt.Println(i18n.T(e.st.Lang, "dev.restored", e.dev.Path()))
	}
	e.connected = true
}

// ApplyOnce рисует один кадр и отправляет его. Нужно, чтобы применить
// статичную схему и выйти — подсветка останется без работающей программы.
func (e *Engine) ApplyOnce() error {
	buf := make([]keyboard.RGB, keyboard.LEDCount)
	e.effect.Render(0, buf, e.ctx, false)
	if b := e.ctx.P.Brightness; b < 0.999 {
		for i := range buf {
			buf[i] = effects.Scale(buf[i], b)
		}
	}
	return e.dev.Show(buf)
}

func (e *Engine) Close() {
	close(e.stop)
	if e.reader != nil {
		e.reader.Stop()
	}
	e.mu.Lock()
	st := e.st
	dirty := e.dirty
	e.mu.Unlock()
	if dirty {
		Save(st)
	}
	// подсветку намеренно не гасим: пусть остаётся последний кадр
	e.dev.Close()
}
