Unix:    creack/pty
Windows: go-winio (ConPTY)
   ↓
vt10x (or Charm ANSI parser)

github.com/creack/pty runs a virtual terminal

High-level data flow (lock this in)
keyboard → PTY → shell
shell    → PTY → vt10x → renderer → real terminal


creack/pty = process + pseudo-terminal

[vt10x](https://github.com/hinshun/vt10x) maintains an in-memory screen state: rows/cols, cursor position, per-cell glyphs (char + fg/bg + mode bits), title, etc.

General ides: read from PTY, feed vt10x, render