package i18n

import "testing"

// Наборы сообщений обязаны совпадать по ключам, иначе часть интерфейса
// внезапно окажется на другом языке.
func TestAllLanguagesHaveSameKeys(t *testing.T) {
	base := messages[EN]
	for _, lang := range Supported {
		table := messages[lang]
		for k := range base {
			if _, ok := table[k]; !ok {
				t.Errorf("в языке %q нет ключа %q", lang, k)
			}
		}
		for k := range table {
			if _, ok := base[k]; !ok {
				t.Errorf("в языке %q лишний ключ %q", lang, k)
			}
		}
	}
}

func TestFallbackToEnglish(t *testing.T) {
	if got := T("de", "app.stopped"); got != messages[EN]["app.stopped"] {
		t.Fatalf("неизвестный язык не откатился к английскому: %q", got)
	}
	if got := T(RU, "нет.такого.ключа"); got != "нет.такого.ключа" {
		t.Fatalf("неизвестный ключ вернул %q", got)
	}
}

func TestFormatting(t *testing.T) {
	if got := T(EN, "app.device", "/dev/hidraw0"); got != "device: /dev/hidraw0" {
		t.Fatalf("подстановка не сработала: %q", got)
	}
}
