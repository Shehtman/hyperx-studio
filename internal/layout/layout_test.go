package layout

import "testing"

func TestVariantsBindToRealLEDs(t *testing.T) {
	byName := map[string]int{}
	for i, n := range LEDNames {
		byName[n] = i
	}
	for _, v := range []string{"ansi", "iso"} {
		keys := Variant(v)
		if len(keys) < 100 {
			t.Fatalf("%s: клавиш всего %d", v, len(keys))
		}
		seen := map[int]string{}
		for _, k := range keys {
			want, ok := byName[k.LED]
			if !ok {
				t.Fatalf("%s: %q нет среди светодиодов устройства", v, k.LED)
			}
			if k.Index != want {
				t.Fatalf("%s: %q привязан к %d, а должен к %d", v, k.LED, k.Index, want)
			}
			if prev, dup := seen[k.Index]; dup {
				t.Fatalf("%s: индекс %d занят и %q, и %q", v, k.Index, prev, k.LED)
			}
			seen[k.Index] = k.LED
		}
		if w, h := Bounds(keys); w != 22.5 || h != 6.25 {
			t.Fatalf("%s: габарит %.2fx%.2f", v, w, h)
		}
	}
}

func TestEvIndexUnique(t *testing.T) {
	keys := Variant("ansi")
	m := EvIndex(keys)
	if len(m) < 100 {
		t.Fatalf("кодов evdev всего %d", len(m))
	}
	if _, ok := m["KEY_SPACE"]; !ok {
		t.Fatal("пробел не сопоставлен коду evdev")
	}
}

// В ряду не должно быть щелей там, где клавиши идут вплотную.
// Щель между "]" и Enter означала, что Г-образный Enter нарисован
// прямоугольником и не накрывает свою верхнюю часть.
func TestNoGapBeforeEnter(t *testing.T) {
	for _, v := range []string{"ansi", "iso"} {
		var rowEnd float64
		var enterStart float64 = -1
		for _, k := range Variant(v) {
			if k.Y != 2.25 {
				continue
			}
			if k.LED == "Key: ]" {
				rowEnd = k.X + k.W
			}
			if k.LED == "Key: Enter" || k.LED == "Key: \\ (ANSI)" {
				enterStart = k.X
			}
		}
		if enterStart < 0 {
			t.Fatalf("%s: в ряду 2 нет ни Enter, ни обратного слеша", v)
		}
		if enterStart-rowEnd > 0.01 {
			t.Fatalf("%s: щель %.2f между \"]\" (%.2f) и следующей клавишей (%.2f)",
				v, enterStart-rowEnd, rowEnd, enterStart)
		}
	}
}

// Г-образный Enter обязан быть помечен, иначе интерфейс нарисует его
// прямоугольником и накроет соседнюю клавишу.
func TestISOEnterMarkedLShaped(t *testing.T) {
	for _, k := range Variant("iso") {
		if k.LED == "Key: Enter" {
			if k.Shape != "iso-enter" {
				t.Fatalf("Enter в ISO без пометки формы: %q", k.Shape)
			}
			if k.W != 1.5 || k.H != 2 {
				t.Fatalf("Enter в ISO размером %.2fx%.2f", k.W, k.H)
			}
			return
		}
	}
	t.Fatal("Enter в раскладке ISO не найден")
}
