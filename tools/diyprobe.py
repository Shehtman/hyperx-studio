#!/usr/bin/env python3
"""Разведка протокола HyperX Alloy Origins через интерфейс 3.

Интерфейс 3 (MI_03) — вендорный канал: 64-байтные отчёты без Report ID,
драйвером не занят, поддерживает и запись, и чтение. Именно на него
ссылается NGENUITY.

Команды:
  read              показать буфер устройства
  mode N [сек]      послать 04 F2 .. NN (селектор режима) и подождать
  sweep A B [сек]   пройти селекторы от A до B, останавливаясь на каждом
  direct            вернуть прямой режим (04 F2 .. 09) и залить цвет
  reset             сброс USB — клавиатура возвращается к своему эффекту
  listen [сек]      слушать события клавиатуры (Fn-комбинации)
"""
import os, sys, time, select
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import usbhid

IFACE = 3

def open_dev():
    d = usbhid.Dev(); d.claim(IFACE); return d

def cmd_read():
    d = open_dev()
    r = d.get_report(IFACE, usbhid.TYPE_FEATURE)
    print(usbhid.dump(r)); d.close()

def send(d, payload):
    d.set_report(IFACE, usbhid.TYPE_FEATURE, payload)

def mode_packet(sel):
    p = bytearray(64); p[0] = 0x04; p[1] = 0xF2; p[8] = sel & 0xFF
    return p

def fill(d, rgb):
    """Залить всю клавиатуру одним цветом — видно, слушается ли прямой режим."""
    for _ in range(8):
        p = bytearray(64)
        for j in range(16):
            p[j*4:j*4+4] = bytes((0x81,) + rgb)
        send(d, p)

def cmd_mode(sel, hold):
    d = open_dev()
    p = mode_packet(sel)
    print(f"селектор 0x{sel:02x}: {bytes(p)[:12].hex(' ')}")
    send(d, p)
    r = d.get_report(IFACE, usbhid.TYPE_FEATURE)
    print(f"  ответ: {r[:12].hex(' ')}")
    time.sleep(hold)
    d.close()

def cmd_sweep(a, b, hold):
    d = open_dev()
    print("смотрите на клавиатуру; Ctrl+C — прервать\n")
    for sel in range(a, b + 1):
        send(d, mode_packet(sel))
        r = d.get_report(IFACE, usbhid.TYPE_FEATURE)
        print(f"селектор 0x{sel:02x}  ответ {r[:12].hex(' ')}", flush=True)
        time.sleep(hold)
    d.close()

def cmd_direct():
    d = open_dev()
    send(d, mode_packet(0x09))
    fill(d, (0x00, 0x40, 0xFF))
    print("прямой режим включён, залито синим")
    d.close()

def cmd_reset():
    import fcntl
    bus = int(open('/sys/bus/usb/devices/1-1/busnum').read())
    num = int(open('/sys/bus/usb/devices/1-1/devnum').read())
    fd = os.open(f"/dev/bus/usb/{bus:03d}/{num:03d}", os.O_WRONLY)
    fcntl.ioctl(fd, 0x5514, 0); os.close(fd)
    print("сброс выполнен")

def cmd_listen(secs):
    for node in ('/dev/hidraw1', '/dev/hidraw2', '/dev/hidraw0'):
        try:
            fd = os.open(node, os.O_RDONLY | os.O_NONBLOCK)
        except OSError:
            continue
        print(f"слушаю {node}")
        globals().setdefault('_fds', []).append((node, fd))
    fds = globals().get('_fds', [])
    if not fds:
        print("нет доступных узлов"); return
    print(f"{secs:.0f} с — нажимайте Fn-комбинации подсветки")
    t0 = time.time()
    while time.time() - t0 < secs:
        r, _, _ = select.select([f for _, f in fds], [], [], 0.4)
        for fd in r:
            node = next(n for n, f in fds if f == fd)
            try:
                data = os.read(fd, 64)
            except BlockingIOError:
                continue
            if data:
                print(f"[{time.time()-t0:6.2f}] {node}: {data.hex(' ')}", flush=True)
    for _, fd in fds:
        os.close(fd)

if __name__ == '__main__':
    a = sys.argv[1:]
    if not a: print(__doc__); sys.exit(0)
    c = a[0]
    if   c == 'read':   cmd_read()
    elif c == 'mode':   cmd_mode(int(a[1], 0), float(a[2]) if len(a) > 2 else 3)
    elif c == 'sweep':  cmd_sweep(int(a[1], 0), int(a[2], 0), float(a[3]) if len(a) > 3 else 2)
    elif c == 'direct': cmd_direct()
    elif c == 'reset':  cmd_reset()
    elif c == 'listen': cmd_listen(float(a[1]) if len(a) > 1 else 60)
    else: print(__doc__)
