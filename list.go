package main

type Node struct {
    data *Line
    next *Node
    prev *Node
}

type List struct {
    size int
    head *Node
    tail *Node
}

func NewList() *List {
    return &List{0, &Node{nil, nil, nil},
                    &Node{nil, nil, nil},
    }
}

func (L *List) Append(line *Line) {
    if L.size == 0 {
        node := &Node{line, nil, nil}
        L.head.next = node
        L.head.next.prev = L.head
        L.tail.next = node
        L.size += 1
    } else {
        current := L.head
        for current.next != nil {
            current = current.next
        }
        node := &Node{line, nil, current}
        current.next = node
        L.tail.next = current.next
        L.size += 1
    }
}

func (L *List) Prepend(line *Line) {
    if L.head == nil && L.tail == nil {
        node := &Node{line, nil, nil}
        L.head.next = node
        L.head.next.prev = L.head
        L.tail.next = node
        L.size += 1
    } else {
        node := &Node{line, L.head.next, L.head}
        L.head.next = node
        L.head.next.next.prev = node
        L.size += 1
    }
}

func (L *List) DoPrint() {
    current := L.head.next
    for current.next != nil {
        println(current.data)
        current = current.next
    }
    println(current.data)
}

func (L *List) PopFromHead() *Node {
    first := L.head.next
    L.head.next = L.head.next.next
    L.head.next.prev = L.head
    first.next = nil
    first.prev = nil
    L.size -= 1
    return first
}

func (L *List) PopFromTail() *Node {
    last := L.tail.next
    L.tail.next = L.tail.next.prev
    L.tail.next.next = nil
    last.prev = nil
    L.size -= 1
    return last
}

func (L *List) Size() int {
    return L.size
}
