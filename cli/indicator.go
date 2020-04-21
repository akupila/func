package cli

import (
	"time"

	"github.com/func/func/ui"
)

var lineIndicatorFrames = []string{
	"╴ ",
	"─ ",
	"╶╴",
	" ─",
	" ╶",
	" ╶",
	" ─",
	"╶╴",
	"─ ",
	"╴ ",
	"╴ ",
}

func lineIndicator(_ ui.Frame) string {
	frameDur := time.Second / 25
	num := int(time.Now().UnixNano()/frameDur.Nanoseconds()) % len(lineIndicatorFrames)
	return lineIndicatorFrames[num]
}
