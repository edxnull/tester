package main

import "fmt"

// FIFO queue

type Queue struct {
	data []*Line
	front int
    rear  int
	size  int
}

func NewQueue(size int) *Queue {
	return &Queue {
        make([]*Line, size), size-1, size, 0,
    }
}

func (q *Queue) Len() int {
    return len(q.data)
}

func (q *Queue) Cap() int {
    return cap(q.data)
}

func (q *Queue) IsEmpty() bool {
    return q.size == 0
}

func (q *Queue) Enqueue(line *Line) {
    if (q.rear-1) >= 0 {
        q.data[q.rear-1] = line
        q.rear -= 1
        q.size += 1
    } else {
        fmt.Println("ERROR :: We need to allocate more Queue space")
    }
}

func (q *Queue) Dequeue() *Line {
    if q.rear <= q.front {
        deleted := q.data[q.front]
        copy(q.data[q.Len()-q.size+1:], q.data[q.Len()-q.size:q.Len()-1])
        q.data[q.Len()-q.size] = nil
        q.size -= 1
        q.rear += 1
        return deleted
    }
    return nil
}
