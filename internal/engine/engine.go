package engine

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"hyperx-studio/internal/audio"
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
	audio  *audio.Capture

	audioSrc string // источник, на котором запущен текущий захват
	AudioErr string

	// devMu держится на время передачи кадра.
	//
	// Устройство закрывают не только цикл рендера: его меняют из
	// обработчика запросов и отпускают перед сном системы. Без этого замка
	// файл может закрыться прямо посреди отправки, а его номер к тому
	// времени уже достаться кому-то другому.
	//
	// Порядок замков всегда devMu → mu, иначе получится взаимная блокировка.
	devMu sync.Mutex

	// resetSend просит цикл рендера забыть последний отправленный кадр.
	//
	// Сам кадр живёт переменной внутри цикла: писать его туда, откуда
	// читают обработчики запросов, значило бы делить память между
	// потоками без нужды. Здесь достаточно флажка под общим замком.
	resetSend bool

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
	asleep    bool // устройство отпущено на время сна системы
	loopIdle  bool // цикл рендера остановлен и к устройству не обращается
	stop      chan struct{}
	dirty     bool // состояние изменилось, надо сохранить
	saveAt    time.Time
	subs      map[chan []keyboard.RGB]struct{}
	connected bool
	devErr    string // почему устройство не открыто
	FPSReal   float64
	InputOK   bool
	InputErr  string
}

// New поднимает движок. Отсутствие клавиатуры ошибкой не считается.
//
// Раньше без Alloy Origins программа просто не запускалась. Это неудобно и
// незачем: эффекты считаются независимо от устройства, их можно смотреть в
// окне, настраивать схемы и ждать, пока клавиатуру подключат. Цикл рендера
// сам подхватит её, как только она появится.
func New() (*Engine, error) {
	e := &Engine{
		st:    Load(),
		frame: make([]keyboard.RGB, keyboard.LEDCount),
		stop:  make(chan struct{}),
		subs:  map[chan []keyboard.RGB]struct{}{},
		t0:    time.Now(),
	}
	if dev, err := openDevice(e.st.Device); err == nil {
		e.dev = dev
		e.connected = true
	} else {
		e.devErr = err.Error()
	}
	e.rebuild()
	e.effect = effects.New(e.st.Effect)
	if e.st.Overlay != "" {
		e.overlay = effects.New(e.st.Overlay)
	}

	go e.syncAudio()

	e.reader = &input.Reader{OnKey: e.onKey}
	if e.reader.Start() {
		e.InputOK = true
	} else if e.reader.Err != nil {
		e.InputErr = e.reader.Err.Error()
	}
	return e, nil
}

func (e *Engine) DevicePath() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.devPath()
}

// devPath отдаёт путь открытого устройства. Во время сна системы оно
// отпущено, и указателя нет — вызывать Path() напрямую нельзя.
// Вызывается под e.mu.
func (e *Engine) devPath() string {
	if e.dev == nil {
		return ""
	}
	return e.dev.Path()
}

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
	last := time.Now()
	var lastProbe time.Time
	var sentFrame []keyboard.RGB // последний отправленный на устройство кадр
	var lastSend time.Time
	for {
		select {
		case <-e.stop:
			return
		default:
		}
		start := time.Now()
		t := start.Sub(e.t0).Seconds()

		// Шаг времени для эффектов, которые копят состояние. Ограничен
		// сверху: после паузы, сна системы или подвисшего кадра дождь не
		// должен прыгать через всю клавиатуру.
		dt := start.Sub(last).Seconds()
		last = start
		if dt <= 0 || dt > 0.25 {
			dt = 0.25
		}

		e.mu.Lock()
		paused := e.paused || e.asleep
		// Пока цикл стоит, он не трогает устройство — только в этот момент
		// его можно безопасно закрыть перед сном системы.
		e.loopIdle = paused
		fps := e.st.FPS
		if fps < 1 {
			fps = 1
		}
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
		e.ctx.Dt = dt
		e.ctx.Sound = e.sound()
		mask := e.st.MaskSel
		needSave := e.dirty && !e.saveAt.IsZero() && time.Now().After(e.saveAt)
		if needSave {
			e.dirty = false
			e.saveAt = time.Time{}
		}
		stCopy := e.st
		resend := e.resetSend
		e.resetSend = false
		e.mu.Unlock()

		if resend {
			sentFrame = sentFrame[:0]
			lastSend = time.Time{}
		}

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

		// Одинаковые кадры на устройство не гоняем. Каждый кадр — девять
		// управляющих пакетов, а это медленный канал: на статике и на
		// спокойных эффектах повторять их незачем. Изредка кадр всё же
		// повторяем, иначе клавиатура вернётся к своему эффекту.
		showErr := errNoDevice
		e.devMu.Lock()
		e.mu.Lock()
		dev := e.dev
		e.mu.Unlock()
		if dev != nil {
			if sameFrame(buf, sentFrame) && time.Since(lastSend) < holdRefresh {
				showErr = nil
			} else {
				showErr = dev.Show(buf)
				if showErr == nil {
					sentFrame = append(sentFrame[:0], buf...)
					lastSend = time.Now()
				}
			}
		}
		e.devMu.Unlock()
		if showErr != nil {
			// После переподключения клавиатуры прежний hidraw мёртв, а
			// номер узла меняется — ищем устройство заново. Пробуем не
			// чаще раза в секунду: перебор ходит в sysfs, и без
			// клавиатуры это повторялось бы каждый кадр.
			ok := false
			if time.Since(lastProbe) >= time.Second {
				lastProbe = time.Now()
				ok = e.reopen()
			}
			if !ok {
				e.setDisconnected(showErr)
				e.mu.Lock()
				e.frame = buf
				e.mu.Unlock()
				e.publish(buf)
				// Кадры продолжают идти в окно с обычной частотой:
				// без устройства программа остаётся живым
				// предпросмотром, а не застывшей картинкой.
				if d := time.Second/time.Duration(fps) - time.Since(start); d > 0 {
					time.Sleep(d)
				}
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

		if d := time.Second/time.Duration(fps) - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}

// holdRefresh — как часто повторять неизменившийся кадр, чтобы клавиатура
// оставалась в прямом режиме и не включила собственный эффект.
const holdRefresh = 400 * time.Millisecond

// sameFrame сравнивает кадры поэлементно.
func sameFrame(a, b []keyboard.RGB) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// errNoDevice — устройство сейчас не открыто. Отдельная ошибка нужна, чтобы
// цикл рендера шёл обычной веткой переподключения, а не падал на nil.
var errNoDevice = errors.New("устройство не открыто")

// Sleep отпускает клавиатуру перед сном системы.
//
// Без этого компьютер невозможно разбудить ни клавиатурой, ни мышью:
// клавиатура остаётся в прямом режиме подсветки, перестаёт сигналить remote
// wakeup и тянет за собой весь USB-контроллер, на котором сидит.
func (e *Engine) Sleep() {
	e.mu.Lock()
	if e.asleep {
		e.mu.Unlock()
		return
	}
	e.asleep = true
	lang := e.st.Lang
	e.mu.Unlock()
	fmt.Println(i18n.T(lang, "sleep.releasing"))

	// Ждём остановки цикла: закрывать устройство во время отправки кадра
	// нельзя. Полсекунды с запасом хватает даже на 1 кадр в секунду.
	for i := 0; i < 100; i++ {
		e.mu.Lock()
		idle := e.loopIdle
		e.mu.Unlock()
		if idle {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if r := e.reader; r != nil {
		r.Stop()
	}
	// Утилиту записи тоже отпускаем: во сне она не нужна, а при
	// пробуждении звуковой сервер может подняться с другими узлами.
	e.stopAudio()
	e.devMu.Lock()
	e.mu.Lock()
	dev := e.dev
	e.dev = nil
	e.reader = nil
	e.mu.Unlock()
	if dev != nil {
		dev.Close()
	}
	e.devMu.Unlock()

	// Закрыть узел мало: прямой режим живёт в прошивке, а не в дескрипторе.
	// Снимаем его переинициализацией устройства.
	if err := keyboard.ResetUSB(); err != nil {
		fmt.Println(i18n.T(lang, "sleep.resetFailed"), err)
	}
}

// Wake возвращает клавиатуру под управление после пробуждения.
func (e *Engine) Wake() error {
	e.mu.Lock()
	if !e.asleep {
		e.mu.Unlock()
		return nil
	}
	lang := e.st.Lang
	e.mu.Unlock()

	// После сна и переинициализации устройство появляется не сразу:
	// ядру нужно заново опросить шину. Ждём до пяти секунд.
	ok := false
	for i := 0; i < 50; i++ {
		if e.reopen() {
			ok = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	reader := &input.Reader{OnKey: e.onKey}
	started := reader.Start()
	go e.syncAudio()

	e.mu.Lock()
	e.reader = reader
	e.InputOK = started
	if !started && reader.Err != nil {
		e.InputErr = reader.Err.Error()
	}
	e.asleep = false
	e.mu.Unlock()

	if !ok {
		return errors.New(i18n.T(lang, "sleep.noDeviceAfterWake"))
	}
	fmt.Println(i18n.T(lang, "sleep.resumed"))
	return nil
}

// reopen пытается заново открыть устройство: номер узла hidraw после
// переподключения другой, поэтому ищем по идентификаторам, а не по пути.
// openDevice открывает узел hidraw. Пустой pref означает поиск нашей
// клавиатуры по идентификаторам; иначе берём ровно то, что выбрал человек.
func openDevice(pref string) (*keyboard.Device, error) {
	if pref != "" {
		return keyboard.Open(pref)
	}
	paths, err := keyboard.Candidates()
	if err != nil || len(paths) == 0 {
		return nil, fmt.Errorf(i18n.T(Load().Lang, "err.noDevice"),
			keyboard.VendorID, keyboard.ProductID)
	}
	var lastErr error
	for _, p := range paths {
		dev, err := keyboard.Open(p)
		if err == nil {
			return dev, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf(i18n.T(Load().Lang, "err.noAccess"), lastErr)
}

func (e *Engine) reopen() bool {
	e.mu.Lock()
	pref := e.st.Device
	e.mu.Unlock()

	e.devMu.Lock()
	defer e.devMu.Unlock()

	dev, err := openDevice(pref)
	if err != nil {
		e.mu.Lock()
		e.devErr = err.Error()
		e.mu.Unlock()
		return false
	}
	{
		e.mu.Lock()
		old := e.dev
		e.dev = dev
		e.devErr = ""
		e.resetSend = true // на новом узле кадр надо послать заново
		e.mu.Unlock()
		if old != nil {
			old.Close()
		}
		return true
	}
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
	if !e.connected && e.dev != nil {
		fmt.Println(i18n.T(e.st.Lang, "dev.restored", e.devPath()))
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
	if e.dev == nil {
		return errNoDevice
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
	e.stopAudio()
	// подсветку намеренно не гасим: пусть остаётся последний кадр
	e.devMu.Lock()
	e.mu.Lock()
	dev := e.dev
	e.dev = nil
	e.mu.Unlock()
	if dev != nil {
		dev.Close()
	}
	e.devMu.Unlock()
}
