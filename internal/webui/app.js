'use strict';

const $ = (id) => document.getElementById(id);
const SWATCHES = [
  '#ff0000', '#ff6a00', '#ffd400', '#7dff00', '#00ff3c', '#00ffb3', '#00e5ff', '#0077ff',
  '#3b1bff', '#a020f0', '#ff00d4', '#ff2b6b', '#ffffff', '#ffd9b3', '#b3e0ff', '#404040',
];


// ── языки ───────────────────────────────────────────────────────────
// Ключи одинаковы для всех языков; проверка совпадения наборов — в тестах.
const STRINGS = {
  en: {
    pause: 'Pause', resume: 'Resume', blackout: 'Turn off',
    effect: 'Effect', overlay: 'Overlay', none: 'none',
    brightness: 'Brightness', speed: 'Speed', angle: 'Angle', scale: 'Scale',
    density: 'Density', length: 'Length', color: 'Colour', color2: 'Second',
    reaction: 'Keypress', fade: 'Fade', rainbow: 'Rainbow',
    selAll: 'All', selNone: 'Clear', selInvert: 'Invert',
    customColour: 'Custom', resetPaint: 'Reset', selectedOnly: 'Selected only',
    layout: 'Layout', fps: 'Frames/s', autostart: 'Start on login',
    language: 'Language',
    presets: 'Presets', save: 'Save', presetName: 'Name for this scheme',
    deletePreset: 'Delete', presetSaved: 'Scheme saved.',
    presetNameTaken: 'That name is already taken by a built-in scheme.',
    background: 'Background', saturation: 'Saturation',
    sensitivity: 'Sensitivity', reverse: 'Reverse',
    sound: 'Sound', soundSource: 'Source', soundDefault: 'System output',
    noSound: 'No capture tool found. Install pulseaudio-utils or pipewire for sound effects.',
    soundFailed: 'Could not capture sound.',
    customColourAria: 'Fill with a custom colour', keyboardAria: 'Keyboard layout',
    levelAria: 'Sound level',
    tabEffect: 'Effect', tabKeys: 'Keys', tabSetup: 'Setup',
    selection: 'Selection', fill: 'Fill',
    reverseHint: 'The other way round',
    rainbowHint: 'A colour per key',
    maskHint: 'Darken the rest',
    autostartHint: 'Restore lighting at login',
    reactHint: 'Keypress reaction works with a reactive effect or overlay.',
    keysHint: 'Drag across the keyboard to select. Hold Ctrl to add to the selection.',
    windowHint: 'Closing the window leaves the lighting on — the program keeps running in the background.',
    quit: 'Quit', quitBtn: 'Quit the program',
    device: 'Device',
    deviceAuto: 'Detect automatically',
    deviceNone: 'No keyboard connected. The window still works — pick effects and schemes, and it will light up as soon as one is plugged in.',
    deviceFailed: (e) => `Could not open the device — ${e}`,
    deviceOther: 'Not an Alloy Origins — the protocol may not fit it.',
    deviceGone: 'not found',
    presetAria: (name) => `Apply the ${name} scheme`,
    maskOn: (n) => `Only the ${n} selected key(s) are lit.`,
    disconnected: 'Keyboard disconnected. Lighting resumes as soon as it is back.',
    noInput: 'No access to input devices — keypress reaction is off.',
    noLink: 'Lost connection to the application.',
    fpsUnit: '/s',
    fx: {
      static: 'Static', breathing: 'Breathing', spectrum: 'Spectrum cycle',
      rainbow: 'Rainbow wave', colorwave: 'Two-colour wave', gradient: 'Gradient',
      twinkle: 'Twinkle', rain: 'Rain', fire: 'Fire', snake: 'Snake',
      ripple: 'Key ripple', flash: 'Key flash',
      audiobars: 'Equaliser', audiospectrum: 'Sound spectrum',
      audiopulse: 'Sound pulse', audiowave: 'Wave to music',
    },
    ps: {
      aurora: 'Aurora', sunset: 'Sunset', matrix: 'Matrix', starfield: 'Starfield',
      lava: 'Lava', ocean: 'Ocean', cyberpunk: 'Cyberpunk', rainbow: 'Rainbow',
      pastel: 'Pastel', typewriter: 'Typewriter', breathe: 'Calm breathing',
      snake: 'Snake', equalizer: 'Equaliser', bassdrop: 'Bass drop',
      discolight: 'Disco',
    },
  },
  ru: {
    pause: 'Пауза', resume: 'Продолжить', blackout: 'Погасить',
    effect: 'Эффект', overlay: 'Поверх', none: 'нет',
    brightness: 'Яркость', speed: 'Скорость', angle: 'Угол', scale: 'Масштаб',
    density: 'Плотность', length: 'Длина', color: 'Цвет', color2: 'Второй',
    reaction: 'Нажатия', fade: 'Затухание', rainbow: 'Радуга',
    selAll: 'Все', selNone: 'Снять', selInvert: 'Инверт.',
    customColour: 'Свой', resetPaint: 'Сброс', selectedOnly: 'Только выбранные',
    layout: 'Раскладка', fps: 'Кадров/с', autostart: 'Автозапуск',
    language: 'Язык',
    presets: 'Схемы', save: 'Сохранить', presetName: 'Название схемы',
    deletePreset: 'Удалить', presetSaved: 'Схема сохранена.',
    presetNameTaken: 'Это имя занято встроенной схемой.',
    background: 'Фон', saturation: 'Насыщенность',
    sensitivity: 'Чувствительность', reverse: 'Обратно',
    sound: 'Звук', soundSource: 'Источник', soundDefault: 'Системный вывод',
    noSound: 'Не найдено, чем захватывать звук. Для звуковых эффектов нужен pulseaudio-utils или pipewire.',
    soundFailed: 'Не удалось захватить звук.',
    customColourAria: 'Залить своим цветом', keyboardAria: 'Раскладка клавиатуры',
    levelAria: 'Уровень звука',
    tabEffect: 'Эффект', tabKeys: 'Клавиши', tabSetup: 'Настройки',
    selection: 'Выделение', fill: 'Заливка',
    reverseHint: 'В обратную сторону',
    rainbowHint: 'Свой цвет каждой клавише',
    maskHint: 'Гасить остальные',
    autostartHint: 'Возвращать подсветку при входе',
    reactHint: 'Реакция на нажатия работает с отзывчивым эффектом или слоем поверх.',
    keysHint: 'Ведите мышью по клавиатуре, чтобы выделить. С Ctrl выделение добавляется.',
    windowHint: 'Закрытое окно не гасит подсветку — программа продолжает работать в фоне.',
    quit: 'Выход', quitBtn: 'Завершить программу',
    device: 'Устройство',
    deviceAuto: 'Определять самостоятельно',
    deviceNone: 'Клавиатура не подключена. Окно работает: выбирайте эффекты и схемы — подсветка загорится, как только устройство появится.',
    deviceFailed: (e) => `Не удалось открыть устройство — ${e}`,
    deviceOther: 'Это не Alloy Origins — протокол может ему не подойти.',
    deviceGone: 'не найдено',
    presetAria: (name) => `Включить схему «${name}»`,
    maskOn: (n) => `Горят только выбранные клавиши — ${n} шт.`,
    disconnected: 'Клавиатура отключена. Подсветка вернётся, как только устройство появится.',
    noInput: 'Нет доступа к устройствам ввода — реакция на нажатия отключена.',
    noLink: 'Связь с приложением потеряна.',
    fpsUnit: '/с',
    fx: {
      static: 'Статика', breathing: 'Дыхание', spectrum: 'Перелив спектра',
      rainbow: 'Радужная волна', colorwave: 'Волна двух цветов', gradient: 'Градиент',
      twinkle: 'Звёзды', rain: 'Дождь', fire: 'Огонь', snake: 'Змейка',
      ripple: 'Волны от нажатий', flash: 'Вспышка по нажатию',
      audiobars: 'Эквалайзер', audiospectrum: 'Спектр звука',
      audiopulse: 'Пульс звука', audiowave: 'Волна под музыку',
    },
    ps: {
      aurora: 'Северное сияние', sunset: 'Закат', matrix: 'Матрица',
      starfield: 'Звёздное небо', lava: 'Лава', ocean: 'Океан',
      cyberpunk: 'Киберпанк', rainbow: 'Радуга', pastel: 'Пастель',
      typewriter: 'Печатная машинка', breathe: 'Спокойное дыхание',
      snake: 'Змейка', equalizer: 'Эквалайзер', bassdrop: 'Удар баса',
      discolight: 'Дискотека',
    },
  },
};

let LANG = 'en';
const t = (key) => (STRINGS[LANG] || STRINGS.en)[key];
const fxName = (id) => ((STRINGS[LANG] || STRINGS.en).fx[id] || id);
const psName = (p) => (p.builtin
  ? ((STRINGS[LANG] || STRINGS.en).ps[p.id] || p.id)
  : p.name);

// applyLang перерисовывает подписи. Список эффектов пересобирается, потому
// что названия в нём тоже переводятся.
function applyLang(lang) {
  LANG = STRINGS[lang] ? lang : 'en';
  document.documentElement.lang = LANG;

  document.querySelectorAll('[data-i18n]').forEach((el) => {
    const v = t(el.dataset.i18n);
    if (typeof v === 'string') el.textContent = v;
  });
  document.querySelectorAll('[data-i18n-aria]').forEach((el) => {
    const v = t(el.dataset.i18nAria);
    if (typeof v === 'string') el.setAttribute('aria-label', v);
  });
  document.querySelectorAll('[data-i18n-ph]').forEach((el) => {
    const v = t(el.dataset.i18nPh);
    if (typeof v === 'string') el.placeholder = v;
  });

  fillEffects();
  fillPresets();
  fillSources();
  const paused = $('pause').getAttribute('aria-pressed') === 'true';
  $('pause').textContent = paused ? t('resume') : t('pause');
  document.querySelectorAll('.sw').forEach((b) => {
    b.setAttribute('aria-label', `${t('customColour')} ${b.dataset.color}`);
  });
  $('about').textContent = t('windowHint');
  renderNotice();
}

function fillEffects() {
  if (!S) return;
  const eff = $('effect'), ovl = $('overlay');
  const keepE = eff.value || S.state.effect;
  const keepO = ovl.value || S.state.overlay || '';
  eff.textContent = '';
  ovl.textContent = '';
  ovl.appendChild(new Option(t('none'), ''));
  for (const d of S.effects) {
    eff.appendChild(new Option(fxName(d.id), d.id));
    if (d.reactive) ovl.appendChild(new Option(fxName(d.id), d.id));
  }
  eff.value = keepE;
  ovl.value = keepO;
}

let S = null;              // снимок состояния с сервера
let keys = [];             // клавиши текущей раскладки
let selection = new Set();
let effectsById = new Map();

// ── обмен с сервером ────────────────────────────────────────────────

// post возвращает ответ сервера: часть вызовов проверяет, приняли ли запрос.
async function post(path, body) {
  try {
    return await fetch('/api' + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body ?? {}),
    });
  } catch (e) {
    warn(t('noLink'));
    return null;
  }
}

// Одна строка сообщений на все случаи. Приоритет: сначала то, что мешает
// работать, потом то, что объясняет неожиданный результат.
let notice = { device: '', input: '', link: '' };
let lastOnline = null;

function warn(text) {
  notice.link = text || '';
  renderNotice();
}

function renderNotice() {
  const el = $('warn');
  const mask = $('mask').checked && selection.size > 0
    ? t('maskOn')(selection.size)
    : '';
  const text = notice.link || notice.device || notice.input || mask;
  el.textContent = text;
  el.hidden = !text;
  el.classList.toggle('info', !notice.link && !notice.device && !notice.input && !!mask);
}

// ── отрисовка раскладки ─────────────────────────────────────────────
//
// Клавиатура рисуется на холсте, а не в SVG.
//
// В SVG каждая клавиша была узлом документа, и кадр означал сотню записей
// атрибутов: браузер пересчитывал стили, заново раскладывал сцену и заново
// её растрировал. Под композитором, который сам едва справляется, этого
// хватало, чтобы отрисовка встала совсем. Холст перерисовывается целиком
// одним проходом и стоит браузеру одной картинки.
//
// Холста два. Нижний рисует те же клавиши в четверть разрешения и размыт
// средствами браузера — это подсветка столешницы, её считает видеокарта.
// Верхний рисует сами клавиши с подписями.

const KEY_EDGE = 'rgba(255,255,255,.10)';
const KEY_SEL = '#7AA2FF';
const LABEL_LIGHT = '#C9CEDE';
const LABEL_DARK = '#0B0D12';
const U = 60;      // пикселей на юнит во внутренней системе координат
const GAP = 4;
const GLOW_DIV = 4;

let kb = null, kbCtx = null, glow = null, glowCtx = null;
let boardW = 0, boardH = 0;   // размер раскладки во внутренних координатах
let kbScale = 1, glowScale = 1;
let hitBoxes = [];            // прямоугольники клавиш для попадания мышью
const gradients = new Map();  // блик по высоте клавиши — считаем один раз

// Путь клавиши во внутренних координатах. Верхняя часть Enter на ISO шире
// нижней, поэтому у неё свой контур, а не прямоугольник.
function keyPath(ctx, k) {
  const x0 = k.x * U + GAP / 2;
  const y0 = k.y * U + GAP / 2;
  const x1 = (k.x + k.w) * U - GAP / 2;
  const y1 = (k.y + k.h) * U - GAP / 2;
  ctx.beginPath();
  if (k.shape === 'iso-enter') {
    const nx = (k.x + 0.25) * U + GAP / 2;
    const ny = (k.y + 1) * U + GAP / 2;
    ctx.moveTo(x0, y0);
    ctx.lineTo(x1, y0);
    ctx.lineTo(x1, y1);
    ctx.lineTo(nx, y1);
    ctx.lineTo(nx, ny);
    ctx.lineTo(x0, ny);
    ctx.closePath();
    return;
  }
  roundRect(ctx, x0, y0, x1 - x0, y1 - y0, 6);
}

function roundRect(ctx, x, y, w, h, r) {
  const rr = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.arcTo(x + w, y, x + w, y + h, rr);
  ctx.arcTo(x + w, y + h, x, y + h, rr);
  ctx.arcTo(x, y + h, x, y, rr);
  ctx.arcTo(x, y, x + w, y, rr);
  ctx.closePath();
}

// capGloss — вертикальный блик, из-за которого клавиша читается объёмной,
// а не плоской заливкой. Зависит только от границ по вертикали, поэтому
// градиентов получается ровно столько, сколько рядов.
function capGloss(k) {
  const y0 = k.y * U + GAP / 2;
  const y1 = (k.y + k.h) * U - GAP / 2;
  const key = y0 + '/' + y1;
  let g = gradients.get(key);
  if (!g) {
    g = kbCtx.createLinearGradient(0, y0, 0, y1);
    g.addColorStop(0, 'rgba(255,255,255,.13)');
    g.addColorStop(0.5, 'rgba(255,255,255,0)');
    g.addColorStop(1, 'rgba(0,0,0,.22)');
    gradients.set(key, g);
  }
  return g;
}

function buildKeyboard() {
  kb = $('kb');
  glow = $('glow');
  kbCtx = kb.getContext('2d');
  glowCtx = glow.getContext('2d');
  boardW = S.width * U;
  boardH = S.height * U;
  gradients.clear();
  prevHex = '';
  layoutCanvas();
}

// layoutCanvas подгоняет холсты под окно, сохраняя пропорции раскладки.
function layoutCanvas() {
  if (!kb) return;
  const box = $('stage').getBoundingClientRect();
  if (box.width < 2 || box.height < 2) return;

  const scale = Math.min(box.width / boardW, box.height / boardH);
  const cssW = Math.max(1, Math.floor(boardW * scale));
  const cssH = Math.max(1, Math.floor(boardH * scale));
  const dpr = Math.min(window.devicePixelRatio || 1, 2);

  for (const c of [kb, glow]) {
    c.style.width = cssW + 'px';
    c.style.height = cssH + 'px';
  }
  kb.width = Math.round(cssW * dpr);
  kb.height = Math.round(cssH * dpr);
  glow.width = Math.max(1, Math.round(cssW * dpr / GLOW_DIV));
  glow.height = Math.max(1, Math.round(cssH * dpr / GLOW_DIV));
  kbScale = kb.width / boardW;
  glowScale = glow.width / boardW;

  // Градиенты живут в координатах холста и после смены масштаба уже не те.
  gradients.clear();

  hitBoxes = keys.map((k) => ({
    index: k.index,
    x: k.x * U, y: k.y * U, w: k.w * U, h: k.h * U,
  }));

  prevHex = '';                     // размеры сменились, рисуем заново
  drawBoard(lastHex || '');
}

let lastHex = '';
let prevHex = '';

function drawBoard(hex) {
  if (!kbCtx) return;
  const c = kbCtx, g = glowCtx;

  c.setTransform(1, 0, 0, 1, 0, 0);
  c.clearRect(0, 0, kb.width, kb.height);
  c.setTransform(kbScale, 0, 0, kbScale, 0, 0);

  g.setTransform(1, 0, 0, 1, 0, 0);
  g.clearRect(0, 0, glow.width, glow.height);
  g.setTransform(glowScale, 0, 0, glowScale, 0, 0);

  c.lineJoin = 'round';
  c.textAlign = 'center';
  c.textBaseline = 'middle';

  const long = hex.length >= keys.length * 6;

  for (const k of keys) {
    const off = k.index * 6;
    const rgb = long ? hex.slice(off, off + 6) : '000000';
    const fill = '#' + rgb;
    const r = parseInt(rgb.slice(0, 2), 16) || 0;
    const gg = parseInt(rgb.slice(2, 4), 16) || 0;
    const b = parseInt(rgb.slice(4, 6), 16) || 0;
    const lum = r * 0.299 + gg * 0.587 + b * 0.114;

    // подсветка столешницы — только цвет, без подписей и рамок
    if (lum > 6) {
      keyPath(g, k);
      g.fillStyle = fill;
      g.fill();
    }

    keyPath(c, k);
    c.fillStyle = fill;
    c.fill();
    c.fillStyle = capGloss(k);
    c.fill();

    const sel = selection.has(k.index);
    c.strokeStyle = sel ? KEY_SEL : KEY_EDGE;
    c.lineWidth = sel ? 3 : 1.5;
    c.stroke();

    if (k.label) {
      c.fillStyle = lum > 140 ? LABEL_DARK : LABEL_LIGHT;
      c.font = (k.label.length > 2 ? '13px ' : '17px ')
             + 'ui-sans-serif, system-ui, sans-serif';
      const cx = k.shape === 'iso-enter'
        ? (k.x + 0.25 + (k.w - 0.25) / 2) * U
        : (k.x + k.w / 2) * U;
      c.fillText(k.label, cx, (k.y + k.h / 2) * U);
    }
  }
  prevHex = hex;
}

// ── поток кадров ────────────────────────────────────────────────────
//
// Кадры приходят чаще, чем браузер рисует, поэтому копится только
// последний, а отрисовка откладывается до ближайшего кадра браузера.

let pendingHex = null;
let pendingLevel = null;
let framePending = false;
let rafId = 0;
let scheduledAt = 0;

function queueFrame(hex, level) {
  pendingHex = hex;
  lastHex = hex;
  if (level !== undefined) pendingLevel = level;
  if (framePending) return;
  framePending = true;
  scheduledAt = performance.now();
  rafId = requestAnimationFrame(paintPending);
}

function paintPending() {
  framePending = false;
  const h = pendingHex;
  pendingHex = null;
  if (h !== null && h !== prevHex) drawBoard(h);
  if (pendingLevel !== null) {
    // Ширину не трогаем: transform не вызывает пересчёт разметки.
    $('level-bar').style.transform = `scaleX(${pendingLevel})`;
    pendingLevel = null;
  }
}

// Сторож на случай, когда браузер перестаёт выдавать кадры.
//
// requestAnimationFrame молчит, пока окно не рисуется, — и если запрос уже
// отправлен, флаг остаётся поднятым навсегда: новые кадры складываются в
// переменную, а на экран не попадают уже никогда. Раньше это выглядело как
// «показал десяток кадров и замер». Теперь просроченный запрос снимается и
// кадр рисуется по таймеру.
function startWatchdog() {
  setInterval(() => {
    if (!framePending) return;
    if (performance.now() - scheduledAt < 500) return;
    cancelAnimationFrame(rafId);
    paintPending();
  }, 500);
}

// ── выделение мышью ─────────────────────────────────────────────────

function boardPoint(evt) {
  const r = kb.getBoundingClientRect();
  return {
    x: (evt.clientX - r.left) / r.width * boardW,
    y: (evt.clientY - r.top) / r.height * boardH,
  };
}

function keyAt(p) {
  for (const b of hitBoxes) {
    if (p.x >= b.x && p.x <= b.x + b.w && p.y >= b.y && p.y <= b.y + b.h) {
      return b.index;
    }
  }
  return null;
}

function setupSelection() {
  let start = null, additive = false, moved = false, before = null;

  kb.addEventListener('mousedown', (e) => {
    if (e.button !== 0) return;
    additive = e.ctrlKey || e.shiftKey;
    before = new Set(selection);
    start = boardPoint(e);
    moved = false;
    e.preventDefault();
  });

  kb.addEventListener('mousemove', (e) => {
    if (!start) return;
    const p = boardPoint(e);
    if (!moved && Math.abs(p.x - start.x) < 6 && Math.abs(p.y - start.y) < 6) return;
    moved = true;
    const x = Math.min(start.x, p.x), y = Math.min(start.y, p.y);
    const w = Math.abs(p.x - start.x), h = Math.abs(p.y - start.y);

    const hit = new Set();
    for (const b of hitBoxes) {
      if (b.x < x + w && b.x + b.w > x && b.y < y + h && b.y + b.h > y) hit.add(b.index);
    }
    selection = additive ? new Set([...before, ...hit]) : hit;
    dragBox = { x, y, w, h };
    drawBoard(lastHex);
    drawDragBox();
  });

  const finish = (e) => {
    if (!start) return;
    if (!moved) {
      const idx = keyAt(boardPoint(e));
      if (idx === null) {
        if (!additive) selection.clear();
      } else if (additive) {
        selection.has(idx) ? selection.delete(idx) : selection.add(idx);
      } else {
        selection = new Set([idx]);
      }
    }
    dragBox = null;
    start = null;
    pushSelection();
  };

  kb.addEventListener('mouseup', finish);
  kb.addEventListener('mouseleave', (e) => { if (start) finish(e); });
}

let dragBox = null;

function drawDragBox() {
  if (!dragBox || !kbCtx) return;
  const c = kbCtx;
  c.save();
  c.setTransform(kbScale, 0, 0, kbScale, 0, 0);
  c.fillStyle = 'rgba(122,162,255,.12)';
  c.strokeStyle = KEY_SEL;
  c.lineWidth = 2;
  c.setLineDash([7, 5]);
  c.fillRect(dragBox.x, dragBox.y, dragBox.w, dragBox.h);
  c.strokeRect(dragBox.x, dragBox.y, dragBox.w, dragBox.h);
  c.restore();
}

function pushSelection() {
  $('selcount').textContent = selection.size;
  drawBoard(lastHex);
  renderNotice();
  post('/selection', { indices: [...selection] });
}

// ── параметры ───────────────────────────────────────────────────────

const SLIDERS = {
  brightness: { div: 100, fmt: (v) => (v * 100).toFixed(0) + '%' },
  speed: { div: 100, fmt: (v) => v.toFixed(2).replace(/0+$/, '').replace(/\.$/, '') + '×' },
  angle: { div: 1, fmt: (v) => v.toFixed(0) + '°' },
  scale: { div: 100, fmt: (v) => v.toFixed(2).replace(/0+$/, '').replace(/\.$/, '') + '×' },
  density: { div: 100, fmt: (v) => v.toFixed(2) },
  length: { div: 1, fmt: (v) => v.toFixed(0) },
  reactSpeed: { div: 100, fmt: (v) => v.toFixed(2).replace(/0+$/, '').replace(/\.$/, '') + '×' },
  reactFade: { div: 100, fmt: (v) => v.toFixed(2) + ' s' },
  saturation: { div: 100, fmt: (v) => (v * 100).toFixed(0) + '%' },
  sensitivity: { div: 100, fmt: (v) => v.toFixed(2).replace(/0+$/, '').replace(/\.$/, '') + '×' },
};

function readParams() {
  const p = {};
  for (const [id, s] of Object.entries(SLIDERS)) p[id] = Number($(id).value) / s.div;
  p.length = Math.round(p.length);
  p.color1 = $('color1').value;
  p.color2 = $('color2').value;
  p.reactColor = $('reactColor').value;
  p.background = $('background').value;
  p.rainbow = $('rainbow').checked;
  p.reverse = $('reverse').checked;
  return p;
}

function showValues() {
  for (const [id, s] of Object.entries(SLIDERS)) {
    $(id + '-v').textContent = s.fmt(Number($(id).value) / s.div);
    fillTrack($(id));
  }
  $('fps-v').textContent = $('fps').value;
  fillTrack($('fps'));
}

// fillTrack сообщает стилям, какая доля ползунка пройдена.
function fillTrack(el) {
  const min = Number(el.min) || 0;
  const max = Number(el.max);
  const share = max > min ? (Number(el.value) - min) / (max - min) : 0;
  el.style.setProperty('--fill', (share * 100).toFixed(1) + '%');
}

// syncParamVisibility прячет то, на что выбранный эффект не смотрит, — и
// вместе с настройками прячет целые вкладки, чтобы человек не открывал
// пустую.
function syncParamVisibility() {
  const base = effectsById.get($('effect').value);
  const ovl = effectsById.get($('overlay').value);
  const uses = new Set(base ? base.uses || [] : []);
  document.querySelectorAll('#params .ctl[data-p]').forEach((cell) => {
    const p = cell.dataset.p;
    cell.hidden = !(p === 'brightness' || uses.has(p));
  });

  const reactive = (base && base.reactive) || !!ovl;
  $('react-hint').hidden = reactive;
  document.querySelectorAll('#pane-react .ctl').forEach((c) => { c.hidden = !reactive; });


  // Источник звука нужен, только когда его кто-то слушает.
  const wantsAudio = (base && base.audio) || (ovl && ovl.audio);
  tabByName('sound').hidden = !wantsAudio;
  if (!wantsAudio && activeTab === 'sound') showTab('fx');
}

// ── вкладки ─────────────────────────────────────────────────────────

let activeTab = 'fx';

const tabByName = (name) => document.querySelector(`.tab[data-pane="${name}"]`);

function showTab(name) {
  activeTab = name;
  document.querySelectorAll('.tab').forEach((b) => {
    b.classList.toggle('on', b.dataset.pane === name);
    b.setAttribute('aria-selected', String(b.dataset.pane === name));
  });
  document.querySelectorAll('.pane').forEach((p) => {
    p.classList.toggle('on', p.id === 'pane-' + name);
  });
  // Высота панели изменилась — клавиатуре досталось другое место.
  requestAnimationFrame(layoutCanvas);
}

// ── схемы ───────────────────────────────────────────────────────────

function fillDevices() {
  const sel = $('device');
  sel.textContent = '';
  const auto = document.createElement('option');
  auto.value = '';
  auto.textContent = t('deviceAuto');
  sel.appendChild(auto);
  for (const h of S.hardware || []) {
    const o = document.createElement('option');
    o.value = h.path;
    const id = h.vendor.toString(16).padStart(4, '0') + ':'
             + h.product.toString(16).padStart(4, '0');
    const name = (h.name || '').trim() || h.path;
    o.textContent = `${name} · ${id} · ${h.path.replace('/dev/', '')}`;
    o.dataset.known = h.known ? '1' : '';
    sel.appendChild(o);
  }
  const want = S.state.device || '';
  // Выбранное устройство могло исчезнуть — например, его отключили. Молча
  // переключаться на автоопределение нельзя: движок по-прежнему ждёт именно
  // его, и список показывал бы не то, что происходит на самом деле.
  if (want && ![...sel.options].some((o) => o.value === want)) {
    const o = document.createElement('option');
    o.value = want;
    o.textContent = `${want} · ${t('deviceGone')}`;
    sel.appendChild(o);
  }
  sel.value = want;
}

function syncDeviceNotice() {
  if (!S.connected) {
    // Про поломку говорим только тогда, когда человек выбрал устройство
    // сам: при автоопределении «клавиатуры нет» — это обычное состояние,
    // и сыпать в него текстом ioctl незачем.
    notice.device = (S.state.device && S.devErr)
      ? t('deviceFailed')(S.devErr) : t('deviceNone');
  } else {
    const opt = $('device').selectedOptions[0];
    notice.device = (S.state.device && opt && !opt.dataset.known)
      ? t('deviceOther') : '';
  }
  renderNotice();
}

function fillPresets() {
  const sel = $('preset');
  if (!sel || !S) return;
  sel.textContent = '';

  const blank = document.createElement('option');
  blank.value = '';
  blank.textContent = '—';
  sel.append(blank);

  for (const p of S.presets || []) {
    const o = document.createElement('option');
    o.value = p.builtin ? p.id : p.name;
    o.textContent = psName(p);
    o.dataset.builtin = p.builtin ? '1' : '';
    sel.append(o);
  }
  sel.value = '';
  syncPresetButtons();
}

// Удалять можно только свои схемы, поэтому кнопка появляется лишь для них.
function syncPresetButtons() {
  const sel = $('preset');
  const opt = sel.selectedOptions[0];
  $('preset-del').hidden = !opt || !opt.value || opt.dataset.builtin === '1';
}

// ── звук ────────────────────────────────────────────────────────────

async function fillSources() {
  const sel = $('audio-src');
  if (!sel) return;
  let list = [];
  try {
    list = await (await fetch('/api/audio/sources')).json();
  } catch (err) {
    list = [];
  }
  const current = (S && S.state.audioSource) || '';
  sel.textContent = '';

  const auto = document.createElement('option');
  auto.value = '';
  auto.textContent = t('soundDefault');
  sel.append(auto);

  for (const src of list || []) {
    const o = document.createElement('option');
    o.value = src.name;
    o.textContent = src.label || src.name;
    sel.append(o);
  }
  sel.value = current;
}

function syncSoundNotice() {
  const box = $('audio-warn');
  if (!box || !S) return;
  let msg = '';
  if (!S.audioOK) msg = t('noSound');
  else if (S.audioErr) msg = `${t('soundFailed')} ${S.audioErr}`;
  box.textContent = msg;
  box.hidden = !msg;
}

// ── запуск ──────────────────────────────────────────────────────────

async function load() {
  S = await (await fetch('/api/state')).json();
  keys = S.keys;
  selection = new Set(S.state.selection || []);

  effectsById.clear();
  for (const d of S.effects) effectsById.set(d.id, d);
  $('effect').value = S.state.effect;
  $('overlay').value = S.state.overlay || '';
  applyLang(S.state.lang || 'en');
  $('lang').value = LANG;

  const p = S.state.params;
  for (const [id, s] of Object.entries(SLIDERS)) $(id).value = Math.round(p[id] * s.div);
  $('color1').value = p.color1;
  $('color2').value = p.color2;
  $('reactColor').value = p.reactColor;
  $('background').value = p.background || '#000000';
  $('rainbow').checked = p.rainbow;
  $('reverse').checked = !!p.reverse;
  $('variant').value = S.state.variant;
  $('fps').value = S.state.fps;
  $('mask').checked = S.state.maskSelection;
  $('autostart').checked = S.state.autostart;
  $('selcount').textContent = selection.size;

  $('dot').classList.toggle('on', !!S.connected);
  $('devtext').textContent = S.connected ? S.device + ' · ' + S.leds : '—';
  notice.input = S.inputOk ? '' : t('noInput');
  fillDevices();
  syncDeviceNotice();

  fillPresets();
  syncSoundNotice();
  await fillSources();

  showValues();
  syncParamVisibility();
  buildKeyboard();
}

function bind() {
  document.querySelectorAll('.tab').forEach((b) => {
    b.addEventListener('click', () => showTab(b.dataset.pane));
  });
  showTab('fx');

  $('effect').addEventListener('change', (e) => {
    post('/effect', { id: e.target.value });
    syncParamVisibility();
  });
  $('overlay').addEventListener('change', (e) => {
    post('/overlay', { id: e.target.value });
    syncParamVisibility();
  });

  const sendParams = () => { showValues(); post('/params', readParams()); };
  for (const id of Object.keys(SLIDERS)) $(id).addEventListener('input', sendParams);
  for (const id of ['color1', 'color2', 'reactColor', 'background']) {
    $(id).addEventListener('input', sendParams);
  }
  $('rainbow').addEventListener('change', sendParams);
  $('reverse').addEventListener('change', sendParams);

  $('audio-src').addEventListener('change', async (e) => {
    await post('/audio/source', { name: e.target.value });
    // Захват перезапускается не мгновенно, дадим ему подняться.
    setTimeout(async () => { S = await (await fetch('/api/state')).json(); syncSoundNotice(); }, 600);
  });

  const savePreset = async () => {
    const name = $('preset-name').value.trim();
    if (!name) return;
    const res = await post('/preset/save', { name });
    if (res && !res.ok) { warn(t('presetNameTaken')); return; }
    $('preset-name').value = '';
    await load();
  };
  $('preset').addEventListener('change', async (e) => {
    syncPresetButtons();
    if (!e.target.value) return;
    await post('/preset/apply', { id: e.target.value });
    const chosen = e.target.value;
    await load();
    // load() перечитывает список и сбрасывает выбор — возвращаем его,
    // чтобы было видно, какая схема сейчас включена.
    $('preset').value = chosen;
    syncPresetButtons();
  });

  $('preset-del').addEventListener('click', async () => {
    const name = $('preset').value;
    if (!name) return;
    await post('/preset/delete', { name });
    await load();
  });

  $('preset-save').addEventListener('click', savePreset);
  $('preset-name').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') savePreset();
  });

  $('variant').addEventListener('change', async (e) => {
    await post('/variant', { variant: e.target.value });
    await load();
  });
  $('fps').addEventListener('input', () => {
    showValues();
    post('/fps', { fps: Number($('fps').value) });
  });
  $('mask').addEventListener('change', (e) => {
    post('/mask', { on: e.target.checked });
    renderNotice();
  });
  $('device').addEventListener('change', async (e) => {
    const r = await post('/device', { path: e.target.value });
    if (r && !r.ok) warn(t('deviceFailed')(await r.text()));
    else warn('');
    S = await (await fetch('/api/state')).json();
    $('devtext').textContent = S.connected ? S.device + ' · ' + S.leds : '—';
    $('dot').classList.toggle('on', !!S.connected);
    fillDevices();
    syncDeviceNotice();
  });

  $('autostart').addEventListener('change', (e) => post('/autostart', { on: e.target.checked }));
  $('lang').addEventListener('change', (e) => {
    applyLang(e.target.value);
    post('/lang', { lang: e.target.value });
  });
  $('quit').addEventListener('click', () => post('/quit'));

  $('sel-all').addEventListener('click', () => {
    selection = new Set(keys.map((k) => k.index));
    pushSelection();
  });
  $('sel-none').addEventListener('click', () => { selection.clear(); pushSelection(); });
  $('sel-inv').addEventListener('click', () => {
    selection = new Set(keys.map((k) => k.index).filter((i) => !selection.has(i)));
    pushSelection();
  });

  const paint = (color) => {
    post('/paint', { indices: [...selection], color });
    $('effect').value = 'static';
    syncParamVisibility();
  };
  const sw = $('swatches');
  for (const c of SWATCHES) {
    const b = document.createElement('button');
    b.className = 'sw';
    b.style.background = c;
    b.dataset.color = c;
    b.setAttribute('aria-label', `${t('customColour')} ${c}`);
    b.addEventListener('click', () => paint(c));
    sw.appendChild(b);
  }
  $('paintpick').addEventListener('input', (e) => paint(e.target.value));
  $('clearpaint').addEventListener('click', () => post('/clear-paint'));

  $('pause').addEventListener('click', (e) => {
    const on = e.target.getAttribute('aria-pressed') !== 'true';
    e.target.setAttribute('aria-pressed', String(on));
    e.target.classList.toggle('on', on);
    e.target.textContent = on ? t('resume') : t('pause');
    post('/pause', { on });
  });
  $('blackout').addEventListener('click', () => {
    post('/blackout');
    const p = $('pause');
    p.setAttribute('aria-pressed', 'true');
    p.classList.add('on');
    p.textContent = t('resume');
  });

  window.addEventListener('resize', layoutCanvas);
}

let lastFPS = '';

let source = null;

// Пока окно не видно, поток кадров не нужен: браузер всё равно не рисует, а
// разбор сообщений и работа сервера продолжались бы впустую.
function watchVisibility() {
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      if (source) { source.close(); source = null; }
    } else if (!source) {
      prevHex = ''; // за время простоя картинка изменилась, рисуем заново
      stream();
    }
  });
}

function stream() {
  const es = new EventSource('/api/frames');
  source = es;
  es.onmessage = (e) => {
    // Поток восстанавливается сам, а сообщение о потере связи оставалось
    // висеть до перезагрузки страницы и пугало зря.
    if (notice.link) warn('');
    const [hex, fps, conn, level] = e.data.split('|');
    queueFrame(hex, level === undefined ? undefined : Math.min(1, Number(level) || 0));
    // Счётчик кадров меняется раз в секунду, ему очередь не нужна.
    if (fps && fps !== lastFPS) { lastFPS = fps; $('fps-real').textContent = fps + t('fpsUnit'); }
    const online = conn === '1';
    if (online !== lastOnline) {
      lastOnline = online;
      $('dot').classList.toggle('on', online);
      // Состояние берём с сервера целиком: причина отсутствия устройства
      // живёт там, и угадывать её в потоке кадров незачем.
      fetch('/api/state').then((r) => r.json()).then((st) => {
        S = st;
        $('devtext').textContent = S.connected ? S.device + ' · ' + S.leds : '—';
        fillDevices();
        syncDeviceNotice();
      });
    }
  };
  es.onerror = () => {
    $('dot').classList.remove('on');
    warn(t('noLink'));
  };
}

load().then(() => {
  bind();
  setupSelection();
  stream();
  watchVisibility();
  startWatchdog();
}).catch((e) => warn(String(e && e.message ? e.message : e)));
