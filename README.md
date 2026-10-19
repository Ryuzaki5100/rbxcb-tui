# rbxcb-tui

A terminal UI for entering a 3x3x3 Rubik's cube by colour and solving it with
2-stage, 3-stage, or 4-stage algorithms.

The solver is a pure-Go port of the Java reference implementation in
[`Ryuzaki5100/rbxcb`](https://github.com/Ryuzaki5100/rbxcb), so the TUI needs no
JVM, no HTTP server, and no network access.

```text
┌ Puzzles ──────┬─ Cube ─────────────────────────────┬─ Actions ──────────┐
│ > 3x3x3       │          square cube editor        │ solver / actions   │
│   4x4x4       │                                    │                    │
│   2x2x2       ├─ Solution ─────────────────────────┤                    │
│               │ moves, playback status             │                    │
└───────────────┴────────────────────────────────────┴────────────────────┘
```

## Contents

- [Features](#features)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [CLI reference](#cli-reference)
- [Workspace](#workspace)
- [Solving flow](#solving-flow)
- [Controls](#controls)
- [Mouse and hover](#mouse-and-hover)
- [Square cells and auto-calibration](#square-cells-and-auto-calibration)
- [Themes](#themes)
- [Configuration](#configuration)
- [Facelet order](#facelet-order)
- [Architecture](#architecture)
- [Testing](#testing)
- [Java parity fixtures](#java-parity-fixtures)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Limitations](#limitations)

## Features

- Pure-Go 3x3x3 engine with no runtime dependency on Java.
- 2-stage, 3-stage, and 4-stage staged solvers.
- CFOP OLL/PLL solvers in the Go library.
- Interactive colour picker with square, theme-bordered colour chips.
- 3D cube-view navigation: each arrow direction is a four-step cycle.
- Playback that animates the cube state from scramble to solved.
- Re-solve the same input with a different stage without losing entered colours.
- Aspect-aware square cells with automatic terminal calibration.
- Theme picker with built-in and custom themes, persisted across launches.
- Mouse support for puzzles, palette, cells, arrows, actions, and solution moves.
- Hover highlighting for buttons, palette chips, face arrows, and cube cells.
- Monochrome mode with distinguishable per-colour patterns.
- Unit, parity, race, and randomized end-to-end solver tests.

## Requirements

- Go 1.27+
- A terminal at least `100×37`
- Optional: [devenv](https://devenv.sh) + Nix for the pinned environment
- Optional: `just` for the task shortcuts

The TUI is tested on macOS and Linux. Windows builds, but terminal aspect
auto-detection falls back to the configured/default value.

## Quick start

With Nix and devenv (recommended):

```bash
devenv shell
rbxcb-tui
```

One-shot:

```bash
devenv shell -- rbxcb-tui
```

With plain Go:

```bash
go run ./cmd/rbxcb-tui
```

With Nix flakes:

```bash
nix run .
```

Build a local binary:

```bash
just build          # ./bin/rbxcb-tui
```

## CLI reference

```text
rbxcb-tui [flags]

  --demo                 load the sample cube on startup
  --facelets <54 chars>  load a cube state
  --scramble <notation>  apply a scramble such as "R U R' U'"
  --stage <2|3|4>        default solver stage (default 4)
  --theme <name>         theme name (see --list-themes)
  --list-themes          list built-in and configured themes, then exit
  --cell-aspect <float>  manual terminal cell aspect override
  --no-auto-aspect       disable automatic aspect detection
  --no-color             force the monochrome theme
  --no-hover             disable hover highlighting
  --no-mouse             start with mouse capture disabled
  --version              print the version and exit
```

Environment variables:

| Variable | Purpose |
|---|---|
| `RBXCB_THEME` | One-session theme override |
| `NO_COLOR` | Force monochrome mode |
| `XDG_CONFIG_HOME` | Config directory root |

Examples:

```bash
rbxcb-tui --demo
rbxcb-tui --facelets ybgrbrrbbrgbooygorwyyrwyggboboorgwbryobyygyrwwwowgwowg --stage 2
rbxcb-tui --scramble "R U R' U'" --stage 3
rbxcb-tui --theme tokyo-night
rbxcb-tui --cell-aspect 1.75
rbxcb-tui --no-auto-aspect --cell-aspect 1.8
rbxcb-tui --list-themes
```

## Workspace

The TUI is arranged like a code editor.

### No puzzle selected

Only the puzzle list, middle pane, and output panel are shown. The right
actions pane and colour sidebar stay hidden until a puzzle is selected.

```text
┌ Puzzles ──────┬ Middle pane ───────────────────────────────────┐
│               │                                                │
│  > 3x3x3      │        Select a puzzle from the left pane      │
│    4x4x4      │                                                │
│    2x2x2      ├ Solution output ───────────────────────────────┤
│               │ No solution yet.                               │
└───────────────┴────────────────────────────────────────────────┘
```

### Puzzle selected

```text
┌ Puzzles ──────┬─ Cube ─────────────────────────────┬─ Actions ──────────┐
│ > 3x3x3       │          square cube editor        │ Solver             │
│   4x4x4       │                                    │ Actions            │
│   2x2x2       ├─ Solution ─────────────────────────┤ Playback           │
│               │ moves, playback status             │                    │
└───────────────┴────────────────────────────────────┴────────────────────┘
```

- **Puzzles:** pressable puzzle buttons. `3x3x3` is enabled; `4x4x4` and
  `2x2x2` are disabled placeholders.
- **Cube:** the selected puzzle's colour editor.
- **Colours:** a 2×3 sidebar of bordered colour chips.
- **Actions:** solver stage, solve, scramble, clear, mouse, theme, view, help,
  and playback controls.
- **Solution:** the bottom output panel, styled like an editor terminal.

## Solving flow

1. Select `3x3x3` from the left pane.
2. Paint all 54 stickers. Each face's centre colour identifies that face, so
   the six centres must all be different.
3. Choose a stage with `2`, `3`, or `4`.
4. Press `s` to solve.
5. The output panel reports the stage, move count, and elapsed time.
6. Playback updates the cube pane move by move. The final step shows the solved
   cube.

### Playback

- `space` play/pause
- `h` / `l` step backward/forward
- `g` / `G` first/last move
- Click a move in the output panel to jump to it
- Mouse wheel scrolls the output

The editor input is never modified by playback. Press `v` to toggle between the
playback view and the input view, or `e` to return to editing.

### Re-solve with another stage

1. Change the stage with `2`, `3`, or `4`.
2. Press `s` again.

The action label changes to `[s] Re-solve` after a solution exists. The same
entered facelets are reused, playback resets to step 0, and the new solution
replaces the old one.

## Controls

| Scope | Keys | Action |
|---|---|---|
| Panes | `tab` / `shift+tab` | cycle visible panes |
| Puzzles | `↑`/`↓` or `j`/`k`, `enter` | choose and open a puzzle |
| Cell cursor | arrows or `h` `j` `k` `l` | move the focused sticker |
| Face rotation | `ctrl+arrows` or `H` `J` `K` `L` | rotate the cube view |
| Face cycle | `[` / `]` | yaw the cube view |
| Paint | `w` `y` `r` `o` `b` `g` | paint white/yellow/red/orange/blue/green |
| Paint current | `space` | repaint the focused sticker |
| Clear cell | `x` / `backspace` | clear the focused sticker |
| Clear cube | `c` | reset the current puzzle input |
| Stage | `2` `3` `4` | select 2-stage / 3-stage / 4-stage |
| Solve / re-solve | `s` | solve the current input |
| Scramble | `n` | load a random 20-move scramble |
| Playback | `space`, `h`/`l`, `g`/`G` | play/pause, step, first/last |
| View | `v` | toggle playback/input view |
| Edit input | `e` | return to the input cube |
| Actions | `↑`/`↓`, `enter` | select and activate an action |
| Themes | `t` | open the theme picker |
| Cell aspect | `A` | cycle aspect values live |
| Mouse | `m` | toggle mouse capture |
| Help | `?` | help overlay |
| Back | `esc` | cancel a solve or return to puzzles |
| Quit | `q` / `ctrl+c` | quit |

When no puzzle is selected, tab cycles only between Puzzles and Output.

### Face rotation

The cube uses a real view orientation (`front`, `up`, `right`):

- `Up` / `Down` pitch the cube ±90°.
- `Left` / `Right` yaw the cube ±90°.
- Four presses in any direction return to the starting view.
- The displayed grid rotates with the view.
- Painting and playback respect the current orientation.

## Mouse and hover

- Click a puzzle button to select it.
- Click a colour chip to choose the active colour.
- Click a cube cell to paint it; drag to paint multiple cells.
- Click the four face arrows to rotate the view.
- Click action pills to select stages, solve, scramble, clear, toggle mouse,
  open themes, toggle view, or control playback.
- Click a solution move to jump playback to that step.
- Wheel over the output panel scrolls; wheel over the puzzles pane moves the
  cursor.

Hover uses `MouseModeAllMotion` and requires a terminal that reports motion
events. Hovered buttons, chips, arrows, and cells use the theme's darker hover
accent. Use `--no-hover` to fall back to click-only `MouseModeCellMotion`.

## Square cells and auto-calibration

Terminal character cells are not square, so cube cells are sized from an
aspect ratio:

```go
cellH = 4
cellW = round(cellH * cell_aspect)
gapX  = round(cell_aspect)
```

On Linux/macOS the TUI reads the terminal's pixel dimensions with
`TIOCGWINSZ`:

```text
charW  = xpixel / cols
charH  = ypixel / rows
aspect = charH / charW
```

Precedence:

1. `--cell-aspect`
2. auto-detected terminal aspect
3. config `cell_aspect`
4. fallback `1.75`

If the terminal reports zero pixels (some tmux/screen setups), fallback is used.
Press `A` to cycle `1.5, 1.6, 1.7, 1.75, 1.8, 1.9, 2.0` manually.

## Themes

Built-in themes:

- `gruvbox` (default)
- `default`
- `nord`
- `dracula`
- `catppuccin`
- `tokyo-night`
- `onedark`
- `rose-pine`
- `kanagawa`
- `solarized-dark`
- `github-dark`
- `mono`

### Theme precedence

1. `--theme <name>`
2. `RBXCB_THEME`
3. saved `theme` in `config.json`
4. `gruvbox` fallback

### Theme picker and persistence

Press `t` or click `[t] Themes` to open the picker.

- `j`/`k` or arrows move.
- `enter` applies the highlighted theme.
- `esc` closes without changing.
- Mouse hover highlights; click selects.

Selecting a theme writes it to the XDG config and restores it on the next
launch. `--theme`, `RBXCB_THEME`, and `--no-color` are one-session overrides and
do not rewrite the saved theme.

### Monochrome

`mono` / `NO_COLOR` uses distinct foreground patterns per sticker colour:

| Colour | Pattern |
|---|---|
| White | `█` |
| Yellow | `▓` |
| Red | `▒` |
| Orange | `░` |
| Blue | `▚` |
| Green | `▞` |

Mono mode emits no colour SGR and uses a focus marker instead of double pane
borders.

## Configuration

Config path:

```text
$XDG_CONFIG_HOME/rbxcb-tui/config.json
```

Fallback:

```text
~/.config/rbxcb-tui/config.json
```

A missing config is not an error. The file is created when you select a theme
from the picker.

### Schema

| Field | Type | Description |
|---|---|---|
| `theme` | string | Saved theme name |
| `hover` | bool | Enable hover highlighting |
| `mouse` | bool | Start with mouse capture enabled |
| `cell_aspect` | number | Manual cell aspect (height/width multiplier) |
| `auto_aspect` | bool | Enable pixel-based auto-detection |
| `themes` | object | Custom theme definitions |

### Example

```json
{
  "theme": "nord",
  "hover": true,
  "mouse": true,
  "cell_aspect": 1.75,
  "auto_aspect": true,
  "themes": {
    "mine": {
      "accent": "#66d9ef",
      "accent_hover": "#3aa8bd",
      "border": "#49483e",
      "muted": "#75715e",
      "text": "#f8f8f2",
      "error": "#f92672",
      "success": "#a6e22e",
      "stickers": [
        "#f8f8f2",
        "#f4d03f",
        "#e74c3c",
        "#e67e22",
        "#3498db",
        "#2ecc71"
      ],
      "button_bg": "#1f2937",
      "button_bg_hover": "#111827",
      "button_fg": "#e5e7eb",
      "button_fg_hover": "#f9fafb"
    }
  }
}
```

Custom themes appear in the theme picker and `--list-themes`. Missing custom
theme fields inherit from gruvbox. `accent_hover` is derived by darkening the
accent when omitted.

Config writes are atomic: a temporary file is written and renamed into place,
preserving `hover`, `mouse`, aspect settings, and custom themes when the theme
changes.

## Facelet order

Input is serialized as 54 characters in the same order as the Java backend:

```text
U L F R B D
```

Each face is row-major (3 rows of 3). The six centre cells define which colour
belongs to which face, so all six centres must be different.

| Face | Colour | Letter |
|---|---|---|
| U | White | `w` |
| D | Yellow | `y` |
| F | Green | `g` |
| B | Blue | `b` |
| R | Red | `r` |
| L | Orange | `o` |

The entered colour grid is preserved across solves, stage changes, playback, and
re-solves.

## Architecture

```text
cmd/rbxcb-tui          CLI wiring
internal/cube          cubie and facelet model, notation, masks, validation
internal/solver        bidirectional BFS, staged solvers, CFOP
internal/tui           Bubble Tea workspace, editor, actions, output, themes
internal/tui/layout.go pane geometry shared by rendering and mouse hit testing
internal/tui/orientation.go  3D view-orientation tables
e2e                    randomized solver end-to-end tests
testdata               Java-generated parity fixtures
```

Key design points:

- Cubie state uses fixed arrays for fast BFS expansion.
- Masks use comparable fixed-size keys.
- Bidirectional stage search supports cancellation and limits.
- Facelet parsing, move semantics, and rendering are verified against Java
  fixtures.
- Rendering, hitboxes, and mouse mapping all share one layout calculation.
- The cube view uses `(front, up)` orientation with precomputed cell mappings.

## Testing

```bash
just test              # go vet + unit + quick E2E
just test-e2e          # verbose randomized solver E2E
just test-e2e-long     # 100 random scrambles per stage
```

Equivalent direct commands:

```bash
go vet ./...
go test ./...
go test -race ./internal/...
E2E_SCRAMBLES=100 go test ./e2e -run E2E -v -timeout 30m
```

The randomized E2E suite:

1. Generates legal random scrambles.
2. Serializes them to facelets.
3. Parses them through the same path as the TUI.
4. Solves with 2-stage, 3-stage, and 4-stage solvers.
5. Applies the solution and asserts the cube is fully solved.
6. Fails with the seed and scramble printed for reproduction.

CI runs unit tests, race tests, and a randomized E2E batch.

## Java parity fixtures

The Go engine is validated against the sibling Java implementation.

Regenerate fixtures:

```bash
cd ../rbxcb
devenv shell -- fixtures
```

This writes:

```text
../rbxcb-tui/testdata/java_fixtures.json
```

The fixture set includes solved/state facelets, all 18 moves, and deterministic
scrambles.

## Development

```bash
direnv allow           # or: devenv shell
just run
just demo
just test
just lint
just fmt
just build
just nix-build
```

The devenv environment provides:

- Go 1.27 via Nix
- `golangci-lint`
- `gopls`
- `gotools`
- `just`

`nix build` builds the app as a flake package; `nix run .` runs it directly.

## Troubleshooting

### “terminal too small”

The workspace requires at least `100×37`. Enlarge the terminal window.

### Cube cells still look rectangular

Press `A` to cycle aspect values. If one looks square, keep it for the session,
or set it permanently:

```bash
rbxcb-tui --cell-aspect 1.78
```

Auto-detection can fail in tmux/screen or terminals that report zero pixel
dimensions. Use `--no-auto-aspect` with a manual value in that case.

### Theme does not persist

Only selections from the theme picker are saved. `--theme` and `RBXCB_THEME` are
one-session overrides. Check that the config directory is writable; the status
bar reports `(not saved: …)` on failure.

### Colours look wrong or backgrounds bleed

Use a truecolor-capable terminal for the full palette. The cube and palette do
not use background colours, so sticker rendering is stable across terminals.
`--no-color` / `NO_COLOR` switches to the monochrome pattern theme.

### Hover does not work

Hover requires a terminal that supports all-motion mouse reporting. Clicks still
work without it. Use `--no-hover` to disable hover and reduce mouse event
traffic.

## Limitations

- Only **3x3x3** is supported.
- `4x4x4` and `2x2x2` entries are visible but disabled.
- The Java 4x4x4 implementation in the reference repository is incomplete.
- Random 2-stage solves can take noticeably longer than 3-stage or 4-stage.
- The TUI is keyboard/mouse oriented; there is no image upload or camera input.
