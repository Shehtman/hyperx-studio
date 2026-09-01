// Package keyboard общается с клавиатурой напрямую через hidraw.
// Никаких внешних библиотек: ioctl вызывается через syscall.
package keyboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	VendorID  = 0x03F0
	ProductID = 0x0591
)

// HIDIOCSFEATURE(len) = _IOC(_IOC_WRITE|_IOC_READ, 'H', 0x06, len)
func hidiocSFeature(size int) uintptr {
	const dir = 3 // _IOC_WRITE|_IOC_READ
	return uintptr(dir<<30 | size<<16 | 'H'<<8 | 0x06)
}

func sendFeature(f *os.File, buf []byte) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		hidiocSFeature(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return fmt.Errorf("ioctl HIDIOCSFEATURE: %w", errno)
	}
	return nil
}

// Candidates возвращает пути hidraw нашей клавиатуры, отсортированные по
// номеру USB-интерфейса.
//
// Порядок важен: подсветкой заведует нулевой интерфейс, а номера hidraw
// после переподключения устройства меняются произвольно — полагаться на
// hidraw0 нельзя.
func Candidates() ([]string, error) {
	entries, err := filepath.Glob("/sys/class/hidraw/hidraw*")
	if err != nil {
		return nil, err
	}
	want := fmt.Sprintf("%08X:%08X", VendorID, ProductID)

	type cand struct {
		path  string
		iface int
	}
	var found []cand
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(e, "device", "uevent"))
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToUpper(string(data)), want) {
			continue
		}
		found = append(found, cand{
			path:  "/dev/" + filepath.Base(e),
			iface: interfaceNumber(e),
		})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].iface < found[j].iface })

	out := make([]string, 0, len(found))
	for _, c := range found {
		out = append(out, c.path)
	}
	return out, nil
}

// interfaceNumber читает bInterfaceNumber; при неудаче возвращает большое
// число, чтобы такой путь пробовался последним.
func interfaceNumber(sysPath string) int {
	data, err := os.ReadFile(filepath.Join(sysPath, "device", "..", "bInterfaceNumber"))
	if err != nil {
		return 99
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 99
	}
	return n
}

// Info — найденный узел hidraw.
type Info struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Vendor  uint16 `json:"vendor"`
	Product uint16 `json:"product"`
	Iface   int    `json:"iface"`
	// Known поднят у клавиатуры, под которую писался протокол.
	Known bool `json:"known"`
}

// List перечисляет все узлы hidraw в системе.
//
// Нужен для выбора устройства вручную: протокол писался под Alloy Origins,
// но у HyperX много близкой родни, и запретить человеку попробовать свою
// клавиатуру мы не вправе — пусть решает сам, а мы честно пометим, какое
// устройство нам знакомо.
func List() ([]Info, error) {
	entries, err := filepath.Glob("/sys/class/hidraw/hidraw*")
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(e, "device", "uevent"))
		if err != nil {
			continue
		}
		in := Info{Path: "/dev/" + filepath.Base(e), Iface: interfaceNumber(e)}
		for _, line := range strings.Split(string(data), "\n") {
			switch {
			case strings.HasPrefix(line, "HID_NAME="):
				in.Name = strings.TrimPrefix(line, "HID_NAME=")
			case strings.HasPrefix(line, "HID_ID="):
				// формат: 0003:0000VVVV:0000PPPP
				parts := strings.Split(strings.TrimPrefix(line, "HID_ID="), ":")
				if len(parts) == 3 {
					v, err1 := strconv.ParseUint(parts[1], 16, 32)
					p, err2 := strconv.ParseUint(parts[2], 16, 32)
					if err1 == nil && err2 == nil {
						in.Vendor, in.Product = uint16(v), uint16(p)
					}
				}
			}
		}
		in.Known = in.Vendor == VendorID && in.Product == ProductID
		out = append(out, in)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Known != out[j].Known {
			return out[i].Known // своё устройство первым
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Iface < out[j].Iface
	})
	return out, nil
}
