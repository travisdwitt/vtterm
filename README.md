
```
        __   __
.--.--.|  |_|  |_.-----.----.--------.
|  |  ||   _|   _|  -__|   _|        |
 \___/ |____|____|_____|__| |__|__|__|
```
A virtual tabletop in your terminal.

## Features

- **Square and hex grids**
- **Basic Drawing tools**
- **Reusable tokens**
- **Layer system**
- **Pan and zoom**
- **Save/load**
- **16-color palette**
- **Per-token properties**

## Requirements

- Go 1.25+
- A terminal emulator with 256-color support (most modern terminals)

## Building

```bash
git clone https://github.com/traviswitt/vtterm.git
cd vtterm
go build -o vtterm .
```

## Running

```bash
./vtterm
```

Or run directly without building:

```bash
go run .
```

## Creating a Table

Select **New Table** and follow the wizard:

1. **Grid type** — choose between:
   - **Grid** — square cells
   - **Hexes** — flat-top hexagonal cells
   - **None** — blank canvas (no grid)
2. **Height** — number of rows (1-200)
3. **Width** — number of columns (1-200)

### Grid Examples

**Square grid (3x3):**
```
+---+---+---+
|   |   |   |
+---+---+---+
|   |   |   |
+---+---+---+
|   |   |   |
+---+---+---+
```

**Hex grid (3x2):**
```
    +---+       +---+
   /     \     /     \
  +       +---+       +
   \     /     \     /
    +---+       +---+
   /     \     /     \
  +       +---+       +
   \     /     \     /
    +---+       +---+
```

## Table View Keybindings

The table view is where you interact with your map. 
The top bar shows your current position, layer, mode, and available commands.

### Navigation

| Key | Action |
|-----|--------|
| `h` / `l` / `j` / `k` | Move cursor left/right/down/up |
| `H` / `L` / `J` / `K` | Move cursor by 2 in each direction |
| Arrow keys | Same as hjkl |
| `z` | Toggle pan mode (hjkl moves the viewport instead of cursor) |
| `tab` | Jump to next token |
| `shift+tab` | Jump to previous token |
| `esc` | Exit pan mode / cancel current action |

### Mode System

vtterm uses a mode-based interface. Press `m` to enter the mode menu, then press a second key to select an action:

| Sequence | Action |
|----------|--------|
| `m` `m` | **Move** — pick up the shape or token under the cursor |
| `m` `d` | **Draw** — open the draw sub-menu |
| `m` `t` | **Text** — start typing free-form text at cursor position |
| `m` `l` | **Layer** — change the active layer |
| `esc` | Cancel and return to normal mode |

### Drawing

From the draw menu (`m` then `d`):

| Key | Action |
|-----|--------|
| `l` | **Line** — draw a line segment using hjkl to extend |
| `b` | **Box** — draw a rectangle, hjkl to resize |
| `ctrl+s` | Commit the drawing to the overlay |
| `esc` | Cancel and discard |

### Text Mode

Enter with `m` then `t`:

| Key | Action |
|-----|--------|
| Type | Characters appear at cursor position |
| `enter` | New line |
| Arrow keys | Move text cursor |
| `backspace` / `delete` | Delete characters |
| `ctrl+s` | Commit text to the overlay |
| `esc` | Cancel and discard |

### Moving Shapes and Tokens

Enter with `m` then `m` while the cursor is over a shape or token:

| Key | Action |
|-----|--------|
| `h` / `l` / `j` / `k` | Move the shape/token |
| `m` | Open mode menu (for layer changes during move) |
| `enter` | Place at current position |
| `esc` | Cancel and restore original position |

### Layers

Enter layer mode with `m` then `l`:

| Key | Action |
|-----|--------|
| `+` / `=` | Move to higher layer |
| `-` | Move to lower layer |
| `enter` | Confirm (places shape if moving) |
| `esc` / `l` | Return to previous mode |

The current layer number is shown in the top bar as `L:0`, `L:1`, etc. Overlay characters and tokens are drawn only on their assigned layer.

### Tokens

| Key | Action |
|-----|--------|
| `T` | Open the token menu |
| `i` | Inspect token under cursor (show properties) |
| `e` | Edit properties of token under cursor |
| `r` | Rotate token facing (4 directions on square grid, 6 on hex) |
| `d` | Toggle disabled state on token under cursor |
| `D` | Delete token placement from the table |
| `c` | Open color picker for the token or shape under cursor |

### Color Picker

Press `c` on a token or drawn shape to open the color palette:

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate colors |
| `enter` | Apply selected color |
| `esc` | Cancel |

Available colors: Orange, Red, Green, Blue, Yellow, Magenta, Cyan, White, Brown, Dark Red, Dark Green, Dark Blue, Purple, Pink, Light Blue, Gray.

Token colors are **per-placement** — the same token definition can have different colors at different positions on the table.

### Saving and Loading

| Key | Action |
|-----|--------|
| `s` | Save the table (prompts for a name on first save) |
| `S` | Export the grid as a plain text `.txt` file |
| `q` | Quit |
| `ctrl+c` | Force quit |

Tables are saved as JSON files in `~/.vtterm/`. The load screen shows all saved tables and lets you select one with `enter` or delete with `d`.

## Token Library

Access from the main menu (**Tokens**) or from the table view (`T`).

Tokens are reusable objects defined in a global library (`~/.vtterm/tokens.json`). Each token has:
- An **ID** (auto-generated UUID)
- A **folder** (optional, for organization)
- **Properties** — key-value pairs (the first property's value is used as the display label)

### Token Menu Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate the token list |
| `enter` | Place selected token on the table (from table view) or expand/collapse folder |
| `n` | Create a new token |
| `e` | Edit the selected token's properties |
| `d` | Delete selected token or folder (with confirmation) |
| `f` | Create a new folder |
| `esc` | Close the menu |

### Token Properties Editor

When editing a token (`e`), each line is a `Key: Value` pair:

```
Name: Goblin
HP: 15
AC: 12
```

| Key | Action |
|-----|--------|
| Type | Edit property text |
| `enter` | New line (new property) |
| Arrow keys | Move cursor |
| `backspace` / `delete` | Delete characters |
| `ctrl+s` | Save changes |
| `esc` | Discard changes |

### Token Rendering

Tokens appear on the grid as 5x3 ASCII boxes showing the first 3 characters of their display label:

```
+---+
|Gob|
+---+
```

Disabled tokens appear in gray. Colored tokens appear in their assigned color. The default color is orange.

### Folders

Folders organize tokens in the library. They are **collapsed by default** and can be expanded/collapsed with `enter`. Tokens inside a folder are indented:

```
> [Monsters]          <- collapsed folder
v [Players]           <- expanded folder
    Aragorn
    Gandalf
```

## Data Storage

All data is stored in `~/.vtterm/`:

| File | Contents |
|------|----------|
| `*.json` | Saved tables (grid, overlay, token placements) |
| `tokens.json` | Global token library (definitions and folders) |
| `*.txt` | Exported plain-text grid renders |

