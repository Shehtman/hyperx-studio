package engine

import "hyperx-studio/internal/effects"

// Preset — готовая схема: эффект, слой поверх него и все параметры разом.
//
// Встроенные схемы отмечены Builtin: их нельзя удалить, а имя интерфейс
// подставляет сам по идентификатору, чтобы список говорил на языке
// программы. У пользовательских имя своё.
type Preset struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Effect  string         `json:"effect"`
	Overlay string         `json:"overlay"`
	Params  effects.Params `json:"params"`
	Builtin bool           `json:"builtin"`
}

// rgb — короткая запись цвета для таблицы схем ниже.
func rgb(r, g, b uint8) effects.RGB { return effects.RGB{R: r, G: g, B: b} }

// with берёт настройки по умолчанию и меняет только то, что нужно схеме.
// Так новая настройка не забудется ни в одной из них.
func with(f func(*effects.Params)) effects.Params {
	p := effects.DefaultParams()
	f(&p)
	return p
}

// Builtin — схемы, из которых можно выбирать сразу после установки.
func Builtin() []Preset {
	return []Preset{
		{ID: "aurora", Effect: "colorwave", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(0, 255, 170), rgb(120, 0, 255)
				p.Speed, p.Scale, p.Angle = 0.6, 2, 15
			})},
		{ID: "sunset", Effect: "gradient", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(255, 90, 0), rgb(255, 0, 130)
				p.Angle = 20
			})},
		{ID: "matrix", Effect: "rain", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Background = rgb(0, 255, 70), rgb(0, 12, 0)
				p.Speed, p.Density = 1.4, 0.8
			})},
		{ID: "starfield", Effect: "twinkle", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Background = rgb(255, 255, 235), rgb(0, 4, 20)
				p.Speed, p.Density = 0.9, 0.5
			})},
		{ID: "lava", Effect: "fire", Builtin: true,
			Params: with(func(p *effects.Params) { p.Speed = 0.8 })},
		{ID: "ocean", Effect: "colorwave", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(0, 40, 120), rgb(0, 200, 255)
				p.Speed, p.Scale = 0.5, 1.5
			})},
		{ID: "cyberpunk", Effect: "colorwave", Overlay: "ripple", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(255, 0, 200), rgb(0, 230, 255)
				p.ReactColor = rgb(255, 255, 255)
				p.Speed, p.Scale = 0.9, 3
			})},
		{ID: "rainbow", Effect: "rainbow", Builtin: true,
			Params: with(func(p *effects.Params) { p.Speed, p.Scale = 0.8, 1 })},
		{ID: "pastel", Effect: "rainbow", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Speed, p.Scale, p.Saturation = 0.4, 1, 0.35
			})},
		{ID: "typewriter", Effect: "static", Overlay: "flash", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1 = rgb(8, 8, 12)
				p.ReactColor, p.ReactFade = rgb(255, 170, 0), 1.2
			})},
		{ID: "breathe", Effect: "breathing", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Speed = rgb(0, 120, 255), 0.7
			})},
		{ID: "snake", Effect: "snake", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Background = rgb(255, 240, 0), rgb(6, 6, 6)
				p.Speed, p.Length = 0.8, 16
			})},
		{ID: "equalizer", Effect: "audiobars", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(0, 255, 90), rgb(255, 40, 0)
				p.Background = rgb(4, 4, 8)
			})},
		{ID: "bassdrop", Effect: "audiopulse", Builtin: true,
			Params: with(func(p *effects.Params) {
				p.Color1, p.Color2 = rgb(40, 0, 80), rgb(255, 0, 120)
				p.Sensitivity = 1.2
			})},
		{ID: "discolight", Effect: "audiospectrum", Builtin: true,
			Params: with(func(p *effects.Params) { p.Sensitivity = 1.1 })},
	}
}

// Presets отдаёт встроенные схемы вместе с сохранёнными пользователем.
func (e *Engine) Presets() []Preset {
	e.mu.Lock()
	saved := append([]Preset(nil), e.st.Presets...)
	e.mu.Unlock()
	return append(Builtin(), saved...)
}

// ApplyPreset включает схему целиком: эффект, слой и все параметры.
func (e *Engine) ApplyPreset(id string) bool {
	var found *Preset
	for _, p := range e.Presets() {
		if p.key() == id {
			c := p
			found = &c
			break
		}
	}
	if found == nil {
		return false
	}

	e.mu.Lock()
	e.st.Effect = found.Effect
	e.st.Overlay = found.Overlay
	e.st.Params = found.Params
	e.effect = effects.New(found.Effect)
	if found.Overlay == "" {
		e.overlay = nil
	} else {
		e.overlay = effects.New(found.Overlay)
	}
	e.ctx.P = found.Params
	e.touch()
	e.mu.Unlock()

	go e.syncAudio()
	return true
}

// SavePreset запоминает текущую схему под указанным именем. Повторное имя
// перезаписывает прежнюю запись, чтобы список не зарастал дублями.
func (e *Engine) SavePreset(name string) Preset {
	e.mu.Lock()
	defer e.mu.Unlock()

	p := Preset{Name: name, Effect: e.st.Effect, Overlay: e.st.Overlay,
		Params: e.st.Params}
	for i, old := range e.st.Presets {
		if old.Name == name {
			e.st.Presets[i] = p
			e.touch()
			return p
		}
	}
	e.st.Presets = append(e.st.Presets, p)
	e.touch()
	return p
}

// DeletePreset убирает пользовательскую схему. Встроенные не трогаются.
func (e *Engine) DeletePreset(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, p := range e.st.Presets {
		if p.Name == name {
			e.st.Presets = append(e.st.Presets[:i], e.st.Presets[i+1:]...)
			e.touch()
			return true
		}
	}
	return false
}

// key — чем схема адресуется снаружи: встроенная своим идентификатором,
// пользовательская — именем.
func (p Preset) key() string {
	if p.Builtin {
		return p.ID
	}
	return p.Name
}
