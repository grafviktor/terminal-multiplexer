Unix:    creack/pty
Windows: go-winio (ConPTY)
   ↓
vt10x (or Charm ANSI parser)

github.com/creack/pty runs a virtual terminal

High-level data flow (lock this in)
keyboard → PTY → shell
shell    → PTY → vt10x → renderer → real terminal


creack/pty = process + pseudo-terminal

[vt10x](https://github.com/hinshun/vt10x) = understands ANSI + screen state

you = read from PTY, feed vt10x, render