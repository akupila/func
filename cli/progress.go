package cli

import (
	"strings"
	"sync"

	"github.com/func/func/ui"
)

type progressBar struct {
	Width  int
	Frames string

	mu       sync.Mutex
	progress float64
	buf      strings.Builder
}

func newProgressBar(width int) *progressBar {
	return &progressBar{
		Width:  width,
		Frames: "●○",
	}
}

func (p *progressBar) SetProgress(progress float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress = progress
}

func (p *progressBar) Render(frame ui.Frame) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	width := p.Width
	if width > frame.Width {
		width = frame.Width
	}

	frames := []rune(p.Frames)
	filled := frames[0]
	unfilled := frames[len(frames)-1]
	var mid []rune
	if len(frames) > 2 {
		mid = frames[1 : len(frames)-1]
	}

	p.buf.Reset()
	for i := 0; i < width; i++ {
		offset := float64(i) / float64(width)
		v := (p.progress - offset) * float64(width)
		if v > 0.99 {
			v = 1
		}

		if v >= 1 {
			p.buf.WriteRune(filled)
			continue
		}
		if v > 0 && len(mid) > 0 {
			index := v*float64((len(mid)+1)) - 1
			if index >= 0 {
				p.buf.WriteRune(mid[int(index)])
				continue
			}
		}
		p.buf.WriteString(ui.Format(string(unfilled), ui.Dim))
	}
	return p.buf.String()
}
