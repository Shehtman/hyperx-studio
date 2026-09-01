package engine

import (
	"sync"

	"hyperx-studio/internal/audio"
	"hyperx-studio/internal/effects"
)

// Захват звука включается сам, когда выбран звуковой эффект, и глушится,
// когда он больше не нужен: держать открытым чужой поток без дела не стоит.

// audioNeeded — нужен ли сейчас звук. Вызывается под e.mu.
func (e *Engine) audioNeeded() bool {
	return effects.NeedsAudio(e.st.Effect) || effects.NeedsAudio(e.st.Overlay)
}

// syncAudio приводит захват в соответствие с выбранным эффектом.
// Вызывается без блокировки: запуск утилиты записи не мгновенный.
func (e *Engine) syncAudio() {
	e.mu.Lock()
	need := e.audioNeeded()
	src := e.st.AudioSource
	running := e.audio != nil
	sameSource := e.audioSrc == src
	e.mu.Unlock()

	switch {
	case need && running && sameSource:
		return
	case !need && !running:
		return
	}

	// Старый захват в любом случае лишний: либо звук больше не нужен,
	// либо сменился источник.
	e.mu.Lock()
	old := e.audio
	e.audio = nil
	e.mu.Unlock()
	if old != nil {
		old.Stop()
	}
	if !need {
		e.mu.Lock()
		e.AudioErr = ""
		e.mu.Unlock()
		return
	}

	cap, err := audio.Start(src)
	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		e.AudioErr = err.Error()
		return
	}
	e.audio = cap
	e.audioSrc = src
	e.AudioErr = ""
}

// sound собирает картину звука для очередного кадра. Вызывается под e.mu.
func (e *Engine) sound() effects.Sound {
	if e.audio == nil {
		return effects.Sound{}
	}
	if err := e.audio.Err(); err != nil {
		e.AudioErr = err.Error()
		return effects.Sound{}
	}
	f := e.audio.Frame()
	bands := make([]float64, len(f.Bands))
	copy(bands, f.Bands[:])
	return effects.Sound{On: true, Level: f.Level, Beat: f.Beat, Bands: bands}
}

// stopAudio глушит захват. Нужен при остановке программы и перед сном.
func (e *Engine) stopAudio() {
	e.mu.Lock()
	c := e.audio
	e.audio = nil
	e.mu.Unlock()
	if c != nil {
		c.Stop()
	}
}

// AudioSource — источник звука для интерфейса.
type AudioSource = audio.Source

// AudioSources перечисляет, что можно слушать.
func AudioSources() ([]AudioSource, error) { return audio.Sources() }

// AudioAvailable сообщает, есть ли в системе чем захватывать звук.
func AudioAvailable() bool { return audioAvailable() }

// audioAvailable кэширует ответ: он не меняется за время работы программы,
// а спрашивают его на каждом опросе состояния.
var audioAvailable = func() func() bool {
	var once sync.Once
	var ok bool
	return func() bool {
		once.Do(func() { ok = audio.Available() })
		return ok
	}
}()
