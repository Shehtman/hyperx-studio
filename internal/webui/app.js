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
    reaction: 'Keypress reaction', fade: 'Fade', rainbow: 'Rainbow',
    selAll: 'All', selNone: 'Clear', selInvert: 'Invert',
    customColour: 'Custom', resetPaint: 'Reset', selectedOnly: 'Selected only',
    layout: 'Layout', fps: 'Frames/s', autostart: 'Start on login',
    language: 'Language',
    customColourAria: 'Fill with a custom colour', keyboardAria: 'Keyboard layout',
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
    },
  },
  ru: {
    pause: 'Пауза', resume: 'Продолжить', blackout: 'Погасить',
    effect: 'Эффект', overlay: 'Поверх', none: 'нет',
    brightness: 'Яркость', speed: 'Скорость', angle: 'Угол', scale: 'Масштаб',
    density: 'Плотность', length: 'Длина', color: 'Цвет', color2: 'Второй',
    reaction: 'Реакция на нажатия', fade: 'Затухание', rainbow: 'Радуга',
    selAll: 'Все', selNone: 'Снять', selInvert: 'Инверт.',
    customColour: 'Свой', resetPaint: 'Сброс', selectedOnly: 'Только выбранные',
    layout: 'Раскладка', fps: 'Кадров/с', autostart: 'Автозапуск',
    language: 'Язык',
    customColourAria: 'Залить своим цветом', keyboardAria: 'Раскладка клавиатуры',
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
    },
  },
};

let LANG = 'en';
const t = (key) => (STRINGS[LANG] || STRINGS.en)[key];
const fxName = (id) => ((STRINGS[LANG] || STRINGS.en).fx[id] || id);

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

  fillEffects();
  const paused = $('pause').getAttribute('aria-pressed') === 'true';
  $('pause').textContent = paused ? t('resume') : t('pause');
  document.querySelectorAll('.sw').forEach((b) => {
    b.setAttribute('aria-label', `${t('customColour')} ${b.dataset.color}`);
  });
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
let rects = new Map();     // индекс светодиода -> <rect>
let effectsById = new Map();

// ── обмен с сервером ────────────────────────────────────────────────

async function post(path, body) {
  try {
    await fetch('/api' + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body ?? {}),
    });
  } catch (e) {
    warn(t('noLink'));
  }
}

// Одна строка сообщений на все случаи. Приоритет: сначала то, что мешает
// работать, потом то, что объясняет неожиданный результат.
let notice = { device: '', input: '', link: '' };

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

const SVGNS = 'http://www.w3.org/2000/svg';
const KEY_OFF = '#1c1c1c';    // погасшая клавиша
const KEY_EDGE = '#333333';
const KEY_SEL = '#f0f0f0';
const U = 60;      // пикселей на юнит во внутренней системе координат
const GAP = 4;

// Верхняя часть Enter на ISO шире нижней: рисуем контур, а не прямоугольник,
// иначе клавиша накрывает соседнюю.
function isoEnterPath(k) {
  const x0 = k.x * U + GAP / 2;
  const y0 = k.y * U + GAP / 2;
  const x1 = (k.x + k.w) * U - GAP / 2;
  const y1 = (k.y + k.h) * U - GAP / 2;
  const notchX = (k.x + 0.25) * U + GAP / 2;
  const notchY = (k.y + 1) * U + GAP / 2;
  const p = document.createElementNS(SVGNS, 'path');
  p.setAttribute('d',
    `M${x0},${y0} L${x1},${y0} L${x1},${y1} L${notchX},${y1} ` +
    `L${notchX},${notchY} L${x0},${notchY} Z`);
  p.setAttribute('stroke-linejoin', 'round');
  return p;
}

function buildKeyboard() {
  const svg = $('kb');
  svg.textContent = '';
  rects.clear();

  const w = S.width * U, h = S.height * U;
  svg.setAttribute('viewBox', `0 0 ${w} ${h}`);

  for (const k of keys) {
    const g = document.createElementNS(SVGNS, 'g');

    const r = k.shape === 'iso-enter' ? isoEnterPath(k)
                                      : document.createElementNS(SVGNS, 'rect');
    if (k.shape !== 'iso-enter') {
      r.setAttribute('x', k.x * U + GAP / 2);
      r.setAttribute('y', k.y * U + GAP / 2);
      r.setAttribute('width', k.w * U - GAP);
      r.setAttribute('height', k.h * U - GAP);
      r.setAttribute('rx', 5);
    }
    r.setAttribute('class', 'key');
    r.setAttribute('fill', KEY_OFF);
    r.setAttribute('stroke', KEY_EDGE);
    r.setAttribute('stroke-width', 2);
    r.dataset.index = k.index;
    g.appendChild(r);

    if (k.label) {
      const t = document.createElementNS(SVGNS, 'text');
      const cx = k.shape === 'iso-enter' ? k.x + 0.25 + (k.w - 0.25) / 2 : k.x + k.w / 2;
      t.setAttribute('x', cx * U);
      t.setAttribute('y', (k.y + k.h / 2) * U);
      t.setAttribute('text-anchor', 'middle');
      t.setAttribute('dominant-baseline', 'central');
      t.setAttribute('font-size', k.label.length > 2 ? 13 : 17);
      t.setAttribute('fill', '#c9c9d4');
      t.textContent = k.label;
      g.appendChild(t);
    }

    svg.appendChild(g);
    rects.set(k.index, r);
  }
  paintSelection();
}

// Кадр приходит слитной hex-строкой: по 6 символов на светодиод.
function applyFrame(hex) {
  for (const [idx, r] of rects) {
    const off = idx * 6;
    const c = '#' + hex.slice(off, off + 6);
    if (r.getAttribute('fill') !== c) r.setAttribute('fill', c);
    const lum = parseInt(hex.slice(off, off + 2), 16) * 0.299
              + parseInt(hex.slice(off + 2, off + 4), 16) * 0.587
              + parseInt(hex.slice(off + 4, off + 6), 16) * 0.114;
    const label = r.nextSibling;
    if (label) label.setAttribute('fill', lum > 140 ? '#101014' : '#c9c9d4');
  }
}

function paintSelection() {
  if (typeof renderNotice === 'function') setTimeout(renderNotice, 0);
  for (const [idx, r] of rects) {
    const on = selection.has(idx);
    r.setAttribute('stroke', on ? KEY_SEL : KEY_EDGE);
    r.setAttribute('stroke-width', on ? 3.5 : 2);
  }
  $('selcount').textContent = selection.size;
}

function pushSelection() {
  paintSelection();
  post('/selection', { indices: [...selection] });
}

// ── выделение мышью ─────────────────────────────────────────────────

function svgPoint(evt) {
  const svg = $('kb');
  const r = svg.getBoundingClientRect();
  const vb = svg.viewBox.baseVal;
  return {
    x: (evt.clientX - r.left) / r.width * vb.width,
    y: (evt.clientY - r.top) / r.height * vb.height,
  };
}

function setupSelection() {
  const svg = $('kb');
  let start = null, band = null, additive = false, moved = false;

  svg.addEventListener('mousedown', (e) => {
    if (e.button !== 0) return;
    additive = e.ctrlKey || e.shiftKey;
    start = svgPoint(e);
    moved = false;
    e.preventDefault();
  });

  svg.addEventListener('mousemove', (e) => {
    if (!start) return;
    const p = svgPoint(e);
    if (Math.abs(p.x - start.x) < 6 && Math.abs(p.y - start.y) < 6) return;
    moved = true;
    if (!band) {
      band = document.createElementNS(SVGNS, 'rect');
      band.setAttribute('fill', 'rgba(34,197,94,.15)');
      band.setAttribute('stroke', '#22C55E');
      band.setAttribute('stroke-dasharray', '6 4');
      svg.appendChild(band);
    }
    const x = Math.min(start.x, p.x), y = Math.min(start.y, p.y);
    const w = Math.abs(p.x - start.x), h = Math.abs(p.y - start.y);
    band.setAttribute('x', x); band.setAttribute('y', y);
    band.setAttribute('width', w); band.setAttribute('height', h);

    const hit = new Set();
    for (const k of keys) {
      const kx = k.x * U, ky = k.y * U, kw = k.w * U, kh = k.h * U;
      if (kx < x + w && kx + kw > x && ky < y + h && ky + kh > y) hit.add(k.index);
    }
    selection = additive ? new Set([...selection, ...hit]) : hit;
    paintSelection();
  });

  const finish = (e) => {
    if (!start) return;
    if (!moved) {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const idx = el && el.dataset && el.dataset.index !== undefined
        ? Number(el.dataset.index) : null;
      if (idx === null) {
        if (!additive) selection.clear();
      } else if (additive) {
        selection.has(idx) ? selection.delete(idx) : selection.add(idx);
      } else {
        selection = new Set([idx]);
      }
    }
    if (band) { band.remove(); band = null; }
    start = null;
    pushSelection();
  };

  svg.addEventListener('mouseup', finish);
  svg.addEventListener('mouseleave', (e) => { if (start) finish(e); });
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
  reactFade: { div: 100, fmt: (v) => v.toFixed(2) + ' с' },
};

function readParams() {
  const p = {};
  for (const [id, s] of Object.entries(SLIDERS)) p[id] = Number($(id).value) / s.div;
  p.length = Math.round(p.length);
  p.color1 = $('color1').value;
  p.color2 = $('color2').value;
  p.reactColor = $('reactColor').value;
  p.rainbow = $('rainbow').checked;
  return p;
}

function showValues() {
  for (const [id, s] of Object.entries(SLIDERS)) {
    $(id + '-v').textContent = s.fmt(Number($(id).value) / s.div);
  }
  $('fps-v').textContent = $('fps').value;
}

function syncParamVisibility() {
  const base = effectsById.get($('effect').value);
  const ovl = effectsById.get($('overlay').value);
  const uses = new Set(base ? base.uses || [] : []);
  document.querySelectorAll('#params .row[data-p]').forEach((row) => {
    const p = row.dataset.p;
    row.style.display = (p === 'brightness' || uses.has(p)) ? '' : 'none';
  });
  const reactive = (base && base.reactive) || !!ovl;
  $('react').style.display = reactive ? '' : 'none';
  $('paint').style.display = $('effect').value === 'static' ? '' : 'none';
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
  $('rainbow').checked = p.rainbow;
  $('variant').value = S.state.variant;
  $('fps').value = S.state.fps;
  $('mask').checked = S.state.maskSelection;
  $('autostart').checked = S.state.autostart;

  $('dot').classList.add('on');
  $('devtext').textContent = S.device + ' · ' + S.leds;
  notice.input = S.inputOk ? '' : t('noInput');
  notice.device = S.connected ? '' : t('disconnected');
  renderNotice();

  showValues();
  syncParamVisibility();
  buildKeyboard();
}

function bind() {
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
  for (const id of ['color1', 'color2', 'reactColor']) $(id).addEventListener('input', sendParams);
  $('rainbow').addEventListener('change', sendParams);

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
  $('autostart').addEventListener('change', (e) => post('/autostart', { on: e.target.checked }));
  $('lang').addEventListener('change', (e) => {
    applyLang(e.target.value);
    post('/lang', { lang: e.target.value });
  });

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
    $('pause').setAttribute('aria-pressed', 'true');
    $('pause').classList.add('on');
    $('pause').textContent = 'Продолжить';
  });
}

function stream() {
  const es = new EventSource('/api/frames');
  es.onmessage = (e) => {
    const [hex, fps, conn] = e.data.split('|');
    applyFrame(hex);
    if (fps) $('fps-real').textContent = fps + t('fpsUnit');
    const online = conn === '1';
    $('dot').classList.toggle('on', online);
    const msg = online ? '' : t('disconnected');
    if (msg !== notice.device) { notice.device = msg; renderNotice(); }
  };
  es.onerror = () => {
    $('dot').classList.remove('on');
    warn('Связь с приложением потеряна.');
  };
}

load().then(() => { bind(); setupSelection(); stream(); })
      .catch((e) => warn('Ошибка: ' + e.message));
