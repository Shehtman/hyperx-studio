package webui

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"hyperx-studio/internal/effects"
	"hyperx-studio/internal/engine"
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
	re := regexp.MustCompile(`data-i18n(?:-aria|-ph)?="(\w+)"`)
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

// dictSection достаёт вложенный словарь — названия эффектов или схем.
func dictSection(t *testing.T, js, lang, name string) map[string]bool {
	t.Helper()
	start := strings.Index(js, "\n  "+lang+": {")
	if start < 0 {
		t.Fatalf("в словаре нет языка %q", lang)
	}
	body := js[start:]
	if end := strings.Index(body, "\n  },"); end >= 0 {
		body = body[:end]
	}
	sec := strings.Index(body, "\n    "+name+": {")
	if sec < 0 {
		t.Fatalf("у языка %q нет раздела %q", lang, name)
	}
	body = body[sec:]
	end := strings.Index(body, "\n    },")
	if end < 0 {
		t.Fatalf("раздел %q у языка %q не закрыт", name, lang)
	}
	out := map[string]bool{}
	re := regexp.MustCompile(`(\w+):\s*'`)
	for _, m := range re.FindAllStringSubmatch(body[:end], -1) {
		out[m[1]] = true
	}
	return out
}

// Каждый эффект из реестра обязан иметь название на обоих языках: иначе в
// списке появится голый идентификатор вроде audiobars.
func TestEveryEffectIsNamedInBothLanguages(t *testing.T) {
	js := asset(t, "app.js")
	for _, lang := range []string{"en", "ru"} {
		named := dictSection(t, js, lang, "fx")
		for _, d := range effects.All {
			if !named[d.ID] {
				t.Errorf("эффект %q не назван на языке %q", d.ID, lang)
			}
		}
	}
}

// То же для встроенных схем.
func TestEveryBuiltinPresetIsNamed(t *testing.T) {
	js := asset(t, "app.js")
	for _, lang := range []string{"en", "ru"} {
		named := dictSection(t, js, lang, "ps")
		for _, p := range engine.Builtin() {
			if !named[p.ID] {
				t.Errorf("схема %q не названа на языке %q", p.ID, lang)
			}
		}
	}
}

// Схема ссылается на эффекты по идентификатору — опечатка молча превратила
// бы её в статику.
func TestBuiltinPresetsPointAtRealEffects(t *testing.T) {
	known := map[string]bool{}
	for _, d := range effects.All {
		known[d.ID] = true
	}
	for _, p := range engine.Builtin() {
		if !known[p.Effect] {
			t.Errorf("схема %q ссылается на несуществующий эффект %q", p.ID, p.Effect)
		}
		if p.Overlay != "" && !known[p.Overlay] {
			t.Errorf("схема %q ссылается на несуществующий слой %q", p.ID, p.Overlay)
		}
	}
}

// Схема собирается поверх настроек по умолчанию. Если кто-то задаст ей
// параметры голым литералом, молча обнулятся яркость, насыщенность и
// чувствительность — эффект станет чёрным или глухим к звуку.
func TestBuiltinPresetsKeepDefaultsTheyDoNotSet(t *testing.T) {
	for _, p := range engine.Builtin() {
		if p.Params.Brightness <= 0 {
			t.Errorf("у схемы %q нулевая яркость", p.ID)
		}
		if p.Params.Saturation <= 0 {
			t.Errorf("у схемы %q нулевая насыщенность", p.ID)
		}
		if p.Params.Sensitivity <= 0 {
			t.Errorf("у схемы %q нулевая чувствительность", p.ID)
		}
		if p.Params.Speed <= 0 {
			t.Errorf("у схемы %q нулевая скорость", p.ID)
		}
	}
}

// ids собирает идентификаторы, объявленные в разметке.
func ids(t *testing.T, html string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	re := regexp.MustCompile(`\bid="([\w-]+)"`)
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		out[m[1]] = true
	}
	return out
}

// Скрипт обращается к элементам по идентификатору. Опечатка или забытый при
// перекройке разметки элемент — это TypeError на первой же строке загрузки,
// после которого окно остаётся пустым.
func TestScriptOnlyTouchesExistingElements(t *testing.T) {
	html := asset(t, "index.html")
	js := asset(t, "app.js")
	known := ids(t, html)

	re := regexp.MustCompile(`\$\('([\w-]+)'\)`)
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(js, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		if !known[m[1]] {
			t.Errorf("скрипт ищет элемент %q, которого нет в разметке", m[1])
		}
	}
	if len(seen) < 20 {
		t.Fatalf("подозрительно мало обращений к элементам: %d", len(seen))
	}
}

// У каждого ползунка есть поле, куда пишется его значение: скрипт всегда
// обращается к паре id и id-v.
func TestEverySliderHasAValueField(t *testing.T) {
	html := asset(t, "index.html")
	js := asset(t, "app.js")
	known := ids(t, html)

	start := strings.Index(js, "const SLIDERS = {")
	if start < 0 {
		t.Fatal("в скрипте нет таблицы ползунков")
	}
	body := js[start:]
	end := strings.Index(body, "\n};")
	if end < 0 {
		t.Fatal("таблица ползунков не закрыта")
	}

	re := regexp.MustCompile(`(?m)^  (\w+):`)
	found := 0
	for _, m := range re.FindAllStringSubmatch(body[:end], -1) {
		found++
		if !known[m[1]] {
			t.Errorf("ползунка %q нет в разметке", m[1])
		}
		if !known[m[1]+"-v"] {
			t.Errorf("у ползунка %q нет поля %s-v для значения", m[1], m[1])
		}
	}
	if found < 5 {
		t.Fatalf("подозрительно мало ползунков: %d", found)
	}
}

// Вкладка без своей панели — пустая нижняя панель у человека на экране.
func TestEveryTabHasAPane(t *testing.T) {
	html := asset(t, "index.html")
	known := ids(t, html)
	re := regexp.MustCompile(`data-pane="(\w+)"`)
	found := 0
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		found++
		if !known["pane-"+m[1]] {
			t.Errorf("у вкладки %q нет панели #pane-%s", m[1], m[1])
		}
	}
	if found < 4 {
		t.Fatalf("подозрительно мало вкладок: %d", found)
	}
}
