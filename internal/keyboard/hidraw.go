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
