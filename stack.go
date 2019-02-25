package main

import "fmt"

// FILO stack

type Stack struct {
	data []int
	top int
	size int
}

func NewStack(size int) *Stack {
	return &Stack {
		make([]int, size), 0, size,
	}
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

func (s *Stack) Push(num int) {
	if s.top < s.size {
		s.data[s.top] = num
		s.top += 1
	} else {
		fmt.Println("Error :: We need to allocate more Stack space")
	}
}

func (s *Stack) Pop() int {
	s.top -= 1
	if s.top < 0 {
		return -1
	}
	deleted := s.data[s.top]
	s.data[s.top] = 0
	return deleted
}
