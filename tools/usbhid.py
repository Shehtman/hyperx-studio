"""Минимальный доступ к USB-устройству через usbfs: контрольные передачи
и interrupt OUT. Никаких библиотек — только ioctl."""
import ctypes, fcntl, os

class Ctrl(ctypes.Structure):
    _fields_ = [("bRequestType", ctypes.c_ubyte), ("bRequest", ctypes.c_ubyte),
                ("wValue", ctypes.c_ushort), ("wIndex", ctypes.c_ushort),
                ("wLength", ctypes.c_ushort), ("timeout", ctypes.c_uint),
                ("data", ctypes.c_void_p)]

class Bulk(ctypes.Structure):
    _fields_ = [("ep", ctypes.c_uint), ("len", ctypes.c_uint),
                ("timeout", ctypes.c_uint), ("data", ctypes.c_void_p)]

USBDEVFS_CONTROL        = 0xC0185500
USBDEVFS_BULK           = 0xC0185502
USBDEVFS_CLAIMINTERFACE = 0x8004550F
USBDEVFS_RELEASEINTERFACE = 0x80045510

HID_GET_REPORT, HID_SET_REPORT = 0x01, 0x09
TYPE_INPUT, TYPE_OUTPUT, TYPE_FEATURE = 1, 2, 3

class Dev:
    def __init__(self, sysname='1-1'):
        base = f'/sys/bus/usb/devices/{sysname}'
        bus = int(open(base + '/busnum').read())
        num = int(open(base + '/devnum').read())
        self.path = f"/dev/bus/usb/{bus:03d}/{num:03d}"
        self.fd = os.open(self.path, os.O_RDWR)
        self.claimed = set()

    def claim(self, iface):
        fcntl.ioctl(self.fd, USBDEVFS_CLAIMINTERFACE, ctypes.c_uint(iface))
        self.claimed.add(iface)

    def get_report(self, iface, rtype, rid=0, length=64, timeout=2000):
        buf = (ctypes.c_ubyte * length)()
        c = Ctrl(0xA1, HID_GET_REPORT, (rtype << 8) | rid, iface, length,
                 timeout, ctypes.cast(buf, ctypes.c_void_p))
        n = fcntl.ioctl(self.fd, USBDEVFS_CONTROL, c)
        return bytes(buf)[:n]

    def set_report(self, iface, rtype, payload, rid=0, timeout=2000):
        buf = (ctypes.c_ubyte * len(payload))(*payload)
        c = Ctrl(0x21, HID_SET_REPORT, (rtype << 8) | rid, iface, len(payload),
                 timeout, ctypes.cast(buf, ctypes.c_void_p))
        return fcntl.ioctl(self.fd, USBDEVFS_CONTROL, c)

    def write_ep(self, ep, payload, timeout=2000):
        buf = (ctypes.c_ubyte * len(payload))(*payload)
        b = Bulk(ep, len(payload), timeout, ctypes.cast(buf, ctypes.c_void_p))
        return fcntl.ioctl(self.fd, USBDEVFS_BULK, b)

    def close(self):
        for i in sorted(self.claimed):
            try: fcntl.ioctl(self.fd, USBDEVFS_RELEASEINTERFACE, ctypes.c_uint(i))
            except OSError: pass
        os.close(self.fd)

def dump(b, width=16):
    out = []
    for i in range(0, len(b), width):
        chunk = b[i:i+width]
        out.append(f"{i:04x}: {chunk.hex(' '):<{width*3}} ")
    return '\n'.join(out)
