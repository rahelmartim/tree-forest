package trees

import (
	"fmt"
	"math/rand/v2"
)

// remove takes a node out of a binary search tree. A node with two
// children trades places with its successor first. It returns the lowest
// node whose subtree changed, for the trees that fix heights.
func (t *binary) remove(z *node) (from *node) {
	switch {
	case z.left == nil:
		from = z.parent
		t.replace(z, z.right)
	case z.right == nil:
		from = z.parent
		t.replace(z, z.left)
	default:
		y := minimum(z.right)
		t.rec.step("compare", fmt.Sprintf("%d has two children: the smallest key on its right, %d, takes its place", z.key, y.key), z.key, y.key)
		if y.parent == z {
			from = y
		} else {
			from = y.parent
			t.replace(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.replace(z, y)
		y.left = z.left
		y.left.parent = y
	}
	t.size--
	return from
}

/* Binary search tree */

// BST is a binary search tree with no balance.
type BST struct{ binary }

func NewBST() *BST { return &BST{} }

func (t *BST) Insert(key int) bool {
	n, last := t.find(key)
	if n != nil {
		return false
	}
	t.attach(&node{key: key, height: 1}, last)
	t.rec.step("insert", fmt.Sprintf("%d grows as a new leaf", key), key)
	return true
}

func (t *BST) Delete(key int) bool {
	n, _ := t.find(key)
	if n == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		return false
	}
	t.remove(n)
	t.rec.step("delete", fmt.Sprintf("%d is cut from the tree", key))
	return true
}

func (t *BST) Check() error { return t.checkOrder() }

/* AVL tree */

// AVL keeps the heights of the two subtrees of each node within 1.
type AVL struct{ binary }

func NewAVL() *AVL {
	t := &AVL{}
	t.info = func(n *node) string { return fmt.Sprintf("%+d", balance(n)) }
	return t
}

func balance(n *node) int { return avlHeight(n.left) - avlHeight(n.right) }

func (t *AVL) Insert(key int) bool {
	n, last := t.find(key)
	if n != nil {
		return false
	}
	leaf := &node{key: key, height: 1}
	t.attach(leaf, last)
	t.rec.step("insert", fmt.Sprintf("%d grows as a new leaf", key), key)
	t.retrace(last)
	return true
}

func (t *AVL) Delete(key int) bool {
	n, _ := t.find(key)
	if n == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		return false
	}
	from := t.remove(n)
	t.rec.step("delete", fmt.Sprintf("%d is cut from the tree", key))
	t.retrace(from)
	return true
}

// retrace walks up from n, fixes the heights, and rotates where a node
// is out of balance.
func (t *AVL) retrace(n *node) {
	for n != nil {
		n.fixHeight()
		if b := balance(n); b > 1 || b < -1 {
			n = t.rebalance(n)
		}
		n = n.parent
	}
}

func (t *AVL) rebalance(z *node) *node {
	b := balance(z)
	t.rec.step("compare", fmt.Sprintf("%d is out of balance (%+d)", z.key, b), z.key)
	if b > 1 {
		if balance(z.left) < 0 {
			y := z.left
			t.rotateLeft(y)
			t.rec.step("rotate", fmt.Sprintf("left-right case: rotate left at %d first", y.key), y.key, y.parent.key)
		}
		t.rotateRight(z)
		t.rec.step("rotate", fmt.Sprintf("rotate right at %d: %d goes up", z.key, z.parent.key), z.key, z.parent.key)
	} else {
		if balance(z.right) > 0 {
			y := z.right
			t.rotateRight(y)
			t.rec.step("rotate", fmt.Sprintf("right-left case: rotate right at %d first", y.key), y.key, y.parent.key)
		}
		t.rotateLeft(z)
		t.rec.step("rotate", fmt.Sprintf("rotate left at %d: %d goes up", z.key, z.parent.key), z.key, z.parent.key)
	}
	return z.parent
}

func (t *AVL) Check() error {
	if err := t.checkOrder(); err != nil {
		return err
	}
	var walk func(n *node) (int, error)
	walk = func(n *node) (int, error) {
		if n == nil {
			return 0, nil
		}
		l, err := walk(n.left)
		if err != nil {
			return 0, err
		}
		r, err := walk(n.right)
		if err != nil {
			return 0, err
		}
		if l-r > 1 || r-l > 1 {
			return 0, fmt.Errorf("%d is out of balance: heights %d and %d", n.key, l, r)
		}
		if n.height != 1+max(l, r) {
			return 0, fmt.Errorf("the height of %d is %d, want %d", n.key, n.height, 1+max(l, r))
		}
		return 1 + max(l, r), nil
	}
	_, err := walk(t.root)
	return err
}

/* Red-black tree */

// RedBlack is a red-black tree, after Cormen, Leiserson, Rivest, and Stein.
type RedBlack struct{ binary }

func NewRedBlack() *RedBlack {
	t := &RedBlack{}
	t.color = true
	return t
}

func isRed(n *node) bool { return n != nil && n.red }

func (t *RedBlack) Insert(key int) bool {
	n, last := t.find(key)
	if n != nil {
		return false
	}
	z := &node{key: key, red: true, height: 1}
	t.attach(z, last)
	t.rec.step("insert", fmt.Sprintf("%d grows as a new red leaf", key), key)
	for isRed(z.parent) {
		p := z.parent
		g := p.parent // the root is black, so a red parent has a parent
		uncle := g.left
		if p == g.left {
			uncle = g.right
		}
		if isRed(uncle) {
			p.red, uncle.red, g.red = false, false, true
			t.rec.step("recolor", fmt.Sprintf("the parent %d and the uncle %d are red: they turn black, and %d turns red", p.key, uncle.key, g.key), p.key, uncle.key, g.key)
			z = g
			continue
		}
		if p == g.left {
			if z == p.right {
				z = p
				t.rotateLeft(z)
				t.rec.step("rotate", fmt.Sprintf("%d is an inner child: rotate left at %d", z.parent.key, z.key), z.key, z.parent.key)
				p = z.parent
			}
			p.red, g.red = false, true
			t.rotateRight(g)
			t.rec.step("rotate", fmt.Sprintf("rotate right at %d, and swap the colors of %d and %d", g.key, p.key, g.key), g.key, p.key)
		} else {
			if z == p.left {
				z = p
				t.rotateRight(z)
				t.rec.step("rotate", fmt.Sprintf("%d is an inner child: rotate right at %d", z.parent.key, z.key), z.key, z.parent.key)
				p = z.parent
			}
			p.red, g.red = false, true
			t.rotateLeft(g)
			t.rec.step("rotate", fmt.Sprintf("rotate left at %d, and swap the colors of %d and %d", g.key, p.key, g.key), g.key, p.key)
		}
	}
	if t.root.red {
		t.root.red = false
		t.rec.step("recolor", fmt.Sprintf("the root %d turns black", t.root.key), t.root.key)
	}
	return true
}

func (t *RedBlack) Delete(key int) bool {
	z, _ := t.find(key)
	if z == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		return false
	}
	removedRed := z.red
	var x, xp *node
	switch {
	case z.left == nil:
		x, xp = z.right, z.parent
		t.replace(z, z.right)
	case z.right == nil:
		x, xp = z.left, z.parent
		t.replace(z, z.left)
	default:
		y := minimum(z.right)
		t.rec.step("compare", fmt.Sprintf("%d has two children: %d takes its place and its color", z.key, y.key), z.key, y.key)
		removedRed = y.red
		x = y.right
		if y.parent == z {
			xp = y
		} else {
			xp = y.parent
			t.replace(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.replace(z, y)
		y.left = z.left
		y.left.parent = y
		y.red = z.red
	}
	t.size--
	t.rec.step("delete", fmt.Sprintf("%d is cut from the tree", key))
	if !removedRed {
		t.deleteFixup(x, xp)
	}
	return true
}

// deleteFixup gives back the black node that a path lost. x is the node
// with an extra black, and xp is its parent (x can be nil).
func (t *RedBlack) deleteFixup(x, xp *node) {
	for x != t.root && !isRed(x) {
		if x == xp.left {
			w := xp.right
			if isRed(w) {
				w.red, xp.red = false, true
				t.rotateLeft(xp)
				t.rec.step("rotate", fmt.Sprintf("the sibling %d is red: rotate left at %d", w.key, xp.key), w.key, xp.key)
				w = xp.right
			}
			if !isRed(w.left) && !isRed(w.right) {
				w.red = true
				t.rec.step("recolor", fmt.Sprintf("the sibling %d has black children: it turns red, and the problem moves up", w.key), w.key)
				x, xp = xp, xp.parent
				continue
			}
			if !isRed(w.right) {
				w.left.red, w.red = false, true
				t.rotateRight(w)
				t.rec.step("rotate", fmt.Sprintf("rotate right at the sibling %d", w.key), w.key)
				w = xp.right
			}
			w.red, xp.red, w.right.red = xp.red, false, false
			t.rotateLeft(xp)
			t.rec.step("rotate", fmt.Sprintf("rotate left at %d: the paths have the same black count again", xp.key), xp.key, w.key)
			x, xp = t.root, nil
		} else {
			w := xp.left
			if isRed(w) {
				w.red, xp.red = false, true
				t.rotateRight(xp)
				t.rec.step("rotate", fmt.Sprintf("the sibling %d is red: rotate right at %d", w.key, xp.key), w.key, xp.key)
				w = xp.left
			}
			if !isRed(w.left) && !isRed(w.right) {
				w.red = true
				t.rec.step("recolor", fmt.Sprintf("the sibling %d has black children: it turns red, and the problem moves up", w.key), w.key)
				x, xp = xp, xp.parent
				continue
			}
			if !isRed(w.left) {
				w.right.red, w.red = false, true
				t.rotateLeft(w)
				t.rec.step("rotate", fmt.Sprintf("rotate left at the sibling %d", w.key), w.key)
				w = xp.left
			}
			w.red, xp.red, w.left.red = xp.red, false, false
			t.rotateRight(xp)
			t.rec.step("rotate", fmt.Sprintf("rotate right at %d: the paths have the same black count again", xp.key), xp.key, w.key)
			x, xp = t.root, nil
		}
	}
	if x != nil && x.red {
		x.red = false
		t.rec.step("recolor", fmt.Sprintf("%d turns black", x.key), x.key)
	}
}

func (t *RedBlack) Check() error {
	if err := t.checkOrder(); err != nil {
		return err
	}
	if isRed(t.root) {
		return fmt.Errorf("the root is red")
	}
	var walk func(n *node) (int, error)
	walk = func(n *node) (int, error) {
		if n == nil {
			return 1, nil
		}
		if n.red && (isRed(n.left) || isRed(n.right)) {
			return 0, fmt.Errorf("the red node %d has a red child", n.key)
		}
		l, err := walk(n.left)
		if err != nil {
			return 0, err
		}
		r, err := walk(n.right)
		if err != nil {
			return 0, err
		}
		if l != r {
			return 0, fmt.Errorf("the paths under %d have %d and %d black nodes", n.key, l, r)
		}
		if !n.red {
			l++
		}
		return l, nil
	}
	_, err := walk(t.root)
	return err
}

/* Splay tree */

// Splay moves each node that it touches to the root.
type Splay struct{ binary }

func NewSplay() *Splay { return &Splay{} }

func (t *Splay) splay(x *node) {
	for x.parent != nil {
		p := x.parent
		g := p.parent
		switch {
		case g == nil:
			if x == p.left {
				t.rotateRight(p)
			} else {
				t.rotateLeft(p)
			}
			t.rec.step("splay", fmt.Sprintf("zig: %d goes up to the root", x.key), x.key)
		case (x == p.left) == (p == g.left):
			if x == p.left {
				t.rotateRight(g)
				t.rotateRight(p)
			} else {
				t.rotateLeft(g)
				t.rotateLeft(p)
			}
			t.rec.step("splay", fmt.Sprintf("zig-zig: %d goes up two levels, on the same side", x.key), x.key)
		default:
			if x == p.left {
				t.rotateRight(p)
				t.rotateLeft(g)
			} else {
				t.rotateLeft(p)
				t.rotateRight(g)
			}
			t.rec.step("splay", fmt.Sprintf("zig-zag: %d goes up two levels, from the inside", x.key), x.key)
		}
	}
}

func (t *Splay) Search(key int) bool {
	n, last := t.find(key)
	found := n != nil
	if !found {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree: the last node, %d, goes to the root", key, keyOf(last)))
		n = last
	}
	if n != nil {
		t.splay(n)
	}
	return found
}

func keyOf(n *node) int {
	if n == nil {
		return 0
	}
	return n.key
}

func (t *Splay) Insert(key int) bool {
	n, last := t.find(key)
	if n != nil {
		t.splay(n)
		return false
	}
	z := &node{key: key, height: 1}
	t.attach(z, last)
	t.rec.step("insert", fmt.Sprintf("%d grows as a new leaf, then goes to the root", key), key)
	t.splay(z)
	return true
}

func (t *Splay) Delete(key int) bool {
	n, last := t.find(key)
	if n == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		if last != nil {
			t.splay(last)
		}
		return false
	}
	t.splay(n)
	left, right := n.left, n.right
	t.size--
	if left == nil {
		t.root = right
		if right != nil {
			right.parent = nil
		}
		t.rec.step("delete", fmt.Sprintf("%d is cut: its right subtree is the tree now", key))
		return true
	}
	left.parent = nil
	t.root = left
	m := maximum(left)
	t.splay(m)
	m.right = right
	if right != nil {
		right.parent = m
	}
	t.rec.step("delete", fmt.Sprintf("%d is cut: the biggest key on its left, %d, holds both halves", key, m.key), m.key)
	return true
}

func (t *Splay) Check() error { return t.checkOrder() }

/* Treap */

// Treap is a binary search tree by key and a heap by a random priority:
// each parent has a priority at least as high as its children.
type Treap struct {
	binary
	rng *rand.Rand
}

func NewTreap(seed uint64) *Treap {
	t := &Treap{rng: rand.New(rand.NewPCG(seed, seed^0xda3e39cb94b95bdb))}
	t.info = func(n *node) string { return fmt.Sprintf("p%d", n.prio) }
	return t
}

func (t *Treap) Insert(key int) bool {
	n, last := t.find(key)
	if n != nil {
		return false
	}
	z := &node{key: key, height: 1, prio: uint32(1 + t.rng.IntN(99))}
	t.attach(z, last)
	t.rec.step("insert", fmt.Sprintf("%d grows as a new leaf with the random priority %d", key, z.prio), key)
	for z.parent != nil && z.prio > z.parent.prio {
		p := z.parent
		if z == p.left {
			t.rotateRight(p)
		} else {
			t.rotateLeft(p)
		}
		t.rec.step("rotate", fmt.Sprintf("priority %d > %d: %d goes up over %d", z.prio, p.prio, z.key, p.key), z.key, p.key)
	}
	return true
}

func (t *Treap) Delete(key int) bool {
	n, _ := t.find(key)
	if n == nil {
		t.rec.step("missing", fmt.Sprintf("%d is not in the tree", key))
		return false
	}
	// Rotate the node down, under the child with the higher priority,
	// until it is a leaf.
	for n.left != nil || n.right != nil {
		if n.right == nil || (n.left != nil && n.left.prio > n.right.prio) {
			c := n.left
			t.rotateRight(n)
			t.rec.step("rotate", fmt.Sprintf("%d goes down: its child %d has the higher priority", key, c.key), key, c.key)
		} else {
			c := n.right
			t.rotateLeft(n)
			t.rec.step("rotate", fmt.Sprintf("%d goes down: its child %d has the higher priority", key, c.key), key, c.key)
		}
	}
	t.replace(n, nil)
	t.size--
	t.rec.step("delete", fmt.Sprintf("%d is a leaf now, and it is cut", key))
	return true
}

func (t *Treap) Check() error {
	if err := t.checkOrder(); err != nil {
		return err
	}
	var walk func(n *node) error
	walk = func(n *node) error {
		if n == nil {
			return nil
		}
		for _, c := range []*node{n.left, n.right} {
			if c != nil && c.prio > n.prio {
				return fmt.Errorf("the child %d has a higher priority than %d", c.key, n.key)
			}
		}
		if err := walk(n.left); err != nil {
			return err
		}
		return walk(n.right)
	}
	return walk(t.root)
}
