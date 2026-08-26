package keyboard

import "testing"

// Число светодиодов должно совпадать с тем, что отдаёт устройство.
// Если оно разъедется, вся раскладка молча поедет.
func TestLEDCount(t *testing.T) {
	if LEDCount != 107 {
		t.Fatalf("светодиодов %d, ожидалось 107", LEDCount)
	}
	if len(deadSlots) != slotCount-LEDCount {
		t.Fatalf("пропусков %d, а разница слотов и светодиодов %d",
			len(deadSlots), slotCount-LEDCount)
	}
}

func TestHexRoundTrip(t *testing.T) {
	in := []RGB{{R: 0, G: 0, B: 0}, {R: 255, G: 40, B: 0}, {R: 1, G: 2, B: 3}}
	got := Hex(in)
	want := "000000" + "ff2800" + "010203"
	if got != want {
		t.Fatalf("Hex = %q, ожидалось %q", got, want)
	}
}

func TestColorJSON(t *testing.T) {
	var c RGB
	if err := c.UnmarshalJSON([]byte(`"#3b1bff"`)); err != nil {
		t.Fatal(err)
	}
	if c != (RGB{R: 0x3b, G: 0x1b, B: 0xff}) {
		t.Fatalf("разобрано как %+v", c)
	}
	out, err := c.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `"#3b1bff"` {
		t.Fatalf("сериализовано как %s", out)
	}
	if err := c.UnmarshalJSON([]byte(`"3b1bff"`)); err == nil {
		t.Fatal("цвет без решётки должен отвергаться")
	}
}

// Кадр обязан раскладываться по слотам с пропусками ровно в тех местах,
// где у клавиатуры нет светодиода.
func TestSlotExpansion(t *testing.T) {
	colors := make([]RGB, LEDCount)
	for i := range colors {
		colors[i] = RGB{R: uint8(i + 1)}
	}
	slots := make([]RGB, slotCount)
	src := 0
	for i := 0; i < slotCount; i++ {
		if deadSlots[i] {
			continue
		}
		slots[i] = colors[src]
		src++
	}
	if src != LEDCount {
		t.Fatalf("уложено %d цветов из %d", src, LEDCount)
	}
	for i := range slots {
		if deadSlots[i] && slots[i] != (RGB{}) {
			t.Fatalf("в пропуск %d попал цвет", i)
		}
		if !deadSlots[i] && slots[i] == (RGB{}) {
			t.Fatalf("слот %d остался пустым", i)
		}
	}
}

// Кадр обязан начинаться с переключения в прямой режим. Без этого пакета
// клавиатура возвращается к собственному зашитому эффекту и перебивает
// то, что рисует программа.
func TestFrameStartsWithDirectModeSwitch(t *testing.T) {
	packets, err := buildFrame(make([]RGB, LEDCount))
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 9 {
		t.Fatalf("пакетов %d, ожидалось 9 (переключение + 8 с цветами)", len(packets))
	}
	head := packets[0]
	if head[0x01] != 0x04 || head[0x02] != 0xF2 || head[0x09] != 0x09 {
		t.Fatalf("первый пакет не переключает режим: % x", head[:10])
	}
	for i, p := range packets {
		if len(p) != packetSize {
			t.Fatalf("пакет %d длиной %d, ожидалось %d", i, len(p), packetSize)
		}
	}
}

// Цвета должны попадать ровно в свои слоты, а пропуски оставаться нулевыми.
func TestFramePlacesColors(t *testing.T) {
	colors := make([]RGB, LEDCount)
	colors[0] = RGB{R: 0x11, G: 0x22, B: 0x33}
	colors[LEDCount-1] = RGB{R: 0xAA, G: 0xBB, B: 0xCC}

	packets, err := buildFrame(colors)
	if err != nil {
		t.Fatal(err)
	}
	read := func(slot int) RGB {
		p := packets[1+slot/colorsPerPacket]
		off := (slot%colorsPerPacket)*4 + 1
		if p[off] != 0x81 {
			t.Fatalf("слот %d без маркера: %#x", slot, p[off])
		}
		return RGB{R: p[off+1], G: p[off+2], B: p[off+3]}
	}

	if got := read(0); got != colors[0] {
		t.Fatalf("первый светодиод уехал: %+v", got)
	}
	last := slotCount - 1
	for deadSlots[last] {
		last--
	}
	if got := read(last); got != colors[LEDCount-1] {
		t.Fatalf("последний светодиод уехал: %+v", got)
	}
	for slot := range deadSlots {
		if got := read(slot); got != (RGB{}) {
			t.Fatalf("в пропуск %d попал цвет %+v", slot, got)
		}
	}
}
