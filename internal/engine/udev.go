package engine

import (
	"fmt"
	"os"
	"os/exec"

	"hyperx-studio/internal/i18n"
)

const udevPath = "/etc/udev/rules.d/60-hyperx-studio.rules"

// UdevRule — правило доступа к клавиатуре без прав root.
//
// hidraw нужен для подсветки, input — для реакции на нажатия. Метка uaccess
// выдаёт доступ пользователю активной локальной сессии, поэтому включать
// кого-то в группу input не требуется.
const UdevRule = `# HyperX Alloy Origins (03f0:0591) — доступ для hyperx-studio
KERNEL=="hidraw*", ATTRS{idVendor}=="03f0", ATTRS{idProduct}=="0591", TAG+="uaccess"
SUBSYSTEM=="input", ATTRS{idVendor}=="03f0", ATTRS{idProduct}=="0591", TAG+="uaccess"
`

// InstallUdev записывает правило и просит udev перечитать правила.
func InstallUdev() error {
	lang := Load().Lang
	if os.Geteuid() != 0 {
		return fmt.Errorf("%s", i18n.T(lang, "udev.needRoot", os.Args[0]))
	}
	if err := os.WriteFile(udevPath, []byte(UdevRule), 0o644); err != nil {
		return err
	}
	fmt.Println(i18n.T(lang, "udev.written", udevPath))
	for _, args := range [][]string{
		{"udevadm", "control", "--reload-rules"},
		{"udevadm", "trigger", "--subsystem-match=hidraw", "--subsystem-match=input"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			fmt.Printf("%s: %v %s\n", args[0], err, out)
		}
	}
	fmt.Println(i18n.T(lang, "udev.replug"))
	return nil
}

func UdevInstalled() bool {
	_, err := os.Stat(udevPath)
	return err == nil
}
