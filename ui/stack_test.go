package ui

import (
	"fmt"
	"sync"
	"testing"
)

func TestStack_nil(t *testing.T) {
	var s *Stack
	got := s.Render(Frame{}) // Does not panic
	if got != "" {
		t.Errorf("(*Stack)(nil).Render() returned %q, want \"\"", got)
	}
}

func TestStack_Render(t *testing.T) {
	s := &Stack{}

	s.Push(renderFunc(func(f Frame) string {
		return fmt.Sprintf("<%d>", f.Number)
	}))

	got := s.Render(Frame{Number: 123})
	want := "<123>"
	if got != want {
		t.Errorf("Rendered output does not match; got %q, want %q", got, want)
	}
}

func TestStack_io(t *testing.T) {
	s := &Stack{}

	head := testNode("HEAD")
	s.Push(head)
	tail := testNode("TAIL")
	s.Push(tail)
	for i := 0; i < 3; i++ {
		s.Insert(testNode('A'+i), 1)
	}
	s.Remove(head)

	got := s.Render(Frame{})
	want := "C\nB\nA\nTAIL"
	if got != want {
		t.Errorf("Rendered output does not match; got %q, want %q", got, want)
	}
}

func TestStack_concurrent(t *testing.T) {
	s := &Stack{}

	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Push(testNode(""))
		}()
	}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Insert(testNode(""), 0)
		}()
	}
	wg.Wait()

	if s.Len() != 20000 {
		t.Errorf("Len() does not match; got %d, want %d", s.Len(), 20000)
	}
}

func TestStack_Insert_OutOfBounds(t *testing.T) {
	defer func() {
		p := recover()
		if p == nil {
			t.Error("Did not panic")
		}
	}()

	s := &Stack{}
	s.Insert(testNode(""), 123)
}

type testNode string

func (n testNode) Render(f Frame) string { return string(n) }

type renderFunc func(f Frame) string

func (fn renderFunc) Render(f Frame) string {
	return fn(f)
}
