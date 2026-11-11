package trees

import "fmt"

// Heap is a binary min-heap in an array: the children of index i are at
// 2i+1 and 2i+2, and each parent is smaller than its children.
type Heap struct {
	a            []int
	pos          map[int]int // the index of each key, for Insert and Delete
	restructures int
	comparisons  int
	rec          *Recorder
}

func NewHeap() *Heap { return &Heap{pos: map[int]int{}} }

func (h *Heap) Record(on bool) *Recorder {
	if !on {
		h.rec = nil
		return nil
	}
	h.rec = &Recorder{snap: h.Snapshot}
	return h.rec
}

func (h *Heap) Snapshot() *Node {
	var snap func(i int) *Node
	snap = func(i int) *Node {
		if i >= len(h.a) {
			return nil
		}
		return &Node{Keys: []int{h.a[i]}, Children: []*Node{snap(2*i + 1), snap(2*i + 2)}, Info: fmt.Sprintf("[%d]", i)}
	}
	return snap(0)
}

func (h *Heap) Stats() Stats {
	height := 0
	for n := len(h.a); n > 0; n /= 2 {
		height++
	}
	return Stats{Size: len(h.a), Height: height, Restructures: h.restructures, Comparisons: h.comparisons}
}

func (h *Heap) Keys() []int { return append([]int(nil), h.a...) }

// Min returns the smallest key, at the root.
func (h *Heap) Min() (int, bool) {
	if len(h.a) == 0 {
		return 0, false
	}
	return h.a[0], true
}

// index scans the array: a heap is not ordered for search. (Insert and
// Delete use the map of positions instead, so that they stay O(log n).)
func (h *Heap) index(key int) int {
	h.comparisons = 0
	for i, v := range h.a {
		h.comparisons++
		if v == key {
			return i
		}
	}
	return -1
}

func (h *Heap) Search(key int) bool {
	i := h.index(key)
	if i < 0 {
		h.rec.step("missing", fmt.Sprintf("%d is not in the heap: the search read all %d keys", key, len(h.a)))
		return false
	}
	h.rec.step("found", fmt.Sprintf("%d is at index %d, after %d comparisons", key, i, h.comparisons), key)
	return true
}

func (h *Heap) swap(i, j int) {
	h.a[i], h.a[j] = h.a[j], h.a[i]
	h.pos[h.a[i]], h.pos[h.a[j]] = i, j
	h.restructures++
}

// step:start

func (h *Heap) up(i int) {
	for i > 0 {
		p := (i - 1) / 2
		h.comparisons++
		if h.a[p] <= h.a[i] {
			return
		}
		h.swap(i, p)
		h.rec.step("swap", fmt.Sprintf("%d < %d: it swaps with its parent", h.a[p], h.a[i]), h.a[p], h.a[i])
		i = p
	}
}

func (h *Heap) down(i int) {
	for {
		small := i
		for _, c := range []int{2*i + 1, 2*i + 2} {
			if c < len(h.a) {
				h.comparisons++
				if h.a[c] < h.a[small] {
					small = c
				}
			}
		}
		if small == i {
			return
		}
		h.swap(i, small)
		h.rec.step("swap", fmt.Sprintf("%d > %d: it swaps with its smaller child", h.a[small], h.a[i]), h.a[small], h.a[i])
		i = small
	}
}

// step:end

func (h *Heap) Insert(key int) bool {
	h.comparisons = 0
	if _, ok := h.pos[key]; ok {
		return false
	}
	h.a = append(h.a, key)
	h.pos[key] = len(h.a) - 1
	h.rec.step("insert", fmt.Sprintf("%d goes into the next free place, at index %d", key, len(h.a)-1), key)
	h.up(len(h.a) - 1)
	return true
}

// Delete removes any key. The last key fills its place, and then goes up
// or down. To pop the smallest key, delete the root.
func (h *Heap) Delete(key int) bool {
	h.comparisons = 0
	i, ok := h.pos[key]
	if !ok {
		h.rec.step("missing", fmt.Sprintf("%d is not in the heap", key))
		return false
	}
	last := len(h.a) - 1
	if i != last {
		h.swap(i, last)
	}
	h.a = h.a[:last]
	delete(h.pos, key)
	h.rec.step("delete", fmt.Sprintf("%d is cut, and the last key fills its place", key))
	if i < len(h.a) {
		h.down(i)
		h.up(i)
	}
	return true
}

func (h *Heap) Check() error {
	seen := map[int]bool{}
	for i, v := range h.a {
		if seen[v] {
			return fmt.Errorf("%d is in the heap twice", v)
		}
		seen[v] = true
		if i > 0 && h.a[(i-1)/2] > v {
			return fmt.Errorf("the parent of %d is bigger", v)
		}
		if h.pos[v] != i {
			return fmt.Errorf("the position of %d is %d, want %d", v, h.pos[v], i)
		}
	}
	if len(h.pos) != len(h.a) {
		return fmt.Errorf("%d positions for %d keys", len(h.pos), len(h.a))
	}
	return nil
}
