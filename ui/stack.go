package ui

import (
	"bytes"
	"sync"
)

// A Stack maintains a list of child nodes.
//
// All methods are safe for concurrent access.
type Stack struct {
	mu    sync.Mutex
	nodes []Renderer
	buf   bytes.Buffer
}

// Render renders all children in the stack separated by new lines.
//
// Calling Render() on a nil stack returns an empty string.
func (s *Stack) Render(f Frame) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
	for i, c := range s.nodes {
		line := c.Render(f)
		s.buf.WriteString(line)
		if i < len(s.nodes)-1 {
			s.buf.WriteByte('\n')
		}
	}
	return s.buf.String()
}

// Len returns the number of children in the stack.
func (s *Stack) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.nodes)
}

// Push adds a new node to the end of the stack.
func (s *Stack) Push(node Renderer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = append(s.nodes, node)
}

// Insert inserts a node at the given index.
// Panics if the given index does not exist in the stack.
func (s *Stack) Insert(node Renderer, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= len(s.nodes) {
		panic("Out of bounds")
	}
	s.nodes = append(s.nodes, nil)
	copy(s.nodes[index+1:], s.nodes[index:])
	s.nodes[index] = node
}

// Remove removes a node from the stack.
// No-op if the child does not exist.
func (s *Stack) Remove(node Renderer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.nodes {
		if n == node {
			s.nodes = append(s.nodes[:i], s.nodes[i+1:]...)
			return
		}
	}
}
