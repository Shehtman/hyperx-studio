# Changelog

## 1.2.3 — 2026-09-02

- Fixed: the glow under the keys lived a life of its own. It was drawn on a
  second canvas blurred through CSS, and a blurred element gets its own
  compositing layer that the browser refreshes whenever it sees fit — WebKit
  refreshed it late, so patches of light kept burning where keys had gone dark
  several frames earlier. Keys and glow now share one canvas and have nowhere
  left to disagree
- The blur is done by hand instead: the keys are drawn into a small canvas and
  stretched back over the frame, twice at two different scales, for a tight
  core and a wide halo. It also costs slightly less than the CSS layer did

## 1.2.2 — 2026-09-02

- Fixed: "Turn off" did not keep the keyboard off. The render loop stopped
  talking to the device altogether, and the keyboard holds direct mode only
  while frames keep arriving — a few seconds later the firmware took over and
  lit it up again by itself. Pause had the same fault: instead of freezing the
  picture it handed the keyboard back to the factory rainbow. Both now keep
  sending, just the same frame over and over
- The pause and blackout buttons and the frame counter are gone from the
  window. Lighting can still be turned off from the command line with
  hyperx-studio --off
- Fixed: the engine tests reached for a real keyboard when one was plugged in
  and blacked it out for the duration of the run

## 1.2.1 — 2026-09-02

- Fixed: "Turn off" darkened the keyboard but not the window. The render loop
  simply stopped, so the last frame stayed on screen for the rest of the
  session — most visibly as the glow under the keys, which kept burning under
  a keyboard that was already dark. Blackout now sends a black frame to the
  preview as well
- Fixed: the frame stream dropped surplus frames instead of holding them back.
  Losing a frame costs nothing while another one follows, but the last frame
  before the render loop stops has no successor — which is how the blackout
  went missing. Frames are now deferred, and the preview gets an even twenty a
  second instead of a ragged seventeen
- The status strip along the top is gone. A device name, a frame counter and
  two buttons were not worth fifty pixels off the keyboard: the buttons and
  the counter moved to the right of the tab row, the device state sits next to
  the device picker in Setup
- The explanatory lines under Keys and Setup are gone, and the Keypress tab
  now hides itself when no reactive effect or overlay is on — an empty panel
  explaining why it is empty is still an empty panel

## 1.2.0 — 2026-09-02

- The interface has been laid out again. Controls moved from a narrow column
  on the right into a panel across the bottom, split into tabs — a keyboard is
  a wide object, and a side panel was squeezing it into a strip. Every control
  now sits in a grid of equal cells, so labels and fields line up in columns
  instead of each row choosing its own width. New palette, real sliders and
  fields of a single height throughout
- The preview is drawn on a canvas instead of SVG. A hundred keys used to mean
  a hundred attribute writes per frame, each costing a style recalculation and
  a fresh layout; the whole board is now painted in one pass, and the glow
  under the keys is a second, low-resolution canvas blurred by the browser on
  the graphics card
- Fixed: the preview showed twenty frames and froze. When the browser stops
  handing out animation frames, the pending request never completes and the
  flag guarding it stays raised for good — frames kept arriving and stopped
  reaching the screen for the rest of the session. An overdue request is now
  cancelled and the frame painted on a timer
- The window is our own, on GTK and WebKit — no more browser in application
  mode. The title strip is empty: only the window buttons are left, since the
  name is already in the taskbar. The shell recognises the window by our own
  desktop entry, so it carries our name and our icon
- The window is a separate program, hyperx-studio-window. The lighting service
  stays a static binary of its own with no dependencies, unchanged: it never
  loads GTK when running without a window, and a crash in WebKit costs the
  window, not the lighting
- The palette is always available on the Keys tab: picking a colour switches
  to Static by itself, so hiding it until then only got in the way
- The .desktop entry impersonating a Chrome window is gone with the browser
  that needed it

## 1.1.3 — 2026-08-30

- Fixed: sound effects froze. They only checked whether capture was running,
  not whether anything was actually playing. In silence every band reads zero,
  so "Equaliser" drew an empty screen and "Pulse" and "Spectrum" stood still —
  indistinguishable from a hung program. They now fall back to a slow moving
  standby wave whenever the signal is gone
- The window no longer refuses to start without an Alloy Origins. Effects are
  computed independently of the hardware, so the preview works, schemes can be
  edited, and the keyboard lights up the moment it is plugged in
- The preview keeps its normal frame rate while no keyboard is attached;
  previously it dropped to one frame a second
- A keyboard can be picked by hand from a list of all HID nodes, for anyone
  wanting to try the protocol on a different device. Nodes that are not an
  Alloy Origins are marked as such
- The title bar inside the window is gone — the window frame already carries
  the name. Status and the pause and blackout buttons moved into the panel,
  and the keyboard now gets the full width
- The panel's scrollbar no longer sits against the edge of the window
- The application ships its own icon, and the window is recognised by the
  desktop shell instead of being labelled with the browser's internal name.
  The browser gets no profile of its own — the window name does not depend on
  one, while a fresh profile means the whole first-run welcome and a hundred
  and sixty megabytes on disk. The first-run and default-browser prompts are
  suppressed outright
- Fixed: two data races. The render loop wrote the last-sent frame without a
  lock, and changing the device could close the file mid-transfer
- Fixed: "connection lost" stayed on screen after the stream recovered

## 1.1.2 — 2026-08-28

- Fixed: the window was sluggish. Preview frames were sent as fast as the
  keyboard receives them — up to sixty a second — and repainting a hundred keys
  in SVG costs far more than sending them to the device. The preview is now
  capped at twenty frames a second, the device still gets the full rate, and
  the stream is dropped entirely while the window is hidden
- Fixed: "Bass drop" looked dead. It followed overall loudness, which automatic
  gain keeps pinned near the top on steady music, so the keyboard just glowed
  one colour. It now follows the bass rising above its own running average, so
  hits actually read
- Schemes are a plain drop-down again instead of a wall of chips
- Identical frames are no longer resent to the keyboard, which cuts the control
  traffic on static and slow effects

## 1.1.1 — 2026-08-28

- Fixed: after upgrading the package the old version kept running. Installing
  replaces the file on disk but does not touch the process already in memory,
  and launching again only shows the running instance's window — so the new
  effects and schemes did not appear. The package now restarts a running
  instance
- Fixed: the interface is served with `Cache-Control`, so an open window picks
  up the new version instead of showing the cached one

## 1.1.0 — 2026-08-28

- Fixed: waves barely moved at normal speed. Time was divided by the scale, so
  the more stripes were on screen the slower the picture crawled — at scale 5
  the two-colour wave only started moving around 4x. Scale now sets the number
  of stripes and speed sets how fast they travel, independently
- Fixed: rain and twinkle advanced per frame instead of per second, so they ran
  at half speed at 30 frames/s compared to 60
- Sound reactive effects: equaliser, sound spectrum, sound pulse and wave to
  music. System audio is captured through `parec` or `pw-record`; the source is
  selectable and the capture starts and stops on its own
- 15 built-in schemes, from Aurora to Bass drop, plus saving your own
- New settings: background colour, saturation, sensitivity and reverse direction

## 1.0.1 — 2026-08-28

- Fixed: the computer could not be woken from sleep with the keyboard or the
  mouse. The keyboard was left in direct lighting mode with no frames coming
  in, stopped signalling remote wakeup, and took the whole USB controller down
  with it. The keyboard is now released before sleep and picked up again after
  wake-up, through a `systemd` sleep hook

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
