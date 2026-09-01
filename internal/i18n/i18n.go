// Package i18n хранит сообщения командной строки на поддерживаемых языках.
//
// Тексты интерфейса живут отдельно, в самом веб-интерфейсе: там их удобнее
// переключать без перезапуска программы.
package i18n

import "fmt"

const (
	RU = "ru"
	EN = "en"
)

// Supported перечисляет языки в порядке показа в интерфейсе.
var Supported = []string{EN, RU}

func Valid(lang string) bool {
	for _, l := range Supported {
		if l == lang {
			return true
		}
	}
	return false
}

var messages = map[string]map[string]string{
	EN: {
		"app.url":          "HyperX Studio: %s",
		"app.device":       "device: %s",
		"app.windowHint":   "The window can be closed — lighting keeps running.",
		"app.quitHint":     "Stop completely: hyperx-studio --quit",
		"app.alreadyOpen":  "Already running, opening the window.",
		"app.shuttingDown": "shutting down",
		"app.stopped":      "stopped",
		"app.notRunning":   "the application is not running",
		"app.openManually": "Open in a browser: %s",
		"app.windowFailed": "Could not open a window of our own, handing the interface to a browser.",

		"err.prefix":       "Error:",
		"err.noDevice":     "HyperX keyboard not found (%04X:%04X)",
		"err.noAccess":     "no access to the device: %v",
		"err.permsHint":    "\nIf the keyboard is plugged in, the likely cause is missing permissions on /dev/hidraw*.",
		"err.installUdev":  "Install the access rule:  sudo hyperx-studio --install-udev",
		"err.portBusy":     "could not bind %s: %v",
		"err.blackout":     "could not turn the lighting off:",
		"err.apply":        "could not apply the scheme:",
		"err.saveSettings": "could not save settings:",

		"dev.lost":                "keyboard disconnected: %v",
		"dev.restored":            "keyboard is back: %s",
		"sleep.releasing":         "system is going to sleep, releasing the keyboard",
		"sleep.resetFailed":       "could not reset the keyboard before sleep:",
		"sleep.resumed":           "woke up, keyboard is under control again",
		"sleep.noDeviceAfterWake": "keyboard did not come back after wake-up",

		"udev.needRoot": "root privileges required: sudo %s --install-udev",
		"udev.written":  "rule written: %s",
		"udev.replug":   "Unplug and plug the keyboard back in — permissions have to be applied anew.",
	},
	RU: {
		"app.url":          "HyperX Studio: %s",
		"app.device":       "устройство: %s",
		"app.windowHint":   "Окно можно закрыть — подсветка продолжит работать.",
		"app.quitHint":     "Остановить полностью: hyperx-studio --quit",
		"app.alreadyOpen":  "Приложение уже работает, открываю окно.",
		"app.shuttingDown": "завершаю работу",
		"app.stopped":      "остановлено",
		"app.notRunning":   "приложение не запущено",
		"app.openManually": "Откройте в браузере: %s",
		"app.windowFailed": "Своё окно открыть не удалось, отдаю интерфейс браузеру.",

		"err.prefix":       "Ошибка:",
		"err.noDevice":     "клавиатура HyperX не найдена (%04X:%04X)",
		"err.noAccess":     "нет доступа к устройству: %v",
		"err.permsHint":    "\nЕсли устройство на месте, скорее всего нет прав на /dev/hidraw*.",
		"err.installUdev":  "Установите правило доступа:  sudo hyperx-studio --install-udev",
		"err.portBusy":     "не удалось занять %s: %v",
		"err.blackout":     "не удалось погасить:",
		"err.apply":        "не удалось применить:",
		"err.saveSettings": "не удалось сохранить настройки:",

		"dev.lost":                "клавиатура отключена: %v",
		"dev.restored":            "клавиатура снова на связи: %s",
		"sleep.releasing":         "система уходит в сон, отпускаем клавиатуру",
		"sleep.resetFailed":       "не удалось переинициализировать клавиатуру перед сном:",
		"sleep.resumed":           "пробуждение, клавиатура снова под управлением",
		"sleep.noDeviceAfterWake": "клавиатура не вернулась после пробуждения",

		"udev.needRoot": "нужны права root: sudo %s --install-udev",
		"udev.written":  "правило записано: %s",
		"udev.replug":   "Переподключите клавиатуру физически — правам нужно примениться заново.",
	},
}

// T возвращает сообщение на выбранном языке. Неизвестный ключ возвращается
// как есть — это заметно при отладке и не роняет программу.
func T(lang, key string, args ...any) string {
	table, ok := messages[lang]
	if !ok {
		table = messages[EN]
	}
	s, ok := table[key]
	if !ok {
		if s, ok = messages[EN][key]; !ok {
			return key
		}
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}
