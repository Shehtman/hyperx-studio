#!/usr/bin/env bash
# Сборка .deb. Один пакет, ни одной зависимости: бинарник статический.
#
#   DEBFULLNAME="Имя" DEBEMAIL="you@example.com" ./build-deb.sh

set -euo pipefail
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VER="$(cat "$SRC/VERSION")"
REV="1"
ARCH="$(dpkg --print-architecture)"
BUILD="$SRC/build"
OUT="$SRC/dist"

MAINTAINER="${DEBFULLNAME:-HyperX Studio} <${DEBEMAIL:-nobody@localhost}>"
HOMEPAGE="${DEBHOMEPAGE:-https://github.com/Shehtman/hyperx-studio}"

say() { printf '\n\033[1;36m==>\033[0m %s\n' "$1"; }
command -v dpkg-deb >/dev/null || { echo "нужен dpkg-deb"; exit 1; }

rm -rf "$BUILD"; mkdir -p "$BUILD" "$OUT"
PKG="$BUILD/hyperx-studio"
mkdir -p "$PKG/DEBIAN" "$PKG/usr/bin" "$PKG/usr/share/applications" \
         "$PKG/usr/lib/udev/rules.d" "$PKG/usr/share/doc/hyperx-studio" \
         "$PKG/usr/lib/systemd/system-sleep"

say "Собираю статический бинарник"
CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=$VER" \
    -o "$PKG/usr/bin/hyperx-studio" ./cmd/hyperx-studio

# правило доступа берём из самой программы, чтобы оно не разъехалось с кодом
"$PKG/usr/bin/hyperx-studio" --print-udev > "$PKG/usr/lib/udev/rules.d/60-hyperx-studio.rules"

cat > "$PKG/usr/share/applications/hyperx-studio.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=HyperX Studio
Comment=Подсветка клавиатуры HyperX Alloy Origins
Exec=/usr/bin/hyperx-studio
Icon=input-keyboard
Terminal=false
Categories=Utility;Settings;HardwareSettings;
StartupWMClass=hyperx-studio
DESKTOP

cat > "$PKG/usr/lib/systemd/system-sleep/hyperx-studio" <<'HOOK'
#!/bin/sh
# Клавиатуру нужно отпустить перед сном системы.
#
# Пока программа рисует подсветку, клавиатура держится в прямом режиме. Если
# оставить её в нём на время сна, она перестаёт будить компьютер — и тянет за
# собой мышь, если та сидит на том же контроллере USB. Перед сном снимаем
# прямой режим, после пробуждения возвращаем подсветку.
#
# systemd ждёт завершения этого скрипта, поэтому успеваем до заморозки.

BIN=/usr/bin/hyperx-studio
[ -x "$BIN" ] || exit 0

# Настройки лежат в домашнем каталоге владельца, поэтому команду выполняем от
# его имени, а не от root.
run() {
    user=$(ps -o user= -C hyperx-studio 2>/dev/null | head -n1 | tr -d ' ')
    if [ -n "$user" ] && [ "$user" != root ] && command -v runuser >/dev/null 2>&1
    then
        runuser -u "$user" -- "$BIN" "$@" || true
    else
        "$BIN" "$@" || true
    fi
}

case "$1" in
    pre)
        run --sleep
        ;;
    post)
        # Если программа не запущена, возвращать нечего: подсветкой снова
        # заведует сама клавиатура.
        pgrep -x hyperx-studio >/dev/null 2>&1 && run --wake
        ;;
esac
exit 0
HOOK

install -m 644 "$SRC/README.md" "$PKG/usr/share/doc/hyperx-studio/README.md"
cat > "$PKG/usr/share/doc/hyperx-studio/copyright" <<EOF
Upstream-Name: hyperx-studio
Source: $HOMEPAGE

Files: *
License: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
EOF

printf 'hyperx-studio (%s-%s) unstable; urgency=low\n\n  * Первый выпуск.\n\n -- %s  %s\n' \
    "$VER" "$REV" "$MAINTAINER" "$(date -R)" \
    | gzip -9n > "$PKG/usr/share/doc/hyperx-studio/changelog.Debian.gz"

INSTALLED_KB=$(du -sk "$PKG/usr" | cut -f1)
cat > "$PKG/DEBIAN/control" <<EOF
Package: hyperx-studio
Version: $VER-$REV
Section: utils
Priority: optional
Architecture: $ARCH
Installed-Size: $INSTALLED_KB
Maintainer: $MAINTAINER
Homepage: $HOMEPAGE
Description: Подсветка клавиатуры HyperX Alloy Origins
 Самостоятельная программа управления подсветкой: говорит с клавиатурой
 напрямую через hidraw, сама считает эффекты и сама отдаёт веб-интерфейс,
 вшитый в исполняемый файл.
 .
 Ни OpenRGB, ни каких-либо служб или библиотек не требуется: бинарник
 статический и не имеет зависимостей. Схема остаётся на клавиатуре после
 выхода из программы.
 .
 Реакция на нажатия читается через evdev без захвата устройства, поэтому
 обычный ввод не затрагивается.
EOF

cat > "$PKG/DEBIAN/postinst" <<'POSTINST'
#!/bin/sh
set -e
if [ "$1" = "configure" ]; then
    if command -v udevadm >/dev/null 2>&1; then
        udevadm control --reload-rules || true
        udevadm trigger --subsystem-match=hidraw --subsystem-match=input || true
    fi
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database -q /usr/share/applications || true
    fi
    echo "HyperX Studio установлена. Переподключите клавиатуру, чтобы права применились."
fi
exit 0
POSTINST

cat > "$PKG/DEBIAN/postrm" <<'POSTRM'
#!/bin/sh
set -e
if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
    if command -v udevadm >/dev/null 2>&1; then
        udevadm control --reload-rules || true
    fi
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database -q /usr/share/applications || true
    fi
fi
exit 0
POSTRM

find "$PKG" -type d -exec chmod 755 {} +
find "$PKG" -type f -not -path '*/DEBIAN/*' -exec chmod 644 {} +
# Хук сна обязан быть исполняемым: systemd молча пропускает файлы без +x.
chmod 755 "$PKG/usr/bin/hyperx-studio" "$PKG/DEBIAN/postinst" "$PKG/DEBIAN/postrm" \
          "$PKG/usr/lib/systemd/system-sleep/hyperx-studio"

say "Упаковываю"
fakeroot dpkg-deb --build --root-owner-group "$PKG" "$OUT" >/dev/null
DEB="$OUT/hyperx-studio_${VER}-${REV}_${ARCH}.deb"
say "Готово"
ls -lh "$DEB" | awk '{print "  " $9 "  " $5}'
echo
echo "Установка:  sudo apt install $DEB"
echo "Удаление:   sudo apt remove hyperx-studio"
