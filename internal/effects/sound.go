package effects

import "math"

// Sound — то, что сейчас играет. Значения нормированы в 0..1: сам захват
// живёт в пакете audio, сюда попадает уже готовая картина.
type Sound struct {
	// On поднят, пока захват работает. Пока звука нет, эффекты рисуют
	// спокойный фон, а не чёрный экран.
	On bool
	// Level — общая громкость.
	Level float64
	// Beat — превышение баса над его обычным уровнем, то есть удар.
	Beat float64
	// Bands — полосы спектра от низких к высоким.
	Bands []float64
}

// band отдаёт полосу с защитой от пустого среза: захват может ещё не
// запуститься, а кадр уже рисуется.
func (s Sound) band(i int) float64 {
	if i < 0 || i >= len(s.Bands) {
		return 0
	}
	return s.Bands[i]
}

// gain применяет чувствительность и обрезает по единице.
func gain(v float64, ctx *Context) float64 {
	k := ctx.P.Sensitivity
	if k <= 0 {
		k = 1
	}
	return math.Max(0, math.Min(1, v*k))
}

// silenceFloor — ниже этого уровня считаем, что звука нет вовсе.
const silenceFloor = 0.02

// Live сообщает, что захват идёт и в нём есть сигнал.
//
// Одного только On мало. Захват прекрасно работает и в полной тишине —
// когда просто ничего не играет. Полосы тогда нулевые, и эффект, который
// смотрит лишь на On, рисует неподвижный тёмный экран. Со стороны это
// неотличимо от зависшей программы, и жалоба приходит именно такая.
func (s Sound) Live() bool {
	if !s.On {
		return false
	}
	if s.Level > silenceFloor || s.Beat > silenceFloor {
		return true
	}
	for _, b := range s.Bands {
		if b > silenceFloor {
			return true
		}
	}
	return false
}

// idleWave — что показывают звуковые эффекты, пока звука нет.
//
// Обязательно движущаяся картинка: неподвижная означала бы для человека,
// что программа встала. Приглушённая, чтобы отличаться от работы по звуку.
func idleWave(t float64, buf []RGB, ctx *Context) {
	w := math.Max(ctx.W, 1)
	for _, k := range ctx.Keys {
		pos := k.CX() / w
		f := (math.Sin((pos*1.5-t*0.35)*2*math.Pi) + 1) / 2
		buf[k.Index] = Scale(Mix(ctx.P.Color1, ctx.P.Color2, f), 0.10+0.16*f)
	}
}

// ── Эквалайзер ────────────────────────────────────────────────────────

// AudioBars — столбики спектра: низкие частоты слева, высокие справа,
// громкость полосы задаёт высоту столбика.
type AudioBars struct{ base }

func (e *AudioBars) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	n := len(ctx.Sound.Bands)
	if n == 0 || !ctx.Sound.Live() {
		idleWave(t, buf, ctx)
		return
	}
	clear(buf, ctx)
	w := math.Max(ctx.W, 1)
	h := math.Max(ctx.H, 1)

	for _, k := range ctx.Keys {
		// Клавиша принадлежит полосе по своему месту слева направо.
		pos := k.CX() / w
		if ctx.P.Reverse {
			pos = 1 - pos
		}
		i := int(pos * float64(n))
		if i >= n {
			i = n - 1
		}
		if i < 0 {
			i = 0
		}
		level := gain(ctx.Sound.band(i), ctx)

		// Столбик растёт снизу вверх: у клавиатуры ось Y направлена вниз,
		// поэтому нижние ряды — это большие Y.
		rowFromBottom := (h - k.CY()) / h
		if rowFromBottom > level {
			continue
		}
		// Цвет по высоте столбика: спокойный внизу, тревожный на пике.
		f := rowFromBottom / math.Max(level, 0.001)
		buf[k.Index] = Add(buf[k.Index], Mix(ctx.P.Color1, ctx.P.Color2, f))
	}
}

// ── Пульс ─────────────────────────────────────────────────────────────

// AudioPulse — вся клавиатура дышит в такт громкости.
type AudioPulse struct{ base }

func (e *AudioPulse) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	if !ctx.Sound.Live() {
		idleWave(t, buf, ctx)
		return
	}
	// Ведём по удару, а не по громкости: громкость у ровной музыки почти
	// всё время у потолка, и клавиатура просто горела бы одним цветом.
	// Немного громкости всё же подмешиваем, чтобы тихие места отличались
	// от полной тишины.
	level := gain(ctx.Sound.Beat*0.8+ctx.Sound.Level*0.2, ctx)
	c := Mix(ctx.P.Color1, ctx.P.Color2, level)
	// Немного света остаётся всегда: полностью гаснущая клавиатура между
	// ударами выглядит сломанной.
	c = Scale(c, 0.12+0.88*level)
	for _, k := range ctx.Keys {
		buf[k.Index] = c
	}
}

// ── Спектр по клавиатуре ──────────────────────────────────────────────

// AudioSpectrum красит клавиатуру целиком: слева низкие частоты, справа
// высокие, яркость каждой колонки — громкость её полосы.
type AudioSpectrum struct{ base }

func (e *AudioSpectrum) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	n := len(ctx.Sound.Bands)
	if n == 0 || !ctx.Sound.Live() {
		idleWave(t, buf, ctx)
		return
	}
	w := math.Max(ctx.W, 1)
	for _, k := range ctx.Keys {
		pos := k.CX() / w
		if ctx.P.Reverse {
			pos = 1 - pos
		}
		i := int(pos * float64(n))
		if i >= n {
			i = n - 1
		}
		level := gain(ctx.Sound.band(i), ctx)
		hue := pos*0.75 + t*0.02
		buf[k.Index] = Scale(HSV(hue, sat(ctx), 1), 0.06+0.94*level)
	}
}

// ── Волна от громкости ────────────────────────────────────────────────

// AudioWave — бегущая волна, у которой громкость управляет и яркостью,
// и скоростью хода.
type AudioWave struct {
	base
	phase float64
}

func (e *AudioWave) Render(t float64, buf []RGB, ctx *Context, _ bool) {
	if !ctx.Sound.Live() {
		idleWave(t, buf, ctx)
		return
	}
	level := gain(ctx.Sound.Level, ctx)

	// Фазу копим сами: скорость зависит от звука, и просто умножить время
	// на неё нельзя — картинка дёргалась бы назад при спаде громкости.
	e.phase += ctx.Dt * ctx.P.Speed * dir(ctx) * (0.15 + 1.2*level)

	a := ctx.P.Angle * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a)
	for _, k := range ctx.Keys {
		proj := (k.CX()*dx + k.CY()*dy) / math.Max(ctx.W, 1)
		f := (math.Sin((proj-e.phase)*ctx.P.Scale*2*math.Pi) + 1) / 2
		c := Mix(ctx.P.Color1, ctx.P.Color2, f)
		buf[k.Index] = Scale(c, 0.08+0.92*level)
	}
}
