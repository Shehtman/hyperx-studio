# Changelog

## 1.0.0 — 2026-08-27

First release.

- Direct `hidraw` driver for the HyperX Alloy Origins — OpenRGB is not required
- Static binary with no dependencies; the interface is embedded via `embed`
- 12 software effects plus a reactive overlay layer with its own parameters
- Global keypress reading through evdev, without grabbing the device
- Interactive layout: click, Ctrl-click and rubber-band selection; per-key fill
- ANSI and ISO layouts, including the L-shaped ISO Enter
- Automatic reconnection when the keyboard is replugged
- Settings persist across restarts, written atomically
- English and Russian interface, switchable at runtime
- Debian package with a udev rule
