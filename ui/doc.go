// Package ui provides declarative console based user interface output.
//
// Operation
//
// The target renderer is expected to render the entire desired UI every frame.
// Every consecutive render diffs this string and only the necessary updates
// are flushed to the output.
//
// Terminal size
//
// Every render is passed a current frame, which includes the terminal size. In
// case the terminal is resized, a new render is triggered with the new size,
// based on the output of the previous render.
package ui
