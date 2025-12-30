package trie

import (
	"sort"
	"strings"
	"sync"
)

type Trie struct {
	mu   sync.RWMutex
	root *Node
}

type Node struct {
	children map[rune]*Node
	isEnd    bool
	value    string
}

func NewTrie() *Trie {
	return &Trie{
		root: &Node{children: make(map[rune]*Node)},
	}
}

func (t *Trie) Insert(word string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.root
	lowered := strings.ToLower(word)
	for _, char := range lowered {
		if _, exists := current.children[char]; !exists {
			current.children[char] = &Node{children: make(map[rune]*Node)}
		}
		current = current.children[char]
	}
	current.isEnd = true
	current.value = word
}

func (t *Trie) Search(word string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	current := t.root
	lowered := strings.ToLower(word)
	for _, char := range lowered {
		next, exists := current.children[char]
		if !exists {
			return false
		}

		current = next
	}
	return current.isEnd
}

func (t *Trie) Autocomplete(prefix string, limit int) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	current := t.root
	lowered := strings.ToLower(prefix)
	for _, char := range lowered {
		next, exists := current.children[char]
		if !exists {
			return []string{}
		}

		current = next
	}

	var results = make([]string, 0, limit)
	t.dfs(current, &results, limit)
	return results
}

func (t *Trie) dfs(startNode *Node, results *[]string, limit int) {
	stack := []*Node{startNode}

	for len(stack) > 0 {
		if len(*results) >= limit {
			return
		}
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if curr.isEnd {
			*results = append(*results, curr.value)
		}

		keys := make([]rune, 0, len(curr.children))
		for child := range curr.children {
			keys = append(keys, child)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i] > keys[j]
		})

		for _, key := range keys {
			stack = append(stack, curr.children[key])
		}
	}
}
