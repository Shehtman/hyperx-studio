package webui

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

func asset(t *testing.T, name string) string {
	t.Helper()
	data, err := Files.ReadFile(name)
	if err != nil {
		t.Fatalf("нет файла %s: %v", name, err)
	}
	return string(data)
}

// Извлекает имена ключей верхнего уровня из блока словаря.
func dictKeys(t *testing.T, js, lang string) []string {
	t.Helper()
	start := strings.Index(js, "\n  "+lang+": {")
	if start < 0 {
		t.Fatalf("в словаре нет языка %q", lang)
	}
	body := js[start:]
	end := strings.Index(body, "\n  },")
	if end < 0 {
		t.Fatalf("блок языка %q не закрыт", lang)
	}
	body = body[:end]

	// ключи идут по несколько в строке, поэтому ищем их все подряд
	re := regexp.MustCompile(`(\w+):\s`)
	var keys []string
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if m[1] == lang { // строка, открывающая сам блок языка
			continue
		}
		keys = append(keys, m[1])
	}
	sort.Strings(keys)
	return keys
}

// Языки обязаны иметь одинаковый набор ключей: иначе при переключении часть
// подписей останется на другом языке или исчезнет.
func TestLanguagesHaveSameKeys(t *testing.T) {
	js := asset(t, "app.js")
	en := dictKeys(t, js, "en")
	ru := dictKeys(t, js, "ru")
	if len(en) < 20 {
		t.Fatalf("подозрительно мало ключей: %d", len(en))
	}
	if strings.Join(en, ",") != strings.Join(ru, ",") {
		t.Fatalf("наборы ключей расходятся:\n en: %v\n ru: %v", en, ru)
	}
}

// Каждый ключ из разметки должен существовать в словаре.
func TestMarkupKeysExist(t *testing.T) {
	js := asset(t, "app.js")
	html := asset(t, "index.html")
	known := map[string]bool{}
	for _, k := range dictKeys(t, js, "en") {
		known[k] = true
	}
	re := regexp.MustCompile(`data-i18n(?:-aria)?="(\w+)"`)
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		if !known[m[1]] {
			t.Errorf("в разметке ключ %q, которого нет в словаре", m[1])
		}
	}
}

// В разметке не должно остаться зашитого текста на одном языке.
// Исключение — названия самих языков: они всегда пишутся на себе.
func TestMarkupHasNoHardcodedText(t *testing.T) {
	html := asset(t, "index.html")
	for _, line := range strings.Split(html, "\n") {
		if strings.Contains(line, `<option value="ru"`) {
			continue
		}
		for _, r := range line {
			if r >= 'А' && r <= 'я' {
				t.Fatalf("незалокализованный текст в разметке: %s", strings.TrimSpace(line))
			}
		}
	}
}
