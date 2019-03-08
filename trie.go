package main

const R = 26

type TrieNode struct {
	key         string
	end_of_word bool
	paths       int32
	children    [R]*TrieNode
}

func NewTrie() *TrieNode {
	return &TrieNode{}
}

func MakeNull(trie *TrieNode) {
	trie.key = ""
	trie.end_of_word = false
	trie.paths = 0
	for i := 0; i < len(trie.children); i++ {
		trie.children[i] = nil
	}
}

func (trie *TrieNode) KeysWithPrefix(prefix string) {
	var curr_index int8
	var current *TrieNode

	current = trie
	for i := 0; i < len(prefix); i++ {
		curr_index = __get_ascii_index(prefix[i])
		current = current.children[curr_index]
	}

	for _, node := range GetChildren(current) {
		Traverse(prefix, node)
	}
}

func GetChildren(trie *TrieNode) []*TrieNode {
	var children []*TrieNode
	for index := range trie.children {
		if trie.children[index] != nil {
			children = append(children, trie.children[index])
		}
	}
	return children
}

func GetSibling(trie *TrieNode, index int) *TrieNode {
	if index > len(trie.children)-1 {
		return nil
	}
	current := trie
	for i := index + 1; i < len(current.children); i++ {
		if current.children[i] != nil {
			return current.children[i]
		}
	}
	return nil
}

func Traverse(prefix string, trie *TrieNode) {
	var path []*TrieNode
	var current *TrieNode
	var prev *TrieNode
	var children []*TrieNode
	var nloops int

	current = trie

	path = append(path, current)

	if current.paths > 1 {
		children = GetChildren(current)
		nloops = len(children)
	} else {
		nloops = 1
	}

	goback := 0
	revert := len(prefix)
	for i := 0; i < nloops; i++ {
		if len(children) > 1 {
			current = children[i]
			path = append(path, current)
			revert += 1
		}
		for {
			if current.end_of_word && current.paths > 0 {
				print(prefix)
				for i := range path {
					print(path[i].key)
				}
				println("")
			}

			for index := range current.children {
				if current.children[index] != nil {
					if current.paths > 1 {
						prev = GetSibling(current, index)
						goback = revert
					}
					current = current.children[index]
					path = append(path, current)
					revert += 1
					break
				}
			}

			if current.paths == 0 {
				print(prefix)
				for i := range path {
					print(path[i].key)
				}
				println("")
				if prev != nil {
					LEN := len(path)
					for rm := goback; rm < LEN; rm++ {
						path = append(path[0:0], path[:len(path)-1]...)
					}
					current = prev
					path = append(path, current)
					prev = nil
					revert = goback + 1
				} else {
					break
				}
			}
		}
		for rm := len(path); rm > len(prefix); rm-- {
			path = append(path[0:0], path[:len(path)-1]...)
		}
	}
}

func (trie *TrieNode) Search(word string) bool {
	var curr_index int8
	var current *TrieNode

	curr_index = __get_ascii_index(word[0])
	current = trie.children[curr_index]

	if current == nil {
		return false
	}
	for i := 1; i < len(word); i++ {
		curr_index = __get_ascii_index(word[i])
		current = current.children[curr_index]
		if current == nil {
			return false
		}
	}
	if current.end_of_word == true {
		return true
	}
	return false
}

func (trie *TrieNode) Delete(word string) {
	var current *TrieNode
	var nodes []*TrieNode

	current = trie.children[__get_ascii_index(word[0])]
	nodes = append(nodes, current)
	for i := 1; i < len(word); i++ {
		current = current.children[__get_ascii_index(word[i])]
		nodes = append(nodes, current)
		if current.paths >= 1 && current.end_of_word {
			current.end_of_word = false
		}
		if current.paths == 0 && current.end_of_word {
			for i := 0; i < len(nodes)-1; i++ {
				MakeNull(nodes[i])
			}
		}
	}

	// stack = append(stack, ...)
	// if we find the word
	// pop from the stack and set nodes to nil
}

func (trie *TrieNode) Insert(word string) {
	curr_char := 0
	curr_index := __get_ascii_index(word[0])
	add_new_nodes := true
	var current *TrieNode

	if trie.children[curr_index] == nil {
		trie.children[curr_index] = new(TrieNode)
		trie.paths += 1
		current = trie.children[curr_index]
		current.key = string(word[0])
		curr_char += 1
	} else {
		current = trie.children[curr_index]
		for current.key == string(word[curr_char]) || (current.end_of_word != true) {
			if curr_char == len(word)-1 {
				if current.end_of_word == true {
					add_new_nodes = false
					break
				}
				current.end_of_word = true
				add_new_nodes = false
				break
			}
			curr_char += 1
			curr_index = __get_ascii_index(word[curr_char])
			if current.children[curr_index] == nil {
				break
			}
			current = current.children[curr_index]
		}
	}
	if add_new_nodes {
		for i := curr_char; i < len(word); i++ {
			curr_index = __get_ascii_index(word[i])
			current.children[curr_index] = new(TrieNode)
			current.paths += 1
			current = current.children[curr_index]
			current.key = string(word[i])
		}
		current.end_of_word = true
	}
}

func __get_ascii_index(char byte) int8 {
	if ((char >= 'A') && (char <= 'z')) == false {
		return -1
	}
	return map[byte]int8{
		'a': 0, 'b': 1, 'c': 2, 'd': 3,
		'e': 4, 'f': 5, 'g': 6, 'h': 7,
		'i': 8, 'j': 9, 'k': 10, 'l': 11,
		'm': 12, 'n': 13, 'o': 14, 'p': 15,
		'q': 16, 'r': 17, 's': 18, 't': 19,
		'u': 20, 'v': 21, 'w': 22, 'x': 23,
		'y': 24, 'z': 25, 'A': 26, 'B': 27,
		'C': 28, 'D': 29, 'E': 30, 'F': 31,
		'G': 32, 'H': 33, 'I': 34, 'J': 35,
		'K': 36, 'L': 37, 'M': 38, 'N': 39,
		'O': 40, 'P': 41, 'Q': 42, 'R': 43,
		'S': 44, 'T': 45, 'U': 46, 'V': 47,
		'W': 48, 'X': 49, 'Y': 50, 'Z': 51,
	}[char]
}
