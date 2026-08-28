package keyboard

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeDevice раскладывает узел sysfs так, как это делает ядро.
func fakeDevice(t *testing.T, sysRoot, name, vendor, product, bus, dev string) {
	t.Helper()
	dir := filepath.Join(sysRoot, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for file, val := range map[string]string{
		"idVendor": vendor, "idProduct": product,
		"busnum": bus, "devnum": dev,
	} {
		if val == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(val+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// fakeNode создаёт заглушку узла /dev/bus/usb/BBB/DDD.
func fakeNode(t *testing.T, devRoot, bus, dev string) string {
	t.Helper()
	dir := filepath.Join(devRoot, bus)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, dev)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUSBPathFindsKeyboardByIdentifiers(t *testing.T) {
	sysRoot, devRoot := t.TempDir(), t.TempDir()

	// Посторонние устройства и интерфейс самой клавиатуры: у интерфейса нет
	// файлов с идентификаторами, он не должен сбивать поиск.
	fakeDevice(t, sysRoot, "1-2", "09da", "5c19", "1", "3")
	fakeDevice(t, sysRoot, "usb1", "1d6b", "0002", "1", "1")
	fakeDevice(t, sysRoot, "1-1:1.0", "", "", "", "")
	fakeDevice(t, sysRoot, "1-1", "03f0", "0591", "1", "2")

	want := fakeNode(t, devRoot, "001", "002")
	fakeNode(t, devRoot, "001", "003")

	got, err := usbPath(sysRoot, devRoot)
	if err != nil {
		t.Fatalf("устройство не найдено: %v", err)
	}
	if got != want {
		t.Fatalf("получен путь %s, ожидался %s", got, want)
	}
}

// Номера шины и устройства меняются при переподключении, поэтому путь должен
// собираться из sysfs, а не запоминаться.
func TestUSBPathFollowsRenumberedDevice(t *testing.T) {
	sysRoot, devRoot := t.TempDir(), t.TempDir()
	fakeDevice(t, sysRoot, "1-1", "03f0", "0591", "2", "17")
	want := fakeNode(t, devRoot, "002", "017")

	got, err := usbPath(sysRoot, devRoot)
	if err != nil {
		t.Fatalf("устройство не найдено: %v", err)
	}
	if got != want {
		t.Fatalf("получен путь %s, ожидался %s", got, want)
	}
}

func TestUSBPathWithoutKeyboard(t *testing.T) {
	sysRoot, devRoot := t.TempDir(), t.TempDir()
	fakeDevice(t, sysRoot, "1-2", "09da", "5c19", "1", "3")
	fakeNode(t, devRoot, "001", "003")

	if got, err := usbPath(sysRoot, devRoot); err == nil {
		t.Fatalf("чужое устройство принято за нашу клавиатуру: %s", got)
	}
}

// Узел в sysfs есть, а файла устройства нет: путь отдавать нельзя, иначе
// открытие упадёт с невнятной ошибкой.
func TestUSBPathWithoutNode(t *testing.T) {
	sysRoot, devRoot := t.TempDir(), t.TempDir()
	fakeDevice(t, sysRoot, "1-1", "03f0", "0591", "1", "2")

	if got, err := usbPath(sysRoot, devRoot); err == nil {
		t.Fatalf("отдан путь к несуществующему узлу: %s", got)
	}
}
