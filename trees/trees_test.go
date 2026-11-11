package trees

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func each(t *testing.T, fn func(t *testing.T, k Kind)) {
	for _, k := range Kinds {
		t.Run(k.Key, func(t *testing.T) { fn(t, k) })
	}
}

// TestRandomOperations compares every tree with a Go map, after each of
// thousands of random operations, and checks its invariants each time.
func TestRandomOperations(t *testing.T) {
	each(t, func(t *testing.T, k Kind) {
		for seed := uint64(1); seed <= 20; seed++ {
			rng := rand.New(rand.NewPCG(seed, 7))
			tree := k.New()
			want := map[int]bool{}
			for op := 0; op < 400; op++ {
				key := rng.IntN(60)
				switch rng.IntN(3) {
				case 0, 1:
					if got := tree.Insert(key); got == want[key] {
						t.Fatalf("seed %d: Insert(%d) = %v with the key there = %v", seed, key, got, want[key])
					}
					want[key] = true
				default:
					if got := tree.Delete(key); got != want[key] {
						t.Fatalf("seed %d: Delete(%d) = %v, want %v", seed, key, got, want[key])
					}
					delete(want, key)
				}
				if err := tree.Check(); err != nil {
					t.Fatalf("seed %d, operation %d: %v", seed, op, err)
				}
				if probe := rng.IntN(60); tree.Search(probe) != want[probe] {
					t.Fatalf("seed %d: Search(%d) = %v", seed, probe, !want[probe])
				}
			}
			keys := tree.Keys()
			if k.Key == "heap" {
				slices.Sort(keys)
			}
			var expected []int
			for key := range want {
				expected = append(expected, key)
			}
			slices.Sort(expected)
			if !slices.Equal(keys, expected) || tree.Stats().Size != len(expected) {
				t.Fatalf("seed %d: keys %v, want %v", seed, keys, expected)
			}
		}
	})
}

// TestKeysInOrder is the worst case of a plain BST: it becomes a line.
func TestKeysInOrder(t *testing.T) {
	heights := map[string]int{}
	each(t, func(t *testing.T, k Kind) {
		tree := k.New()
		for key := 1; key <= 127; key++ {
			tree.Insert(key)
		}
		if err := tree.Check(); err != nil {
			t.Fatal(err)
		}
		heights[k.Key] = tree.Stats().Height
	})
	if heights["bst"] != 127 {
		t.Fatalf("a BST of keys in order has height %d, want 127", heights["bst"])
	}
	if heights["avl"] != 7 {
		t.Fatalf("an AVL tree of 127 keys in order has height %d, want 7", heights["avl"])
	}
	if heights["redblack"] > 2*7 || heights["heap"] != 7 || heights["btree"] > 7 {
		t.Fatalf("heights %v", heights)
	}
	if heights["treap"] > 30 {
		t.Fatalf("the treap is too tall: %d", heights["treap"])
	}
}

func TestSplayMovesTheKeyToTheRoot(t *testing.T) {
	s := NewSplay()
	for _, k := range []int{50, 30, 70, 20, 40, 60, 80} {
		s.Insert(k)
	}
	for _, k := range []int{20, 80, 40} {
		s.Search(k)
		if s.root.key != k {
			t.Fatalf("after Search(%d), the root is %d", k, s.root.key)
		}
	}
	if s.Search(45) || s.root.key == 45 {
		t.Fatal("a missing key must not be found")
	}
}

func TestHeapPopsInOrder(t *testing.T) {
	h := NewHeap()
	rng := rand.New(rand.NewPCG(3, 3))
	for range 200 {
		h.Insert(rng.IntN(1000))
	}
	last := -1
	for {
		m, ok := h.Min()
		if !ok {
			break
		}
		if m < last {
			t.Fatalf("the heap popped %d after %d", m, last)
		}
		last = m
		h.Delete(m)
		if err := h.Check(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBTreeDegrees(t *testing.T) {
	for degree := 2; degree <= 5; degree++ {
		b := NewBTree(degree)
		rng := rand.New(rand.NewPCG(uint64(degree), 1))
		for range 2000 {
			key := rng.IntN(500)
			if rng.IntN(3) == 0 {
				b.Delete(key)
			} else {
				b.Insert(key)
			}
			if err := b.Check(); err != nil {
				t.Fatalf("degree %d: %v", degree, err)
			}
		}
	}
}

// TestSteps checks the record of an operation: each step has a note and a
// snapshot, and the last snapshot is the tree after the operation.
func TestSteps(t *testing.T) {
	each(t, func(t *testing.T, k Kind) {
		tree := k.New()
		for _, key := range []int{8, 3, 10, 1, 6, 14, 4, 7, 13} {
			tree.Insert(key)
		}
		rec := tree.Record(true)
		tree.Insert(5)
		if len(rec.Steps) == 0 {
			t.Fatal("no steps")
		}
		for _, s := range rec.Steps {
			if s.Note == "" || s.Kind == "" || s.Mark == nil {
				t.Fatalf("an empty step: %+v", s)
			}
		}
		last := rec.Steps[len(rec.Steps)-1].Tree
		if count(last) != 10 || count(tree.Snapshot()) != 10 {
			t.Fatalf("the last step has %d keys, want 10", count(last))
		}
		tree.Record(false)
		tree.Insert(99)
		if len(rec.Steps) == 0 || tree.Stats().Size != 11 {
			t.Fatal("a stopped recorder must not change the tree")
		}
	})
}

func count(n *Node) int {
	if n == nil {
		return 0
	}
	c := len(n.Keys)
	for _, k := range n.Children {
		c += count(k)
	}
	return c
}

func TestStepsTellTheStory(t *testing.T) {
	avl := NewAVL()
	avl.Insert(1)
	avl.Insert(2)
	rec := avl.Record(true)
	avl.Insert(3)
	notes := ""
	for _, s := range rec.Steps {
		notes += s.Kind + ": " + s.Note + "\n"
	}
	for _, want := range []string{"3 > 1: go right", "out of balance (-2)", "rotate left at 1: 2 goes up"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("the AVL steps have no %q:\n%s", want, notes)
		}
	}

	rb := NewRedBlack()
	rec = rb.Record(true)
	rb.Insert(10)
	if s := rec.Steps[len(rec.Steps)-1]; s.Kind != "recolor" || s.Tree.Color != "black" {
		t.Fatalf("the first key must turn black: %+v", s)
	}

	b := NewBTree(2)
	for _, k := range []int{1, 2, 3} {
		b.Insert(k)
	}
	rec = b.Record(true)
	b.Insert(4)
	if rec.Steps[0].Kind != "split" || b.Stats().Height != 2 {
		t.Fatalf("a full root must split: %+v", rec.Steps[0])
	}
}

func TestStatsAndEmptyTrees(t *testing.T) {
	each(t, func(t *testing.T, k Kind) {
		tree := k.New()
		if tree.Snapshot() != nil || tree.Stats() != (Stats{}) || tree.Delete(1) || tree.Search(1) {
			t.Fatal("an empty tree must be empty")
		}
		tree.Insert(1)
		if tree.Insert(1) {
			t.Fatal("a key went in twice")
		}
		tree.Delete(1)
		if tree.Stats().Size != 0 || tree.Snapshot() != nil {
			t.Fatal("the tree is not empty again")
		}
		if err := tree.Check(); err != nil {
			t.Fatal(err)
		}
	})
	if _, ok := FindKind("avl"); !ok {
		t.Fatal("FindKind")
	}
	if _, ok := FindKind("oak"); ok {
		t.Fatal("FindKind found an oak")
	}
}

func TestOrderAndBench(t *testing.T) {
	for _, p := range Patterns {
		keys, err := Order(p, 9, 1)
		if err != nil {
			t.Fatal(err)
		}
		sorted := slices.Clone(keys)
		slices.Sort(sorted)
		for i, v := range sorted {
			if v != i {
				t.Fatalf("%s is not a permutation: %v", p, keys)
			}
		}
	}
	if z, _ := Order("zigzag", 6, 1); !slices.Equal(z, []int{0, 5, 1, 4, 2, 3}) {
		t.Fatalf("zigzag %v", z)
	}
	if _, err := Order("spiral", 5, 1); err == nil {
		t.Fatal("an unknown pattern must fail")
	}
	each(t, func(t *testing.T, k Kind) {
		r, err := Bench(k.Key, 2000, "random", 1)
		if err != nil {
			t.Fatal(err)
		}
		if r.Keys != 2000 || r.Height == 0 || r.Comparisons < 1 {
			t.Fatalf("%+v", r)
		}
	})
	bst, _ := Bench("bst", 1000000, "sorted", 1)
	if bst.Keys != MaxBenchBSTKeys {
		t.Fatalf("a BST of sorted keys must stop at %d keys, not %d", MaxBenchBSTKeys, bst.Keys)
	}
	avl, _ := Bench("avl", 5000, "sorted", 1)
	if avl.Height > 14 || avl.Restructures == 0 {
		t.Fatalf("AVL on sorted keys: %+v", avl)
	}
}

func BenchmarkInsert(b *testing.B) {
	keys, _ := Order("random", 10000, 1)
	for _, k := range Kinds {
		b.Run(k.Key, func(b *testing.B) {
			for b.Loop() {
				tree := k.New()
				for _, key := range keys {
					tree.Insert(key)
				}
			}
		})
	}
}
