package effects

import (
	"math"
	"testing"

	"hyperx-studio/internal/keyboard"
	"hyperx-studio/internal/layout"
)

func testCtx() (*Context, layout.Key) {
	keys := layout.Variant("ansi")
	w, h := layout.Bounds(keys)
	p := DefaultParams()
	ctx := &Context{Keys: keys, W: w, H: h, P: p, PerKey: map[int]RGB{}}
	var hit layout.Key
	for _, k := range keys {
		if k.LED == "Key: G" {
			hit = k
		}
	}
	ctx.Hits = []Hit{{Index: hit.Index, CX: hit.CX(), CY: hit.CY(), Born: 0, Seed: 0.5}}
	return ctx, hit
}

func lit(buf []RGB, ctx *Context) []layout.Key {
	var out []layout.Key
	for _, k := range ctx.Keys {
		if buf[k.Index] != (RGB{}) {
			out = append(out, k)
		}
	}
	return out
}

// Волна должна расходиться кольцом, а не светить одной клавишей.
func TestRippleExpands(t *testing.T) {
	ctx, hit := testCtx()
	ctx.P.Rainbow = false
	e := &Ripple{}
	var radii []float64
	for _, at := range []float64{0.10, 0.35, 0.70, 1.10} {
		buf := make([]RGB, keyboard.LEDCount)
		e.Render(at, buf, ctx, true)
		on := lit(buf, ctx)
		if len(on) < 2 {
			t.Fatalf("t=%.2f: горит %d клавиш, ожидалась не одна", at, len(on))
		}
		var sum float64
		for _, k := range on {
			sum += math.Hypot(k.CX()-hit.CX(), (k.CY()-hit.CY())*1.6)
		}
		radii = append(radii, sum/float64(len(on)))
	}
	for i := 1; i < len(radii); i++ {
		if radii[i] <= radii[i-1] {
			t.Fatalf("радиус не растёт: %v", radii)
		}
	}

	// после затухания не должно остаться ничего
	buf := make([]RGB, keyboard.LEDCount)
	e.Render(ctx.P.ReactFade+0.2, buf, ctx, true)
	if n := len(lit(buf, ctx)); n != 0 {
		t.Fatalf("после затухания горит %d клавиш", n)
	}
}

// Хит из чужой шкалы времени (абсолютной) не должен светить вечно.
func TestRippleIgnoresFutureHit(t *testing.T) {
	ctx, _ := testCtx()
	ctx.Hits[0].Born = 1e6
	buf := make([]RGB, keyboard.LEDCount)
	(&Ripple{}).Render(0.3, buf, ctx, true)
	if n := len(lit(buf, ctx)); n != 0 {
		t.Fatalf("хит из будущего зажёг %d клавиш", n)
	}
}

// Вспышка гаснет монотонно и только на нажатой клавише.
func TestFlashFades(t *testing.T) {
	ctx, hit := testCtx()
	ctx.P.Rainbow = false
	e := &ReactiveFlash{}
	prev := 256
	for _, at := range []float64{0.05, 0.4, 0.8, 1.2} {
		buf := make([]RGB, keyboard.LEDCount)
		e.Render(at, buf, ctx, true)
		on := lit(buf, ctx)
		if len(on) != 1 || on[0].Index != hit.Index {
			t.Fatalf("t=%.2f: горит %d клавиш", at, len(on))
		}
		c := buf[hit.Index]
		v := int(max3(c.R, c.G, c.B))
		if v >= prev {
			t.Fatalf("t=%.2f: яркость %d не убывает (было %d)", at, v, prev)
		}
		prev = v
	}
}

func max3(a, b, c uint8) uint8 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

// Слой поверх не должен затирать основной эффект.
func TestOverlayKeepsBackground(t *testing.T) {
	ctx, hit := testCtx()
	ctx.P.Color1 = RGB{R: 40, B: 60}
	buf := make([]RGB, keyboard.LEDCount)
	(&Static{}).Render(0, buf, ctx, false)

	tmp := make([]RGB, keyboard.LEDCount)
	(&ReactiveFlash{}).Render(0.05, tmp, ctx, true)
	for _, k := range ctx.Keys {
		if tmp[k.Index] != (RGB{}) {
			buf[k.Index] = Add(buf[k.Index], tmp[k.Index])
		}
	}
	for _, k := range ctx.Keys {
		if k.Index == hit.Index {
			continue
		}
		if buf[k.Index] != ctx.P.Color1 {
			t.Fatalf("слой затёр фон на клавише %s", k.LED)
		}
	}
	if buf[hit.Index] == ctx.P.Color1 {
		t.Fatal("нажатая клавиша не подсветилась")
	}
}

// Все эффекты обязаны укладываться в границы буфера и давать корректный цвет.
func TestAllEffectsRender(t *testing.T) {
	ctx, _ := testCtx()
	for _, d := range All {
		e := d.Make()
		for i := 0; i < 20; i++ {
			buf := make([]RGB, keyboard.LEDCount)
			e.Render(float64(i)/30, buf, ctx, false)
		}
	}
}
