package termsize

import (
	"github.com/containerd/console"
)

// DefaultSize return current terminal size with default width and height, the default values will be used if any error occurs.
func DefaultSize(w, h int) (width int, height int) {
	width, height = w, h
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	current := console.Current()
	if ws, err := current.Size(); err == nil {
		width, height = int(ws.Width), int(ws.Height)
	}
	return
}
