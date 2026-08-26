// Package layout описывает физическую раскладку клавиатуры.
package layout

// Key — одна клавиша. Координаты в «юнитах»: 1 юнит равен ширине обычной
// клавиши, поэтому раскладка масштабируется без пересчёта.
type Key struct {
	LED   string  `json:"led"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Label string  `json:"label"`
	Ev    string  `json:"ev"`    // имя кода evdev, пусто если клавиша не шлёт событий
	Index int     `json:"index"` // индекс светодиода в кадре
	// Shape задаёт нестандартный контур: "iso-enter" — Г-образный Enter,
	// у которого верхняя часть шире нижней. Пусто — обычный прямоугольник.
	Shape string `json:"shape,omitempty"`
}

func (k Key) CX() float64 { return k.X + k.W/2 }
func (k Key) CY() float64 { return k.Y + k.H/2 }

// Variant выбирает раскладку по имени.
func Variant(name string) []Key {
	if name == "iso" {
		return ISO
	}
	return ANSI
}

// Bounds — габариты раскладки в юнитах.
func Bounds(keys []Key) (w, h float64) {
	for _, k := range keys {
		if k.X+k.W > w {
			w = k.X + k.W
		}
		if k.Y+k.H > h {
			h = k.Y + k.H
		}
	}
	return
}

// EvIndex — соответствие имени кода evdev индексу светодиода.
func EvIndex(keys []Key) map[string]int {
	m := make(map[string]int, len(keys))
	for _, k := range keys {
		if k.Ev != "" {
			m[k.Ev] = k.Index
		}
	}
	return m
}
