// Package effects считает кадры подсветки.
//
// Клавиатура умеет только прямое управление, аппаратных эффектов у неё нет.
// Поэтому всё рисуется здесь и уходит на устройство кадр за кадром.
package effects

import (
	"math"
	"math/rand"

	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/layout"
)

type RGB = keyboard.RGB

// Hit — нажатие клавиши. Born измеряется в той же шкале, что и время кадра.
type Hit struct {
	Index int
	CX    float64
	CY    float64
	Born  float64
	Seed  float64
}

// Params — настройки, приходящие из интерфейса.
type Params struct {
	Speed      float64 `json:"speed"`
	Brightness float64 `json:"brightness"`
	Angle      float64 `json:"angle"`
	Scale      float64 `json:"scale"`
	Density    float64 `json:"density"`
	Length     int     `json:"length"`
	Color1     RGB     `json:"color1"`
	Color2     RGB     `json:"color2"`
	ReactColor RGB     `json:"reactColor"`
	ReactSpeed float64 `json:"reactSpeed"`
	ReactFade  float64 `json:"reactFade"`
	Rainbow    bool    `json:"rainbow"`
}

func DefaultParams() Params {
	return Params{
		Speed: 1, Brightness: 1, Angle: 0, Scale: 1, Density: 0.5, Length: 12,
		Color1: RGB{R: 255, G: 40}, Color2: RGB{G: 60, B: 255},
		ReactColor: RGB{G: 200, B: 255}, ReactSpeed: 1, ReactFade: 1.6,
		Rainbow: true,
	}
}

// Context — всё, что нужно эффекту для отрисовки кадра.
type Context struct {
	Keys      []layout.Key
	W, H      float64
	Hits      []Hit
	P         Params
	PerKey    map[int]RGB
	Selection map[int]bool

	snakeOrder []int
}

// ── работа с цветом ───────────────────────────────────────────────────

func HSV(h, s, v float64) RGB {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
	}
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return RGB{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255)}
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

func Mix(a, b RGB, t float64) RGB {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return RGB{
		R: clamp8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: clamp8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: clamp8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

func Scale(c RGB, f float64) RGB {
	return RGB{R: clamp8(float64(c.R) * f), G: clamp8(float64(c.G) * f), B: clamp8(float64(c.B) * f)}
}

func Add(a, b RGB) RGB {
	return RGB{
		R: clamp8(float64(a.R) + float64(b.R)),
		G: clamp8(float64(a.G) + float64(b.G)),
		B: clamp8(float64(a.B) + float64(b.B)),
	}
}

func clear(buf []RGB, ctx *Context) {
	for _, k := range ctx.Keys {
		buf[k.Index] = RGB{}
	}
}

// ── эффекты ───────────────────────────────────────────────────────────

// Effect рисует один кадр в buf.
type Effect interface {
	ID() string
	Name() string
	Reactive() bool
	Render(t float64, buf []RGB, ctx *Context, asOverlay bool)
}

type base struct {
	id, name string
	reactive bool
}

func (b base) ID() string     { return b.id }
func (b base) Name() string   { return b.name }
func (b base) Reactive() bool { return b.reactive }

type Static struct{ base }

func (e *Static) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	for _, k := range ctx.Keys {
		if c, ok := ctx.PerKey[k.Index]; ok {
			buf[k.Index] = c
		} else {
			buf[k.Index] = ctx.P.Color1
		}
	}
}

type Breathing struct{ base }

func (e *Breathing) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	f := (math.Sin(t*ctx.P.Speed*2) + 1) / 2
	f *= f
	c := Scale(ctx.P.Color1, 0.05+0.95*f)
	for _, k := range ctx.Keys {
		buf[k.Index] = c
	}
}

type SpectrumCycle struct{ base }

func (e *SpectrumCycle) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	c := HSV(t*ctx.P.Speed*0.15, 1, 1)
	for _, k := range ctx.Keys {
		buf[k.Index] = c
	}
}

type RainbowWave struct{ base }

func (e *RainbowWave) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	a := ctx.P.Angle * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a)
	for _, k := range ctx.Keys {
		proj := (k.CX()*dx + k.CY()*dy) / math.Max(ctx.W, 1)
		buf[k.Index] = HSV(proj*ctx.P.Scale-t*ctx.P.Speed*0.2, 1, 1)
	}
}

type ColorWave struct{ base }

func (e *ColorWave) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	a := ctx.P.Angle * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a)
	for _, k := range ctx.Keys {
		proj := (k.CX()*dx + k.CY()*dy) / math.Max(ctx.W, 1)
		f := (math.Sin((proj*ctx.P.Scale-t*ctx.P.Speed*0.2)*2*math.Pi) + 1) / 2
		buf[k.Index] = Mix(ctx.P.Color1, ctx.P.Color2, f)
	}
}

type Gradient struct{ base }

func (e *Gradient) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	a := ctx.P.Angle * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a)
	lo, hi := math.Inf(1), math.Inf(-1)
	vals := make([]float64, len(ctx.Keys))
	for i, k := range ctx.Keys {
		v := k.CX()*dx + k.CY()*dy
		vals[i] = v
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	for i, k := range ctx.Keys {
		buf[k.Index] = Mix(ctx.P.Color1, ctx.P.Color2, (vals[i]-lo)/span)
	}
}

type Twinkle struct {
	base
	stars map[int]float64
}

func (e *Twinkle) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	if e.stars == nil {
		e.stars = map[int]float64{}
	}
	if rand.Float64() < ctx.P.Density*0.5 && len(ctx.Keys) > 0 {
		k := ctx.Keys[rand.Intn(len(ctx.Keys))]
		if _, ok := e.stars[k.Index]; !ok {
			e.stars[k.Index] = t
		}
	}
	clear(buf, ctx)
	life := 1.5 / math.Max(ctx.P.Speed, 0.05)
	for idx, born := range e.stars {
		age := (t - born) / life
		if age >= 1 {
			delete(e.stars, idx)
			continue
		}
		buf[idx] = Scale(ctx.P.Color1, math.Sin(age*math.Pi))
	}
}

type Rain struct {
	base
	drops [][2]float64
}

func (e *Rain) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	if rand.Float64() < ctx.P.Density*0.4 {
		e.drops = append(e.drops, [2]float64{rand.Float64() * ctx.W, -1})
	}
	clear(buf, ctx)
	alive := e.drops[:0]
	for _, d := range e.drops {
		d[1] += ctx.P.Speed * 0.12
		if d[1] > ctx.H+2 {
			continue
		}
		alive = append(alive, d)
		for _, k := range ctx.Keys {
			if math.Abs(k.CX()-d[0]) > 1 {
				continue
			}
			dy := k.CY() - d[1]
			if dy > -0.5 && dy < 2.5 {
				f := 1.0
				if dy > 0 {
					f = math.Max(0, 1-math.Abs(dy)/2.5)
				}
				f *= math.Max(0, 1-math.Abs(k.CX()-d[0]))
				buf[k.Index] = Add(buf[k.Index], Scale(ctx.P.Color1, f))
			}
		}
	}
	e.drops = alive
}

type Fire struct{ base }

func (e *Fire) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	s := ctx.P.Speed
	for _, k := range ctx.Keys {
		n := math.Sin(k.CX()*1.7+t*s*3)*0.5 +
			math.Sin(k.CX()*0.6-t*s*2.1)*0.3 +
			math.Sin(k.CX()*3.3+t*s*4.7)*0.2
		heat := k.CY()/math.Max(ctx.H, 1) + n*0.35
		heat = math.Max(0, math.Min(1, heat))
		var c RGB
		switch {
		case heat < 0.45:
			c = Mix(RGB{R: 60}, RGB{R: 255, G: 40}, heat/0.45)
		case heat < 0.8:
			c = Mix(RGB{R: 255, G: 40}, RGB{R: 255, G: 170}, (heat-0.45)/0.35)
		default:
			c = Mix(RGB{R: 255, G: 170}, RGB{R: 255, G: 245, B: 190}, (heat-0.8)/0.2)
		}
		buf[k.Index] = c
	}
}

type Snake struct{ base }

func (e *Snake) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	if ctx.snakeOrder == nil || len(ctx.snakeOrder) != len(ctx.Keys) {
		ord := make([]int, len(ctx.Keys))
		for i := range ord {
			ord[i] = i
		}
		// построчно слева направо
		for i := 1; i < len(ord); i++ {
			for j := i; j > 0; j-- {
				a, b := ctx.Keys[ord[j-1]], ctx.Keys[ord[j]]
				if a.CY() > b.CY() || (a.CY() == b.CY() && a.CX() > b.CX()) {
					ord[j-1], ord[j] = ord[j], ord[j-1]
				} else {
					break
				}
			}
		}
		ctx.snakeOrder = ord
	}
	clear(buf, ctx)
	n := len(ctx.snakeOrder)
	if n == 0 {
		return
	}
	tail := ctx.P.Length
	if tail < 1 {
		tail = 1
	}
	head := math.Mod(t*ctx.P.Speed*18, float64(n))
	for j := 0; j < tail; j++ {
		pos := (int(head)-j)%n + n
		k := ctx.Keys[ctx.snakeOrder[pos%n]]
		buf[k.Index] = Scale(ctx.P.Color1, 1-float64(j)/float64(tail))
	}
}

type Ripple struct{ base }

func (e *Ripple) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	clear(buf, ctx)
	fade := math.Max(0.15, ctx.P.ReactFade)
	const ring = 1.7
	for _, h := range ctx.Hits {
		age := t - h.Born
		if age < 0 || age > fade {
			continue
		}
		r := age * ctx.P.ReactSpeed * 12
		f0 := 1 - age/fade
		c := ctx.P.ReactColor
		if ctx.P.Rainbow {
			c = HSV(h.Seed, 1, 1)
		}
		for _, k := range ctx.Keys {
			d := math.Hypot(k.CX()-h.CX, (k.CY()-h.CY)*1.6)
			w := math.Abs(d - r)
			if w < ring {
				buf[k.Index] = Add(buf[k.Index], Scale(c, (1-w/ring)*f0))
			}
		}
	}
}

type ReactiveFlash struct{ base }

func (e *ReactiveFlash) Render(t float64, buf []RGB, ctx *Context, asOverlay bool) {
	bg := RGB{}
	if !asOverlay {
		bg = ctx.P.Color2
	}
	for _, k := range ctx.Keys {
		buf[k.Index] = bg
	}
	fade := math.Max(0.1, ctx.P.ReactFade)
	for _, h := range ctx.Hits {
		age := t - h.Born
		if age < 0 || age > fade {
			continue
		}
		f := 1 - age/fade
		c := ctx.P.ReactColor
		if ctx.P.Rainbow {
			c = HSV(h.Seed, 1, 1)
		}
		if asOverlay {
			buf[h.Index] = Scale(c, f)
		} else {
			buf[h.Index] = Mix(c, bg, 1-f)
		}
	}
}

// New создаёт эффект по идентификатору.
func New(id string) Effect {
	for _, d := range All {
		if d.ID == id {
			return d.Make()
		}
	}
	return &Static{base{"static", "Статика", false}}
}

// Descriptor описывает эффект для интерфейса.
type Descriptor struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Reactive bool          `json:"reactive"`
	Uses     []string      `json:"uses"`
	Make     func() Effect `json:"-"`
}

var All = []Descriptor{
	{"static", "Статика", false, []string{"color1"}, func() Effect { return &Static{base{"static", "Статика", false}} }},
	{"breathing", "Дыхание", false, []string{"speed", "color1"}, func() Effect { return &Breathing{base{"breathing", "Дыхание", false}} }},
	{"spectrum", "Перелив спектра", false, []string{"speed"}, func() Effect { return &SpectrumCycle{base{"spectrum", "Перелив спектра", false}} }},
	{"rainbow", "Радужная волна", false, []string{"speed", "angle", "scale"}, func() Effect { return &RainbowWave{base{"rainbow", "Радужная волна", false}} }},
	{"colorwave", "Волна двух цветов", false, []string{"speed", "color1", "color2", "angle", "scale"}, func() Effect { return &ColorWave{base{"colorwave", "Волна двух цветов", false}} }},
	{"gradient", "Градиент", false, []string{"color1", "color2", "angle"}, func() Effect { return &Gradient{base{"gradient", "Градиент", false}} }},
	{"twinkle", "Звёзды", false, []string{"speed", "color1", "density"}, func() Effect { return &Twinkle{base: base{"twinkle", "Звёзды", false}} }},
	{"rain", "Дождь", false, []string{"speed", "color1", "density"}, func() Effect { return &Rain{base: base{"rain", "Дождь", false}} }},
	{"fire", "Огонь", false, []string{"speed"}, func() Effect { return &Fire{base{"fire", "Огонь", false}} }},
	{"snake", "Змейка", false, []string{"speed", "color1", "length"}, func() Effect { return &Snake{base{"snake", "Змейка", false}} }},
	{"ripple", "Волны от нажатий", true, nil, func() Effect { return &Ripple{base{"ripple", "Волны от нажатий", true}} }},
	{"flash", "Вспышка по нажатию", true, nil, func() Effect { return &ReactiveFlash{base{"flash", "Вспышка по нажатию", true}} }},
}
