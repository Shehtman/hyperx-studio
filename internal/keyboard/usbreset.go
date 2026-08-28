package keyboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// USBDEVFS_RESET — _IO('U', 20).
const usbdevfsReset = 0x5514

// ResetUSB просит ядро переинициализировать клавиатуру.
//
// Это единственный способ снять прямой режим подсветки. Команды «вернись к
// своему эффекту» в протоколе нет: её не знают ни OpenRGB, ни, судя по
// трафику, сама NGENUITY — включить прямой режим можно, выключить нельзя.
// После переинициализации прошивка стартует заново и возвращается к эффекту
// из собственной памяти.
//
// Нужно перед сном системы. Клавиатура, оставленная в прямом режиме без
// потока кадров, перестаёт сигналить remote wakeup, и компьютер нельзя
// разбудить — ни ею, ни мышью на том же контроллере.
func ResetUSB() error {
	path, err := usbPath(sysfsUSB, devBusUSB)
	if err != nil {
		return err
	}
	// Узел /dev/bus/usb доступен на запись владельцу локальной сессии:
	// его размечает та же метка uaccess, что и hidraw. Root не нужен.
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer f.Close()

	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, f.Fd(), usbdevfsReset, 0,
	); errno != 0 {
		return fmt.Errorf("переинициализация %s: %w", path, errno)
	}
	return nil
}

// Каталоги вынесены в переменные, чтобы поиск можно было проверить на
// подставном дереве без настоящего устройства.
var (
	sysfsUSB  = "/sys/bus/usb/devices"
	devBusUSB = "/dev/bus/usb"
)

// USBPath возвращает узел /dev/bus/usb нашей клавиатуры.
//
// Номера шины и устройства меняются при каждом переподключении, поэтому путь
// каждый раз собираем заново, а устройство ищем по идентификаторам.
func USBPath() (string, error) { return usbPath(sysfsUSB, devBusUSB) }

func usbPath(sysRoot, devRoot string) (string, error) {
	dirs, err := filepath.Glob(filepath.Join(sysRoot, "*"))
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		if !idMatches(dir) {
			continue
		}
		bus := readUint(filepath.Join(dir, "busnum"))
		dev := readUint(filepath.Join(dir, "devnum"))
		if bus == 0 || dev == 0 {
			continue
		}
		path := fmt.Sprintf("%s/%03d/%03d", devRoot, bus, dev)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("устройство %04X:%04X не найдено на шине USB",
		VendorID, ProductID)
}

// idMatches проверяет идентификаторы производителя и модели в sysfs.
// Файлы есть только у устройств целиком, у интерфейсов их нет — так мы
// заодно отсеиваем каталоги вида 1-1:1.0.
func idMatches(dir string) bool {
	return readHex(filepath.Join(dir, "idVendor")) == VendorID &&
		readHex(filepath.Join(dir, "idProduct")) == ProductID
}

func readHex(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 32)
	if err != nil {
		return -1
	}
	return int(v)
}

func readUint(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return v
}
