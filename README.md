<div align="center">

<img src="docs/banner.svg" alt="HyperX Studio" width="100%">

**English** · [Русский](README.ru.md)

[![CI](https://github.com/Shehtman/hyperx-studio/actions/workflows/ci.yml/badge.svg)](https://github.com/Shehtman/hyperx-studio/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-22c55e?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Dependencies](https://img.shields.io/badge/dependencies-none-22c55e?style=flat-square)](go.mod)
[![Binary](https://img.shields.io/badge/binary-7.4%20MB%20static-8a8a8a?style=flat-square)](#build-from-source)
[![Platform](https://img.shields.io/badge/platform-Linux-8a8a8a?style=flat-square&logo=linux&logoColor=white)](#requirements)

### Per-key RGB control for the HyperX Alloy Origins on Linux

One static binary for the lighting, a native window for the interface.
No OpenRGB, no daemons, no browser.

</div>

> **Unofficial project.** Not affiliated with HP Inc. or HyperX. The device
> protocol was studied from the open OpenRGB implementation; the code here was
> written from scratch.

---

## Why

NGENUITY is Windows-only, and OpenRGB exposes this keyboard with a single mode:
`Direct`. There are no firmware effects available on Linux, so every animation
is computed in software and pushed frame by frame.

That turns out to be an advantage — nothing is limited by what the firmware
happens to support.

<table>
<tr><td width="50%" valign="top">

### What you get

- **Zero dependencies.** The lighting service is statically linked — `ldd`
  reports *not a dynamic executable*. Runs on any x86-64 Linux.
- **No middleman.** The keyboard is opened directly over `/dev/hidraw`; there is
  no server in between to accidentally close.
- **A window of its own** on GTK and WebKit — not a browser in disguise. Where
  the libraries are missing, the interface falls back to a browser and
  everything else works unchanged.
- **Lighting survives exit.** The device keeps the last frame as long as it has
  power.
- **16 effects** with a live preview of the actual frame being sent.
- **Global keypress reaction** through evdev, without grabbing the device.
- **Two languages** — English and Russian, switchable at runtime.

</td><td width="50%" valign="top">

<img src="docs/panel.svg" alt="Control panel" width="100%">

</td></tr>
</table>

---

## Requirements

| | |
|---|---|
| **OS** | Debian 11+ / Ubuntu 22.04+ or any x86-64 Linux |
| **Hardware** | HyperX Alloy Origins (`03f0:0591`), full-size. It also runs without one, as a preview window; another HID node can be picked by hand |
| **Runtime** | none — the binary is statically linked |
| **Window** | GTK 3 and WebKitGTK 4.1, already present on GNOME. Without them the interface opens in a browser |
| **Build** | Go 1.21+; the window also needs `libgtk-3-dev` and `libwebkit2gtk-4.1-dev` |

The window is drawn by a separate program, `hyperx-studio-window`. The lighting
service does not depend on it: without the window it runs exactly as before,
and the interface opens in a browser instead.

## Install

Grab the package from [Releases](https://github.com/Shehtman/hyperx-studio/releases) and install it:

```bash
sudo apt install ./hyperx-studio_1.2.3-1_amd64.deb
```

The package ships the binary, a desktop entry and a udev rule. **Unplug and
plug the keyboard back in afterwards** — permissions have to be applied anew.

### Build from source

```bash
git clone https://github.com/Shehtman/hyperx-studio.git
cd hyperx-studio
go build -o hyperx-studio ./cmd/hyperx-studio
go build -o hyperx-studio-window ./cmd/hyperx-window   # the window, optional
sudo ./hyperx-studio --install-udev
```

## Usage

```bash
hyperx-studio              # open the window
hyperx-studio --no-window  # run in the background (used by autostart)
hyperx-studio --quit       # stop the running instance
hyperx-studio --apply      # apply the saved scheme and exit
hyperx-studio --off        # turn the lighting off and exit
hyperx-studio --browser    # show the interface in a browser instead of our own window
```

The window can be closed — lighting keeps running. Launching again shows the
window of the running instance instead of starting a second one.

`--apply` is meant for static schemes: colours are set, the process exits, and
the keyboard keeps them. Animations need the program running.

## Effects

| | | |
|---|---|---|
| **Static** | **Breathing** | **Spectrum cycle** |
| **Rainbow wave** | **Two-colour wave** | **Gradient** |
| **Twinkle** | **Rain** | **Fire** |
| **Snake** | **Equaliser** | **Sound spectrum** |
| **Sound pulse** | **Wave to music** | |
| **Key ripple** | **Key flash** | |

The last two are reactive and can be layered on top of any other effect, with
their own speed, fade and colour.

Every effect takes brightness, and most take speed, angle, scale, density,
colours, background, saturation and direction — the panel shows only the ones
the current effect actually uses.

### Schemes

Fifteen ready-made schemes come built in — Aurora, Matrix, Starfield, Lava,
Cyberpunk, Typewriter, Bass drop and more. One click sets the effect, the
overlay and every parameter at once. Your own combination can be saved under
any name and appears in the same list.

### Sound

Four effects follow whatever is playing. The audio is taken from the system
output, so it reacts to anything — a player, a browser tab, a game.

This is the one place where an outside program is needed: the stream comes from
`parec` (`pulseaudio-utils`) or `pw-record` (`pipewire`), one of which is
already present on a normal Ubuntu desktop. Without them the rest of the
program works exactly as before and the sound effects simply say so.

Capture starts when a sound effect is selected and stops when it is not — no
microphone or output is held open in the background. By default the source is
the monitor of the current output device; any other source can be picked in the
panel.

## Persistence

Three cases behave differently, and the distinction matters:

| Situation | What happens |
|---|---|
| Application closed, even by `kill -9` | Keyboard keeps displaying the last frame |
| Machine rebooted | Lighting returns to the factory scheme |
| Machine suspended | Keyboard is released for the duration of sleep, scheme comes back on wake-up |
| Next launch | Effect, parameters, per-key colours and selection are restored |

Schemes cannot be written into the keyboard's own memory: the device only
accepts direct frames. Enabling **Start on login** is the only way to have your
lighting back after a reboot.

Settings are written to `~/.config/hyperx-studio/config.json` 1.5 seconds after
any change, atomically through a temporary file.

### Sleep

While the program is running, the keyboard is held in direct lighting mode. A
keyboard left in that mode with no frames arriving stops signalling remote
wakeup — and since it usually shares a USB controller with the mouse, the
computer then cannot be woken by either.

So the keyboard is handed back before the machine sleeps. The package installs
a hook in `/usr/lib/systemd/system-sleep/`; `systemd` waits for it, so this
always happens before processes are frozen:

```bash
hyperx-studio --sleep   # release the keyboard, restore its own mode
hyperx-studio --wake    # take it back and repaint the saved scheme
```

There is no protocol command for leaving direct mode — neither OpenRGB nor
NGENUITY knows one. The mode is cleared by asking the kernel to reinitialise
the device, which the program does through `USBDEVFS_RESET`.

## How it works

```
┌────────────┐   HID feature reports   ┌──────────────────┐
│  effects   │ ──────────────────────▶ │ Alloy Origins    │
│  engine    │      /dev/hidraw        │ 107 LEDs         │
└────────────┘                         └──────────────────┘
      ▲                                          │
      │ keystrokes                               │ USB
      │ /dev/input/event*                        ▼
┌────────────┐                          ┌──────────────────┐
│  reactive  │                          │  your fingers    │
│  layer     │ ◀────────────────────────│                  │
└────────────┘                          └──────────────────┘
```

A frame is nine 65-byte feature reports: one that switches the keyboard into
direct mode and eight carrying colours, 16 per packet, 4 bytes each — an `0x81`
marker plus three components.

There are 126 positions in a frame but only 107 LEDs: nineteen positions are
physically absent and must still receive zeroes, otherwise the layout shifts
and colours land on neighbouring keys.

<details>
<summary><b>Details worth knowing</b></summary>

**Mode switching happens on every frame,** not once at open time. Send it only
once and the keyboard eventually falls back to the scheme stored in its own
memory, overriding what the program draws.

**Device lookup is by identifiers, not by path.** The `/dev/hidrawN` number
changes when the cable is replugged, so the program searches for the keyboard
and always takes USB interface `00` — the one that owns the lighting. If a
frame fails to send, the device is reopened automatically.

**Keystrokes are stamped in the same clock as frames.** Mixing the absolute
`time.perf_counter`-style clock with time-since-start sends hit ages far
negative: ripples get a negative radius and never draw, flashes stick at full
brightness. There is a test for exactly that.

**Reactive input never grabs the device**, so typing is unaffected.

</details>

## ANSI or ISO

The layout has to match your keyboard; it is not detected automatically. Tell
them apart by the Enter key:

- **ANSI** — Enter is one row tall, with a separate `\ |` key above it.
- **ISO** — Enter is L-shaped across two rows, with `#` to the left of its
  lower part and an extra key next to `Z`.

## Troubleshooting

<details>
<summary><b>“Keyboard not found”</b></summary>

The udev rule is missing, so the device is only visible to root:

```bash
sudo hyperx-studio --install-udev
```

Then physically reconnect the keyboard — `udevadm trigger` does not reliably
re-apply permissions to an already-connected device. Inspect the rule with
`hyperx-studio --print-udev`.

</details>

<details>
<summary><b>Reactive effects do nothing</b></summary>

No read access to `/dev/input/event*`. The udev rule grants it through the
`uaccess` tag; otherwise add yourself to the `input` group and log back in.

</details>

<details>
<summary><b>The computer will not wake from sleep</b></summary>

Fixed in 1.0.1. If you are upgrading from 1.0.0, make sure the sleep hook is in
place:

```bash
ls -l /usr/lib/systemd/system-sleep/hyperx-studio
```

If the file is missing, reinstall the package. No reboot is needed — `systemd`
picks the hook up on the next suspend.
</details>

<details>
<summary><b>Nothing lights up except one key</b></summary>

**Selected only** is enabled. It deliberately blanks every key outside the
selection; a notice in the panel says how many are lit while it is on.

</details>

## Development

```bash
go test ./...           # all tests, no hardware required
go run ./cmd/gen-assets # regenerate README artwork from the real layout
./build-deb.sh          # build the .deb
```

| Package | Responsibility |
|---|---|
| `internal/keyboard` | hidraw protocol, frame layout across slots |
| `internal/layout` | key geometry, LED and evdev bindings |
| `internal/effects` | animation engine |
| `internal/input` | keystroke reading |
| `internal/engine` | state, render loop, persistence |
| `internal/webui` | interface, embedded into the binary |
| `internal/i18n` | command-line messages |

Tests cover the frame layout, LED bindings, reactive timing, settings
persistence and translation completeness — none of them need the keyboard.

## Contributing

Issues and pull requests are welcome at [Shehtman/hyperx-studio](https://github.com/Shehtman/hyperx-studio).
Run `go test ./...` before sending a change — the whole suite works without the
keyboard.

## License

[MIT](LICENSE).

Hardware knowledge comes from [OpenRGB](https://openrgb.org) (GPLv2); this is an
independent implementation and contains none of its source.
