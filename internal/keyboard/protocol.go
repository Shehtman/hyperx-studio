package keyboard

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Устройство принимает 126 позиций подсветки, из которых 19 физически
// отсутствуют — им всё равно нужно слать нули, иначе поедет вся раскладка.
const (
	slotCount       = 126
	colorsPerPacket = 16
	packetSize      = 65
)

var deadSlots = map[int]bool{
	23: true, 29: true, 41: true, 47: true, 59: true,
	70: true, 71: true, 87: true, 88: true, 93: true,
	99: true, 100: true, 102: true, 108: true,
	113: true, 114: true, 120: true, 123: true, 124: true,
}

// LEDCount — сколько реальных светодиодов у клавиатуры.
var LEDCount = func() int {
	n := 0
	for i := 0; i < slotCount; i++ {
		if !deadSlots[i] {
			n++
		}
	}
	return n
}()

type RGB struct{ R, G, B uint8 }

// Device — открытая клавиатура.
type Device struct {
	f    *os.File
	path string
}

func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	d := &Device{f: f, path: path}
	if err := d.init(); err != nil {
		f.Close()
		return nil, err
	}
	return d, nil
}

func (d *Device) Path() string { return d.path }
func (d *Device) Close() error { return d.f.Close() }

// initPacket переводит подсветку в режим прямого управления.
//
// Его нужно слать перед КАЖДЫМ кадром, а не однократно при открытии:
// иначе клавиатура через некоторое время возвращается к эффекту, зашитому
// в её собственную память, и перебивает то, что рисует программа.
func initPacket() []byte {
	buf := make([]byte, packetSize)
	buf[0x00] = 0x00
	buf[0x01] = 0x04
	buf[0x02] = 0xF2
	buf[0x09] = 0x09
	return buf
}

func (d *Device) init() error {
	return sendFeature(d.f, initPacket())
}

// buildFrame собирает пакеты одного кадра: сначала переключение в прямой
// режим, затем цвета. Вынесено отдельно, чтобы проверять без устройства.
func buildFrame(colors []RGB) ([][]byte, error) {
	if len(colors) != LEDCount {
		return nil, fmt.Errorf("нужно %d цветов, передано %d", LEDCount, len(colors))
	}

	// раскладываем по слотам, вставляя нули в отсутствующие позиции
	slots := make([]RGB, slotCount)
	src := 0
	for i := 0; i < slotCount; i++ {
		if deadSlots[i] {
			continue
		}
		slots[i] = colors[src]
		src++
	}

	packets := [][]byte{initPacket()}
	for start := 0; start < slotCount; start += colorsPerPacket {
		buf := make([]byte, packetSize)
		buf[0] = 0x00
		for j := 0; j < colorsPerPacket && start+j < slotCount; j++ {
			c := slots[start+j]
			off := j*4 + 1
			buf[off+0] = 0x81
			buf[off+1] = c.R
			buf[off+2] = c.G
			buf[off+3] = c.B
		}
		packets = append(packets, buf)
	}
	return packets, nil
}

// Show отправляет кадр. Длина colors должна быть LEDCount.
func (d *Device) Show(colors []RGB) error {
	packets, err := buildFrame(colors)
	if err != nil {
		return err
	}
	for i, p := range packets {
		if err := sendFeature(d.f, p); err != nil {
			return fmt.Errorf("пакет %d из %d: %w", i+1, len(packets), err)
		}
	}
	return nil
}

// Off гасит всю подсветку.
func (d *Device) Off() error {
	return d.Show(make([]RGB, LEDCount))
}

// MarshalJSON отдаёт цвет как "#rrggbb": интерфейсу удобнее работать со
// строкой, а не с тремя числами.
func (c RGB) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))), nil
}

func (c *RGB) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if len(s) != 7 || s[0] != '#' {
		return fmt.Errorf("цвет %q: ожидается #rrggbb", s)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return fmt.Errorf("цвет %q: %w", s, err)
	}
	c.R, c.G, c.B = uint8(v>>16), uint8(v>>8), uint8(v)
	return nil
}

// Hex — представление кадра одной строкой, по 6 символов на светодиод.
func Hex(frame []RGB) string {
	buf := make([]byte, 0, len(frame)*6)
	const digits = "0123456789abcdef"
	put := func(b uint8) { buf = append(buf, digits[b>>4], digits[b&0x0f]) }
	for _, c := range frame {
		put(c.R)
		put(c.G)
		put(c.B)
	}
	return string(buf)
}
