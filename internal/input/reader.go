// Package input читает нажатия клавиш напрямую из /dev/input.
//
// Устройства не захватываются: мы только подсматриваем коды, обычный ввод
// работает как обычно.
package input

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// На amd64 структура input_event занимает 24 байта:
// два int64 времени, два uint16 (type, code) и int32 value.
const eventSize = 24

const (
	evKey      = 0x01
	valuePress = 1
)

// Reader следит за клавиатурами и сообщает об их нажатиях.
type Reader struct {
	OnKey func(name string)

	mu      sync.Mutex
	files   []*os.File
	stopped bool
	Devices []string
	Err     error
}

// findKeyboards ищет устройства, которые выглядят как клавиатуры и доступны
// на чтение. Совпадение по имени не требуется: подсветка должна реагировать
// на любой ввод, в том числе с ноутбучной клавиатуры.
func findKeyboards(match string) ([]string, []*os.File) {
	paths, _ := filepath.Glob("/dev/input/event*")
	var names []string
	var files []*os.File
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		name := deviceName(p)
		if match != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(match)) {
			f.Close()
			continue
		}
		if !hasLetterKeys(p) {
			f.Close()
			continue
		}
		files = append(files, f)
		names = append(names, name)
	}
	return names, files
}

func deviceName(devPath string) string {
	base := filepath.Base(devPath)
	data, err := os.ReadFile(filepath.Join("/sys/class/input", base, "device", "name"))
	if err != nil {
		return base
	}
	return strings.TrimSpace(string(data))
}

// hasLetterKeys отсеивает мыши и кнопки питания: у настоящей клавиатуры
// в битовой карте EV_KEY присутствуют буквы.
func hasLetterKeys(devPath string) bool {
	base := filepath.Base(devPath)
	data, err := os.ReadFile(filepath.Join("/sys/class/input", base, "device", "capabilities", "key"))
	if err != nil {
		return false
	}
	// KEY_A = 30, KEY_Z = 44 — проверяем, что установлены оба бита
	return bitSet(string(data), 30) && bitSet(string(data), 44)
}

func bitSet(hexGroups string, bit int) bool {
	groups := strings.Fields(strings.TrimSpace(hexGroups))
	if len(groups) == 0 {
		return false
	}
	// группы идут от старших к младшим, по 64 бита каждая
	idx := len(groups) - 1 - bit/64
	if idx < 0 {
		return false
	}
	var v uint64
	for _, ch := range groups[idx] {
		d := hexVal(ch)
		if d < 0 {
			return false
		}
		v = v<<4 | uint64(d)
	}
	return v&(1<<uint(bit%64)) != 0
}

func hexVal(ch rune) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'f':
		return int(ch-'a') + 10
	case ch >= 'A' && ch <= 'F':
		return int(ch-'A') + 10
	}
	return -1
}

// Start запускает чтение. Возвращает false, если ни одного устройства нет.
func (r *Reader) Start() bool {
	names, files := findKeyboards("")
	if len(files) == 0 {
		r.Err = errNoDevices
		return false
	}
	r.mu.Lock()
	r.files = files
	r.Devices = names
	r.mu.Unlock()

	for _, f := range files {
		go r.loop(f)
	}
	return true
}

type inputError string

func (e inputError) Error() string { return string(e) }

const errNoDevices = inputError(
	"нет доступа к /dev/input/event*: реакция на нажатия отключена")

func (r *Reader) loop(f *os.File) {
	buf := make([]byte, eventSize*16)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return
		}
		r.mu.Lock()
		stopped := r.stopped
		r.mu.Unlock()
		if stopped {
			return
		}
		for off := 0; off+eventSize <= n; off += eventSize {
			rec := buf[off : off+eventSize]
			typ := binary.LittleEndian.Uint16(rec[16:18])
			code := binary.LittleEndian.Uint16(rec[18:20])
			val := int32(binary.LittleEndian.Uint32(rec[20:24]))
			if typ != evKey || val != valuePress {
				continue
			}
			if name, ok := CodeNames[code]; ok && r.OnKey != nil {
				r.OnKey(name)
			}
		}
	}
}

// Stop прекращает чтение и закрывает устройства.
func (r *Reader) Stop() {
	r.mu.Lock()
	r.stopped = true
	files := r.files
	r.files = nil
	r.mu.Unlock()
	for _, f := range files {
		f.Close()
	}
}
