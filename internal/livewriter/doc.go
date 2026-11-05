/*
Package livewriter implements a minimal “live” terminal writer that replaces
its previously flushed output with the next flushed output, such as for
continuously updating a status display.

mobydig previously used github.com/gosuri/uilive for this, but uilive v0.0.4
queries the terminal size using the TIOCGWINSZ ioctl into a 4 byte variable,
while the kernel writes an 8 byte struct winsize. Each flush thus silently
overwrites the 4 bytes of whatever variable the linker places next; for
instance, coverage package IDs, corrupting coverage data. Additionally, each
flush leaks a /dev/tty file descriptor until garbage collected. As uilive
isn't maintained anymore, livewriter implements just what mobydig needs.
*/
package livewriter
