// Генератор картинок для README.
//
// Клавиатура на баннере рисуется из той же раскладки, что и в программе,
// поэтому картинка не может разойтись с действительностью.
package main

import (
	"fmt"
	"os"
	"strings"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/layout"
)

const (
	unit   = 34.0 // пикселей на юнит клавиатуры
	gap    = 3.0
	frames = 24 // кадров в цикле анимации
	cycle  = "7s"
)

func main() {
	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	write("docs/banner.svg", banner())
	write("docs/panel.svg", panel())
	fmt.Println("готово: docs/banner.svg, docs/panel.svg")
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("  %s — %d КБ\n", path, len(body)/1024)
}

// waveColors возвращает кадры радужной волны для клавиши.
func waveColors(k layout.Key, w float64) []string {
	out := make([]string, 0, frames+1)
	for i := 0; i <= frames; i++ {
		phase := float64(i) / frames
		c := effects.HSV(k.CX()/w-phase, 0.85, 1)
		out = append(out, fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
	}
	return out
}

func banner() string {
	keys := layout.ANSI
	w, h := layout.Bounds(keys)
	boardW, boardH := w*unit, h*unit
	padX, padTop := 40.0, 118.0
	width := boardW + padX*2
	height := boardH + padTop + 44

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" role="img" aria-label="HyperX Studio">`,
		width, height, width, height)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="14" fill="#0d0d0d"/>`, width, height)

	// заголовок
	fmt.Fprintf(&b, `<text x="%.0f" y="58" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="34" font-weight="700" fill="#f0f0f0">HyperX Studio</text>`, padX)
	fmt.Fprintf(&b, `<text x="%.0f" y="88" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="15" fill="#8a8a8a">Per-key RGB for HyperX Alloy Origins on Linux — one static binary, no daemons</text>`, padX)

	for _, k := range keys {
		x := padX + k.X*unit + gap/2
		y := padTop + k.Y*unit + gap/2
		kw := k.W*unit - gap
		kh := k.H*unit - gap
		vals := strings.Join(waveColors(k, w), ";")
		fmt.Fprintf(&b,
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="#1c1c1c">`+
				`<animate attributeName="fill" values="%s" dur="%s" repeatCount="indefinite"/></rect>`,
			x, y, kw, kh, vals, cycle)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// panel рисует облик боковой панели: то же расположение и те же цвета,
// что в самой программе.
func panel() string {
	const (
		width  = 300.0
		height = 466.0
	)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" role="img" aria-label="Control panel">`,
		width, height, width, height)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="10" fill="#181818"/>`, width, height)

	y := 26.0
	label := func(text string) {
		fmt.Fprintf(&b, `<text x="18" y="%.0f" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#8a8a8a">%s</text>`, y+12, text)
	}
	field := func(text string) {
		fmt.Fprintf(&b, `<rect x="104" y="%.0f" width="178" height="24" rx="4" fill="#202020" stroke="#2c2c2c"/>`, y)
		fmt.Fprintf(&b, `<text x="114" y="%.0f" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#e6e6e6">%s</text>`, y+16, text)
	}
	slider := func(frac float64, val string) {
		fmt.Fprintf(&b, `<rect x="104" y="%.0f" width="140" height="4" rx="2" fill="#2c2c2c"/>`, y+10)
		fmt.Fprintf(&b, `<rect x="104" y="%.0f" width="%.0f" height="4" rx="2" fill="#22c55e"/>`, y+10, 140*frac)
		fmt.Fprintf(&b, `<circle cx="%.0f" cy="%.0f" r="6" fill="#22c55e"/>`, 104+140*frac, y+12)
		fmt.Fprintf(&b, `<text x="252" y="%.0f" font-family="ui-monospace,Consolas,monospace" font-size="11" fill="#8a8a8a">%s</text>`, y+16, val)
	}
	rule := func() {
		fmt.Fprintf(&b, `<rect x="0" y="%.0f" width="%.0f" height="1" fill="#2c2c2c"/>`, y, width)
		y += 14
	}

	label("Effect")
	field("Rainbow wave")
	y += 32
	label("Overlay")
	field("Key ripple")
	y += 40
	rule()
	label("Brightness")
	slider(1, "100%")
	y += 28
	label("Speed")
	slider(0.42, "1.05×")
	y += 28
	label("Angle")
	slider(0.16, "58°")
	y += 40
	rule()
	fmt.Fprintf(&b, `<text x="18" y="%.0f" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="11" fill="#8a8a8a">Keypress reaction</text>`, y+10)
	y += 24
	label("Fade")
	slider(0.36, "1.60 s")
	y += 34

	// палитра
	sw := []string{"#ff0000", "#ff6a00", "#ffd400", "#7dff00", "#00ff3c", "#00ffb3", "#00e5ff", "#0077ff"}
	for i, c := range sw {
		fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="28" height="24" rx="3" fill="%s"/>`,
			18+float64(i)*33, y, c)
	}
	y += 38
	for _, btn := range []struct {
		x float64
		t string
	}{{18, "All"}, {110, "Clear"}, {202, "Invert"}} {
		fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="80" height="26" rx="4" fill="#202020" stroke="#2c2c2c"/>`, btn.x, y)
		fmt.Fprintf(&b, `<text x="%.0f" y="%.0f" text-anchor="middle" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="12" fill="#e6e6e6">%s</text>`,
			btn.x+40, y+17, btn.t)
	}

	b.WriteString(`</svg>`)
	return b.String()
}
