## DRAFT ##
Unix:    creack/pty
Windows: go-winio (ConPTY)
   ->
vt10x (or Charm ANSI parser)

github.com/creack/pty runs a virtual terminal

High-level data flow (lock this in)
keyboard -> PTY -> shell
shell    -> PTY -> vt10x -> renderer -> real terminal


creack/pty = process + pseudo-terminal

[vt10x](https://github.com/hinshun/vt10x) maintains an in-memory screen state: rows/cols, cursor position, per-cell glyphs (char + fg/bg + mode bits), title, etc.

General ideas: read from PTY, feed vt10x, render

## Terminal escape sequences ##

See the list of sequences here: https://gist.github.com/ConnerWill/d4b6c776b509add763e17f9f113fd25b

All start from ESC (escape, `\x1b`). When issue escape, the terminal switches to its own "mini-language". For instance:
```
\x1b[31m = whole comman
\x1b     = ESC
[        = CSI (Control Sequence Introducer)
31       = parameter
m        = command
```
This is xterm reference https://invisible-island.net/xterm/ctlseqs/ctlseqs.html with escape sequences examples. But you should read it like that:
```
written: OSC 0 ; <title> BEL
actual:  \x1b]0;<title>\x07

written: CSI Pm m
actual:  \x1b[31;1m   (red + bold)
         \x1b[0m      (reset)

         \x1b[?25l (hide cursor)
         \x1b[?25h (show cursor)

written: CSI Ps ; Ps H
actual:  \x1b[10;20H
```

Here is the additional info:
|Symbol|Actual bytes|
|------|------------|
| ESC  | \x1b       |
| CSI  | \x1b[      |
| OS   | \x1b]      |
| BEL  | \x07       |
| ST   | \x1b\      |

## Logical model ##

Window contains layout (LayoutCompositor)
Pane prints border set underlying terminal rect
Terminal mantains pty connection and renders terminal

```
Manager
   ├── Window
   │   └── Pane
   │       └── Terminal
   └── Window
       ├── Pane
       │   └── Terminal
       └── Pane
           └── Terminal
```