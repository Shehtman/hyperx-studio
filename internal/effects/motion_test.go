package effects

import (
	"math"
	"testing"

	"hyperx-studio/internal/keyboard"
)

// frameDiff — средняя разница двух кадров по каналу на клавишу. Ноль означает
// полностью замерший эффект.
func frameDiff(a, b []keyboard.RGB, ctx *Context) float64 {
	var sum float64
	for _, k := range ctx.Keys {
		x, y := a[k.Index], b[k.Index]
		sum += math.Abs(float64(x.R)-float64(y.R)) +
			math.Abs(float64(x.G)-float64(y.G)) +
			math.Abs(float64(x.B)-float64(y.B))
	}
	return sum / float64(len(ctx.Keys)*3)
}

func renderAt(e Effect, ctx *Context, t float64) []keyboard.RGB {
	buf := make([]keyboard.RGB, keyboard.LEDCount)
	e.Render(t, buf, ctx, false)
	return buf
}

// Волна обязана заметно двигаться на обычной скорости 1x при любом масштабе.
//
// Раньше время делилось на масштаб, и на пяти полосах картинка трогалась с
// места только к 4x — на 1x эффект выглядел статичным.
func TestWavesMoveAtNormalSpeed(t *testing.T) {
	for _, eff := range []Effect{
		&ColorWave{base{"colorwave", "", false}},
		&RainbowWave{base{"rainbow", "", false}},
	} {
		for _, scale := range []float64{1, 3, 5, 8} {
			ctx, _ := testCtx()
			ctx.P.Speed = 1
			ctx.P.Scale = scale

			// Смотрим несколько моментов: на крупном масштабе период
			// короткий, и одна выбранная точка может случайно совпасть с
			// целым периодом — картинка сойдётся сама с собой.
			base := renderAt(eff, ctx, 0)
			var best float64
			for step := 1; step <= 5; step++ {
				if d := frameDiff(base, renderAt(eff, ctx, float64(step)*0.1), ctx); d > best {
					best = d
				}
			}
			if best < 12 {
				t.Errorf("%s на масштабе %.0f почти не движется за полсекунды: разница %.1f",
					eff.ID(), scale, best)
			}
		}
	}
}

// Масштаб задаёт число полос, скорость — их ход. Смешивать их нельзя:
// иначе крупный масштаб тормозит анимацию.
func TestWaveTravelIndependentOfScale(t *testing.T) {
	// Ищем сдвиг картинки во времени: при каком t кадр совпадёт с кадром,
	// сдвинутым на четверть ширины клавиатуры.
	travelAt := func(scale float64) float64 {
		ctx, _ := testCtx()
		ctx.P.Speed = 1
		ctx.P.Scale = scale
		ctx.P.Angle = 0
		eff := &ColorWave{base{"colorwave", "", false}}

		base := renderAt(eff, ctx, 0)
		best, bestT := math.Inf(1), 0.0
		for step := 1; step <= 400; step++ {
			tt := float64(step) * 0.01
			if d := frameDiff(base, renderAt(eff, ctx, tt), ctx); d < best {
				best, bestT = d, tt
			}
		}
		return bestT
	}

	// Период повторения зависит от масштаба, а вот скорость хода — нет:
	// на вдвое большем масштабе картинка повторяется вдвое чаще.
	one, two := travelAt(2), travelAt(4)
	if one <= 0 || two <= 0 {
		t.Fatalf("не удалось измерить период: %.2f и %.2f", one, two)
	}
	ratio := one / two
	if ratio < 1.6 || ratio > 2.4 {
		t.Errorf("период на масштабах 2 и 4 относится как %.2f, ожидалось около 2: "+
			"скорость хода зависит от масштаба", ratio)
	}
}

// Дождь должен падать с одинаковой скоростью независимо от частоты кадров.
// Раньше капля смещалась на фиксированную величину за кадр, и на 30 кадрах
// дождь шёл вдвое медленнее, чем на 60.
func TestRainFallsAtSameSpeedAtAnyFPS(t *testing.T) {
	fall := func(dt float64) float64 {
		ctx, _ := testCtx()
		ctx.P.Speed = 1
		ctx.P.Density = 0 // новые капли не нужны, следим за одной
		ctx.Dt = dt

		e := &Rain{base: base{"rain", "", false}}
		e.drops = [][2]float64{{ctx.W / 2, 0}}

		buf := make([]keyboard.RGB, keyboard.LEDCount)
		for elapsed := 0.0; elapsed < 1.0-1e-9; elapsed += dt {
			e.Render(elapsed, buf, ctx, false)
		}
		if len(e.drops) == 0 {
			t.Fatal("капля пропала раньше времени")
		}
		return e.drops[0][1]
	}

	at60, at30 := fall(1.0/60), fall(1.0/30)
	if math.Abs(at60-at30) > 0.2 {
		t.Errorf("за секунду капля прошла %.2f при 60 кадрах и %.2f при 30", at60, at30)
	}
	if at60 < 1 {
		t.Errorf("за секунду капля прошла всего %.2f ряда — дождь стоит на месте", at60)
	}
}

// Шаг времени приходит из движка и на первом кадре может оказаться нулевым.
// Эффекты не должны на этом падать или делить на ноль.
func TestEffectsSurviveZeroTimeStep(t *testing.T) {
	for _, d := range All {
		ctx, _ := testCtx()
		ctx.Dt = 0
		buf := make([]keyboard.RGB, keyboard.LEDCount)
		eff := d.Make()
		eff.Render(0, buf, ctx, false)
		eff.Render(0.016, buf, ctx, false)
	}
}

// Звуковой эффект обязан шевелиться и в тишине.
//
// Захват может идти при полной тишине — когда просто ничего не играет.
// Раньше эффекты проверяли только «работает ли захват», получали нулевые
// полосы и рисовали неподвижный, а «Эквалайзер» и вовсе пустой экран.
// Пользователь видел это как зависшую программу.
func TestAudioEffectsAnimateOnSilence(t *testing.T) {
	for _, d := range All {
		if !d.Audio {
			continue
		}
		ctx, _ := testCtx()
		ctx.Dt = 1.0 / 60
		// захват идёт, сигнала нет
		ctx.Sound = Sound{On: true, Bands: make([]float64, 12)}

		eff := d.Make()
		base := renderAt(eff, ctx, 0)
		var best float64
		var litN int
		for step := 1; step <= 30; step++ {
			f := renderAt(eff, ctx, float64(step)/60)
			if v := frameDiff(base, f, ctx); v > best {
				best = v
			}
			litN = 0
			for _, k := range ctx.Keys {
				if f[k.Index] != (keyboard.RGB{}) {
					litN++
				}
			}
		}
		if best < 3 {
			t.Errorf("%s в тишине почти не движется: разница %.1f", d.ID, best)
		}
		if litN == 0 {
			t.Errorf("%s в тишине не зажигает ни одной клавиши", d.ID)
		}
	}
}

// Тишина и сигнал должны выглядеть по-разному, иначе заставка подменяет
// собой работу эффекта.
func TestAudioEffectsReactToSound(t *testing.T) {
	for _, d := range All {
		if !d.Audio {
			continue
		}
		mk := func(level float64) []keyboard.RGB {
			ctx, _ := testCtx()
			ctx.Dt = 1.0 / 60
			bands := make([]float64, 12)
			for i := range bands {
				bands[i] = level
			}
			ctx.Sound = Sound{On: true, Level: level, Beat: level, Bands: bands}
			return renderAt(d.Make(), ctx, 0.5)
		}
		ctx, _ := testCtx()
		if frameDiff(mk(0), mk(0.7), ctx) < 3 {
			t.Errorf("%s одинаково выглядит в тишине и под музыку", d.ID)
		}
	}
}
