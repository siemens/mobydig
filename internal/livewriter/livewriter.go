// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package livewriter

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// eraseLineAbove moves the cursor one line up and then erases that line.
const eraseLineAbove = "\x1b[1A\x1b[2K"

// ansiEscape matches CSI escape sequences, such as color and style changes,
// that do not occupy any terminal columns.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]")

// Writer buffers output and on each Flush replaces the output of the previous
// Flush with the newly buffered output. Flushed output is expected to end with
// a newline.
type Writer struct {
	out   io.Writer
	width func() int // terminal width in columns, or 0 if unknown.
	buf   bytes.Buffer
	lines int // terminal lines occupied by the previously flushed output.
}

// New returns a new Writer writing to out.
func New(out io.Writer) *Writer {
	return &Writer{
		out:   out,
		width: func() int { return termWidth(out) },
	}
}

// Write appends p to the output shown by the next Flush.
func (w *Writer) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// Flush replaces the previously flushed output with the buffered output.
func (w *Writer) Flush() error {
	var out bytes.Buffer
	for range w.lines {
		out.WriteString(eraseLineAbove)
	}
	w.lines = countLines(w.buf.Bytes(), w.width())
	out.Write(w.buf.Bytes())
	w.buf.Reset()
	_, err := w.out.Write(out.Bytes())
	return err
}

// countLines returns the number of terminal lines the newline-terminated lines
// in text occupy, taking line wrapping at the specified width into account,
// unless width is 0.
func countLines(text []byte, width int) int {
	lines := 0
	for {
		idx := bytes.IndexByte(text, '\n')
		if idx < 0 {
			return lines
		}
		lines += rows(text[:idx], width)
		text = text[idx+1:]
	}
}

// rows returns the number of terminal lines the specified line occupies.
func rows(line []byte, width int) int {
	if width <= 0 {
		return 1
	}
	cols := utf8.RuneCount(ansiEscape.ReplaceAll(line, nil))
	return max(1, (cols+width-1)/width)
}

// termWidth returns the width in columns of the terminal out writes to, or 0
// if out isn't a terminal.
func termWidth(out io.Writer) int {
	f, ok := out.(*os.File)
	if !ok {
		return 0
	}
	conn, err := f.SyscallConn()
	if err != nil {
		return 0
	}
	var ws *unix.Winsize
	if ctrlerr := conn.Control(func(fd uintptr) {
		ws, err = unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	}); ctrlerr != nil || err != nil {
		return 0
	}
	return int(ws.Col)
}
