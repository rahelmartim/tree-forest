package trees

import (
	"fmt"
	"slices"
)

// BTree is a B-tree of minimum degree t, after Cormen, Leiserson, Rivest,
// and Stein. A node holds from t-1 to 2t-1 keys (the root can hold fewer),
// and all leaves are at the same depth. With t = 2, it is a 2-3-4 tree.
type BTree struct {
	root         *bnode
	t            int
	size         int
	restructures int
	comparisons  int
	rec          *Recorder
}

type bnode struct {
	keys []int
	kids []*bnode
}

func (n *bnode) leaf() bool { return len(n.kids) == 0 }

// NewBTree makes a B-tree with a minimum degree from 2 to 5.
func NewBTree(t int) *BTree {
	return &BTree{root: &bnode{}, t: max(2, min(5, t))}
}

func (b *BTree) Record(on bool) *Recorder {
	if !on {
		b.rec = nil
		return nil
	}
	b.rec = &Recorder{snap: b.Snapshot}
	return b.rec
}

func (b *BTree) Snapshot() *Node {
	if b.size == 0 {
		return nil
	}
	var snap func(n *bnode) *Node
	snap = func(n *bnode) *Node {
		s := &Node{Keys: slices.Clone(n.keys)}
		for _, k := range n.kids {
			s.Children = append(s.Children, snap(k))
		}
		return s
	}
	return snap(b.root)
}

func (b *BTree) Stats() Stats {
	h := 0
	if b.size > 0 {
		for n := b.root; ; n = n.kids[0] {
			h++
			if n.leaf() {
				break
			}
		}
	}
	return Stats{Size: b.size, Height: h, Restructures: b.restructures, Comparisons: b.comparisons}
}

func (b *BTree) Keys() []int {
	var out []int
	var walk func(n *bnode)
	walk = func(n *bnode) {
		for i, k := range n.keys {
			if !n.leaf() {
				walk(n.kids[i])
			}
			out = append(out, k)
		}
		if !n.leaf() {
			walk(n.kids[len(n.keys)])
		}
	}
	walk(b.root)
	return out
}

// index returns the first position with a key ≥ key, and counts the
// comparisons.
func (b *BTree) index(n *bnode, key int) int {
	i := 0
	for i < len(n.keys) && key > n.keys[i] {
		i++
		b.comparisons++
	}
	if i < len(n.keys) {
		b.comparisons++
	}
	return i
}

func (b *BTree) Search(key int) bool {
	b.comparisons = 0
	n := b.root
	for {
		i := b.index(n, key)
		if i < len(n.keys) && n.keys[i] == key {
			b.rec.step("found", fmt.Sprintf("%d is here", key), key)
			return true
		}
		if n.leaf() {
			b.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
			return false
		}
		b.rec.step("compare", fmt.Sprintf("%d goes down between %s", key, between(n.keys, i)), n.keys...)
		n = n.kids[i]
	}
}

func between(keys []int, i int) string {
	switch {
	case len(keys) == 0:
		return "nothing"
	case i == 0:
		return fmt.Sprintf("the start and %d", keys[0])
	case i == len(keys):
		return fmt.Sprintf("%d and the end", keys[i-1])
	}
	return fmt.Sprintf("%d and %d", keys[i-1], keys[i])
}

func (b *BTree) contains(key int) bool {
	n := b.root
	for {
		i, found := slices.BinarySearch(n.keys, key)
		if found {
			return true
		}
		if n.leaf() {
			return false
		}
		n = n.kids[i]
	}
}

// step:insert:start

// Insert splits each full node on the way down, so that the leaf always
// has room for the key.
func (b *BTree) Insert(key int) bool {
	b.comparisons = 0
	if b.contains(key) {
		return false
	}
	if len(b.root.keys) == 2*b.t-1 {
		old := b.root
		b.root = &bnode{kids: []*bnode{old}}
		b.split(b.root, 0)
	}
	n := b.root
	for {
		i := b.index(n, key)
		if n.leaf() {
			n.keys = slices.Insert(n.keys, i, key)
			b.size++
			b.rec.step("insert", fmt.Sprintf("%d goes into the leaf", key), key)
			return true
		}
		b.rec.step("compare", fmt.Sprintf("%d goes down between %s", key, between(n.keys, i)), n.keys...)
		if len(n.kids[i].keys) == 2*b.t-1 {
			b.split(n, i)
			if key > n.keys[i] {
				i++
			}
		}
		n = n.kids[i]
	}
}

// split moves the middle key of the full child i up into n.
func (b *BTree) split(n *bnode, i int) {
	y := n.kids[i]
	mid := y.keys[b.t-1]
	z := &bnode{keys: slices.Clone(y.keys[b.t:])}
	if !y.leaf() {
		z.kids = slices.Clone(y.kids[b.t:])
		y.kids = y.kids[:b.t:b.t]
	}
	y.keys = y.keys[: b.t-1 : b.t-1]
	n.keys = slices.Insert(n.keys, i, mid)
	n.kids = slices.Insert(n.kids, i+1, z)
	b.restructures++
	b.rec.step("split", fmt.Sprintf("a full node splits: %d goes up, and two nodes stay below", mid), mid)
}

// step:insert:end

// Delete removes a key in one pass down. Before it goes into a child
// with only t-1 keys, it gives that child one more key: it borrows from a
// sibling, or it merges two children.
func (b *BTree) Delete(key int) bool {
	b.comparisons = 0
	if !b.contains(key) {
		b.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		return false
	}
	b.delete(b.root, key)
	b.size--
	if len(b.root.keys) == 0 && !b.root.leaf() {
		b.root = b.root.kids[0]
		b.rec.step("merge", "the root is empty: the tree is one level shorter")
	}
	return true
}

func (b *BTree) delete(n *bnode, key int) {
	t := b.t
	i := b.index(n, key)
	if i < len(n.keys) && n.keys[i] == key {
		switch {
		case n.leaf():
			n.keys = slices.Delete(n.keys, i, i+1)
			b.rec.step("delete", fmt.Sprintf("%d is cut from the leaf", key))
		case len(n.kids[i].keys) >= t:
			pred := b.last(n.kids[i])
			n.keys[i] = pred
			b.rec.step("delete", fmt.Sprintf("%d is cut: the key before it, %d, takes its place", key, pred), pred)
			b.delete(n.kids[i], pred)
		case len(n.kids[i+1].keys) >= t:
			succ := b.first(n.kids[i+1])
			n.keys[i] = succ
			b.rec.step("delete", fmt.Sprintf("%d is cut: the key after it, %d, takes its place", key, succ), succ)
			b.delete(n.kids[i+1], succ)
		default:
			b.merge(n, i)
			b.delete(n.kids[i], key)
		}
		return
	}
	if len(n.kids[i].keys) == t-1 {
		switch {
		case i > 0 && len(n.kids[i-1].keys) >= t:
			b.borrowLeft(n, i)
		case i < len(n.kids)-1 && len(n.kids[i+1].keys) >= t:
			b.borrowRight(n, i)
		case i < len(n.kids)-1:
			b.merge(n, i)
		default:
			b.merge(n, i-1)
			i--
		}
	}
	b.delete(n.kids[i], key)
}

func (b *BTree) last(n *bnode) int {
	for !n.leaf() {
		n = n.kids[len(n.kids)-1]
	}
	return n.keys[len(n.keys)-1]
}

func (b *BTree) first(n *bnode) int {
	for !n.leaf() {
		n = n.kids[0]
	}
	return n.keys[0]
}

// merge joins child i, the key i of n, and child i+1 into one node.
func (b *BTree) merge(n *bnode, i int) {
	child, sibling := n.kids[i], n.kids[i+1]
	mid := n.keys[i]
	child.keys = append(append(child.keys, mid), sibling.keys...)
	child.kids = append(child.kids, sibling.kids...)
	n.keys = slices.Delete(n.keys, i, i+1)
	n.kids = slices.Delete(n.kids, i+1, i+2)
	b.restructures++
	b.rec.step("merge", fmt.Sprintf("two small nodes merge, with %d between them", mid), mid)
}

// borrowLeft moves a key from the left sibling, through n, into child i.
func (b *BTree) borrowLeft(n *bnode, i int) {
	child, left := n.kids[i], n.kids[i-1]
	child.keys = slices.Insert(child.keys, 0, n.keys[i-1])
	n.keys[i-1] = left.keys[len(left.keys)-1]
	left.keys = left.keys[:len(left.keys)-1]
	if !left.leaf() {
		child.kids = slices.Insert(child.kids, 0, left.kids[len(left.kids)-1])
		left.kids = left.kids[:len(left.kids)-1]
	}
	b.restructures++
	b.rec.step("borrow", fmt.Sprintf("a small node borrows a key from its left sibling: %d goes up", n.keys[i-1]), n.keys[i-1], child.keys[0])
}

// borrowRight moves a key from the right sibling, through n, into child i.
func (b *BTree) borrowRight(n *bnode, i int) {
	child, right := n.kids[i], n.kids[i+1]
	child.keys = append(child.keys, n.keys[i])
	n.keys[i] = right.keys[0]
	right.keys = slices.Delete(right.keys, 0, 1)
	if !right.leaf() {
		child.kids = append(child.kids, right.kids[0])
		right.kids = slices.Delete(right.kids, 0, 1)
	}
	b.restructures++
	b.rec.step("borrow", fmt.Sprintf("a small node borrows a key from its right sibling: %d goes up", n.keys[i]), n.keys[i], child.keys[len(child.keys)-1])
}

func (b *BTree) Check() error {
	keys := b.Keys()
	if len(keys) != b.size {
		return fmt.Errorf("size %d, but %d keys", b.size, len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			return fmt.Errorf("the keys are out of order at %d", keys[i])
		}
	}
	leafDepth := -1
	var walk func(n *bnode, depth int, root bool) error
	walk = func(n *bnode, depth int, root bool) error {
		if !root && (len(n.keys) < b.t-1 || len(n.keys) > 2*b.t-1) {
			return fmt.Errorf("a node has %d keys, want %d to %d", len(n.keys), b.t-1, 2*b.t-1)
		}
		if root && len(n.keys) > 2*b.t-1 {
			return fmt.Errorf("the root has %d keys", len(n.keys))
		}
		if n.leaf() {
			if leafDepth == -1 {
				leafDepth = depth
			}
			if depth != leafDepth {
				return fmt.Errorf("leaves at depths %d and %d", leafDepth, depth)
			}
			return nil
		}
		if len(n.kids) != len(n.keys)+1 {
			return fmt.Errorf("a node with %d keys has %d children", len(n.keys), len(n.kids))
		}
		for _, k := range n.kids {
			if err := walk(k, depth+1, false); err != nil {
				return err
			}
		}
		return nil
	}
	if b.size > 0 && len(b.root.keys) == 0 {
		return fmt.Errorf("the root is empty")
	}
	return walk(b.root, 0, true)
}
