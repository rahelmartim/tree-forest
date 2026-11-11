// Package trees has classic search trees and a heap: BST, AVL, red-black,
// splay, treap, B-tree, and a binary min-heap. Each one can record the
// steps of an operation (comparisons, rotations, recolors, splits), so
// that a page can play them as an animation.
package trees

import "fmt"

// Node is a snapshot of one node, for the page. A binary tree has two
// children, and a missing child is nil.
type Node struct {
	Keys     []int   `json:"keys"`
	Children []*Node `json:"children,omitempty"`
	Color    string  `json:"color,omitempty"` // red or black, in a red-black tree
	Info     string  `json:"info,omitempty"`  // such as the height or the priority
}

// Step is one moment of an operation.
type Step struct {
	Kind string `json:"kind"` // compare, insert, delete, found, missing, rotate, recolor, split, merge, borrow, swap, splay
	Note string `json:"note"`
	Tree *Node  `json:"tree"`
	Mark []int  `json:"mark"` // the keys to highlight
}

// Recorder keeps the steps of operations. A nil recorder keeps nothing,
// for the benchmarks.
type Recorder struct {
	Steps []Step
	snap  func() *Node
}

func (r *Recorder) step(kind, note string, mark ...int) {
	if r == nil || r.snap == nil {
		return
	}
	if mark == nil {
		mark = []int{}
	}
	r.Steps = append(r.Steps, Step{Kind: kind, Note: note, Tree: r.snap(), Mark: mark})
}

// Stats describes the tree.
type Stats struct {
	Size         int `json:"size"`
	Height       int `json:"height"`       // levels: an empty tree has 0
	Restructures int `json:"restructures"` // rotations, splits and merges, or swaps, since the start
	Comparisons  int `json:"comparisons"`  // key comparisons in the last operation
}

// Tree is the interface of every structure in this package.
type Tree interface {
	Insert(key int) bool // false if the key is already there
	Delete(key int) bool // false if the key is not there
	Search(key int) bool
	Snapshot() *Node
	Stats() Stats
	Keys() []int  // in order (for a heap: in the order of its array)
	Check() error // the invariants of the structure
	// Record starts or stops the record of steps.
	Record(on bool) *Recorder
}

// Kind is one kind of tree.
type Kind struct {
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	Summary      string      `json:"summary"`
	Restructures string      `json:"restructures"` // what Stats.Restructures counts
	Search       string      `json:"search"`       // the cost of a search
	New          func() Tree `json:"-"`
}

// Kinds lists the structures, in the order of the page.
var Kinds = []Kind{
	{"bst", "Binary search tree", "Smaller keys go left, bigger keys go right. Nothing keeps it balanced, so keys in order make a long line.", "none", "O(height): O(n) in the worst case", func() Tree { return NewBST() }},
	{"avl", "AVL tree", "A BST in which the heights of the two subtrees of every node differ by at most 1. Rotations fix it after each change.", "rotations", "O(log n)", func() Tree { return NewAVL() }},
	{"redblack", "Red-black tree", "A BST with a color on each node. No red node has a red child, and every path down has the same number of black nodes.", "rotations", "O(log n)", func() Tree { return NewRedBlack() }},
	{"splay", "Splay tree", "A BST that moves each key that it touches to the root. Keys that you use often stay near the top.", "rotations", "O(log n) on average over many operations", func() Tree { return NewSplay() }},
	{"treap", "Treap", "A BST by key and a heap by a random priority. The random priorities keep it balanced on average.", "rotations", "O(log n) expected", func() Tree { return NewTreap(1) }},
	{"btree", "B-tree", "Each node holds many keys, and all leaves are at the same depth. Full nodes split, and nodes with too few keys merge.", "splits and merges", "O(log n)", func() Tree { return NewBTree(2) }},
	{"heap", "Binary heap", "A complete binary tree in an array. Each parent is smaller than its children, so the smallest key is at the root.", "swaps", "O(n) for any key, O(1) for the smallest", func() Tree { return NewHeap() }},
}

// FindKind returns the kind with the key.
func FindKind(key string) (Kind, bool) {
	for _, k := range Kinds {
		if k.Key == key {
			return k, true
		}
	}
	return Kind{}, false
}

/* The binary trees share nodes with parent links, and rotations. */

type node struct {
	key                 int
	left, right, parent *node
	height              int    // AVL
	red                 bool   // red-black
	prio                uint32 // treap
}

type binary struct {
	root         *node
	size         int
	restructures int
	comparisons  int
	rec          *Recorder
	info         func(*node) string // the extra text of a node in a snapshot
	color        bool               // show the colors
}

func (t *binary) Record(on bool) *Recorder {
	if !on {
		t.rec = nil
		return nil
	}
	t.rec = &Recorder{snap: t.Snapshot}
	return t.rec
}

func (t *binary) Snapshot() *Node {
	var snap func(n *node) *Node
	snap = func(n *node) *Node {
		if n == nil {
			return nil
		}
		s := &Node{Keys: []int{n.key}, Children: []*Node{snap(n.left), snap(n.right)}}
		if t.info != nil {
			s.Info = t.info(n)
		}
		if t.color {
			s.Color = "black"
			if n.red {
				s.Color = "red"
			}
		}
		return s
	}
	return snap(t.root)
}

func heightOf(n *node) int {
	if n == nil {
		return 0
	}
	return 1 + max(heightOf(n.left), heightOf(n.right))
}

func (t *binary) Stats() Stats {
	return Stats{Size: t.size, Height: heightOf(t.root), Restructures: t.restructures, Comparisons: t.comparisons}
}

func (t *binary) Keys() []int {
	var out []int
	var walk func(n *node)
	walk = func(n *node) {
		if n != nil {
			walk(n.left)
			out = append(out, n.key)
			walk(n.right)
		}
	}
	walk(t.root)
	return out
}

// find walks down to the key. It returns the node, or nil and the last
// node on the way (the parent of a new node).
func (t *binary) find(key int) (found, last *node) {
	t.comparisons = 0
	n := t.root
	for n != nil {
		last = n
		t.comparisons++
		switch {
		case key == n.key:
			t.rec.step("found", fmt.Sprintf("%d is here", key), key)
			return n, last
		case key < n.key:
			t.rec.step("compare", fmt.Sprintf("%d < %d: go left", key, n.key), n.key)
			n = n.left
		default:
			t.rec.step("compare", fmt.Sprintf("%d > %d: go right", key, n.key), n.key)
			n = n.right
		}
	}
	return nil, last
}

// attach adds a new leaf under the parent that find returned.
func (t *binary) attach(n, parent *node) {
	n.parent = parent
	switch {
	case parent == nil:
		t.root = n
	case n.key < parent.key:
		parent.left = n
	default:
		parent.right = n
	}
	t.size++
}

func (t *binary) Search(key int) bool {
	n, _ := t.find(key)
	if n == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
	}
	return n != nil
}

// replace puts v where u was, under u's parent.
func (t *binary) replace(u, v *node) {
	switch {
	case u.parent == nil:
		t.root = v
	case u == u.parent.left:
		u.parent.left = v
	default:
		u.parent.right = v
	}
	if v != nil {
		v.parent = u.parent
	}
}

func minimum(n *node) *node {
	for n.left != nil {
		n = n.left
	}
	return n
}

func maximum(n *node) *node {
	for n.right != nil {
		n = n.right
	}
	return n
}

func avlHeight(n *node) int {
	if n == nil {
		return 0
	}
	return n.height
}

func (n *node) fixHeight() {
	n.height = 1 + max(avlHeight(n.left), avlHeight(n.right))
}

// rotateLeft lifts x.right above x.
//
//	  x              y
//	 / \            / \
//	a   y    →     x   c
//	   / \        / \
//	  b   c      a   b
func (t *binary) rotateLeft(x *node) {
	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	t.replace(x, y)
	y.left = x
	x.parent = y
	x.fixHeight()
	y.fixHeight()
	t.restructures++
}

// rotateRight lifts x.left above x.
func (t *binary) rotateRight(x *node) {
	y := x.left
	x.left = y.right
	if y.right != nil {
		y.right.parent = x
	}
	t.replace(x, y)
	y.right = x
	x.parent = y
	x.fixHeight()
	y.fixHeight()
	t.restructures++
}

// checkOrder tests that the keys are in order and that the parent links
// are right.
func (t *binary) checkOrder() error {
	var walk func(n *node, lo, hi *int) error
	walk = func(n *node, lo, hi *int) error {
		if n == nil {
			return nil
		}
		if (lo != nil && n.key <= *lo) || (hi != nil && n.key >= *hi) {
			return fmt.Errorf("key %d is out of order", n.key)
		}
		for _, c := range []*node{n.left, n.right} {
			if c != nil && c.parent != n {
				return fmt.Errorf("the parent link of %d is wrong", c.key)
			}
		}
		if err := walk(n.left, lo, &n.key); err != nil {
			return err
		}
		return walk(n.right, &n.key, hi)
	}
	if t.root != nil && t.root.parent != nil {
		return fmt.Errorf("the root has a parent")
	}
	if got := len(t.Keys()); got != t.size {
		return fmt.Errorf("size %d, but %d keys", t.size, got)
	}
	return walk(t.root, nil, nil)
}
