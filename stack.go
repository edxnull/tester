package main

// FILO stack

type Stack struct {
	data []*Line
	top  int
	size int
}

func NewStack(size int) *Stack {
	return &Stack{make([]*Line, size), 0, size}
}

func (s *Stack) Len() int {
	return len(s.data)
}

func (s *Stack) Cap() int {
	return cap(s.data)
}

func (s *Stack) IsEmpty() bool {
	return s.top == 0
}

func (s *Stack) Push(line *Line) {
	if s.top < s.size {
		s.data[s.top] = line
		s.top += 1
	} else {
		panic("Error :: We need to allocate more Stack space")
	}
}

func (s *Stack) Pop() *Line {
	s.top -= 1
	if s.top < 0 {
		return nil
	}
	deleted := s.data[s.top]
	s.data[s.top] = nil
	return deleted
}

func (s *Stack) GetLast() *Line {
	if s.IsEmpty() {
		return nil
	}
	return s.data[s.top-1]
}
