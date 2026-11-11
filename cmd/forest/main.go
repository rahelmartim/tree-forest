// Forest prints a tree in the terminal, on its side: the root is on the
// left, and the right child is above its parent.
//
//	go run ./cmd/forest -tree avl -insert 1,2,3,4,5,6,7
//	go run ./cmd/forest -tree btree -insert 1,2,3,4,5,6,7,8,9 -delete 5 -steps
package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/rahelmartim/tree-forest/trees"
)

func main() {
	kind := flag.String("tree", "avl", "the tree: bst, avl, redblack, splay, treap, btree, or heap")
	insert := flag.String("insert", "50,30,70,20,40,60,80", "keys to insert, with commas")
	del := flag.String("delete", "", "keys to delete after the inserts, with commas")
	steps := flag.Bool("steps", false, "print every step")
	flag.Parse()

	k, ok := trees.FindKind(*kind)
	if !ok {
		log.Fatalf("unknown tree %q", *kind)
	}
	t := k.New()
	run := func(list string, op func(int) bool, name string) {
		for _, f := range strings.Split(list, ",") {
			if f = strings.TrimSpace(f); f == "" {
				continue
			}
			key, err := strconv.Atoi(f)
			if err != nil {
				log.Fatalf("%q is not a number", f)
			}
			rec := t.Record(*steps)
			op(key)
			if rec != nil {
				fmt.Printf("%s %d\n", name, key)
				for _, s := range rec.Steps {
					fmt.Printf("  %-8s %s\n", s.Kind, s.Note)
				}
			}
		}
	}
	run(*insert, t.Insert, "insert")
	run(*del, t.Delete, "delete")

	s := t.Stats()
	fmt.Printf("%s: %d keys, height %d, %d %s\n\n", k.Name, s.Size, s.Height, s.Restructures, k.Restructures)
	print(t.Snapshot(), "", true)
}

func label(n *trees.Node) string {
	parts := make([]string, len(n.Keys))
	for i, k := range n.Keys {
		parts[i] = strconv.Itoa(k)
	}
	text := strings.Join(parts, " ")
	if len(n.Keys) > 1 {
		text = "[" + text + "]"
	}
	if n.Color == "red" {
		text += " (red)"
	}
	if n.Info != "" {
		text += " " + n.Info
	}
	return text
}

// print draws the children from the last to the first, so that the tree
// reads from the top: the biggest keys are above.
func print(n *trees.Node, prefix string, root bool) {
	if n == nil {
		return
	}
	for i := len(n.Children) - 1; i >= len(n.Children)/2; i-- {
		print(n.Children[i], prefix+"      ", false)
	}
	mark := "── "
	if root {
		mark = ""
	}
	fmt.Println(prefix + mark + label(n))
	for i := len(n.Children)/2 - 1; i >= 0; i-- {
		print(n.Children[i], prefix+"      ", false)
	}
}
