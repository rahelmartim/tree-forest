// Forestgif writes an animated GIF for the README: a BST, an AVL tree, a
// red-black tree, and a B-tree grow from the same keys, 1 to 31 in order.
//
//	go run ./cmd/forestgif -out docs/forest.gif
package main

import (
	"flag"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"

	"github.com/rahelmartim/tree-forest/trees"
)

var palette = color.Palette{
	color.RGBA{246, 243, 232, 255}, // 0 background
	color.RGBA{122, 86, 52, 255},   // 1 bark
	color.RGBA{111, 174, 95, 255},  // 2 leaf
	color.RGBA{63, 125, 78, 255},   // 3 moss
	color.RGBA{192, 57, 43, 255},   // 4 berry
	color.RGBA{46, 42, 38, 255},    // 5 charcoal
	color.RGBA{214, 176, 124, 255}, // 6 plank
	color.RGBA{242, 193, 78, 255},  // 7 sun
	color.RGBA{221, 233, 214, 255}, // 8 panel line
}

const (
	laneW, laneH = 300, 260
	gap          = 12
)

func main() {
	out := flag.String("out", "docs/forest.gif", "file to write")
	keys := flag.Int("keys", 31, "keys to insert, in order")
	flag.Parse()

	kinds := []string{"bst", "avl", "redblack", "btree"}
	forest := make([]trees.Tree, len(kinds))
	for i, k := range kinds {
		kind, _ := trees.FindKind(k)
		forest[i] = kind.New()
	}
	anim := &gif.GIF{}
	for n := 1; n <= *keys; n++ {
		for _, t := range forest {
			t.Insert(n)
		}
		img := image.NewPaletted(image.Rect(0, 0, 2*laneW+gap, 2*laneH+gap), palette)
		for i, t := range forest {
			left, top := (i%2)*(laneW+gap), (i/2)*(laneH+gap)
			frame(img, left, top)
			drawTree(img, t.Snapshot(), kinds[i] == "btree", left, top, n)
		}
		anim.Image = append(anim.Image, img)
		delay := 18
		if n == *keys {
			delay = 300 // hold the end
		}
		anim.Delay = append(anim.Delay, delay)
	}
	file, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if err := gif.EncodeAll(file, anim); err != nil {
		log.Fatal(err)
	}
}

func frame(img *image.Paletted, left, top int) {
	for x := left; x < left+laneW; x++ {
		img.SetColorIndex(x, top, 8)
		img.SetColorIndex(x, top+laneH-1, 8)
	}
	for y := top; y < top+laneH; y++ {
		img.SetColorIndex(left, y, 8)
		img.SetColorIndex(left+laneW-1, y, 8)
	}
}

type spot struct {
	x, y  float64
	color uint8
	width float64 // the half width of a B-tree node; 0 for a circle
}

// drawTree places the nodes like the page does: the order of the keys for
// x, and the depth for y. It scales the tree into the lane.
func drawTree(img *image.Paletted, root *trees.Node, multi bool, left, top, newest int) {
	var spots []spot
	var edges [][2]int
	order, depthMax := 0.0, 0
	var walk func(n *trees.Node, depth int) (int, float64)
	walk = func(n *trees.Node, depth int) (int, float64) {
		if n == nil {
			return -1, 0
		}
		depthMax = max(depthMax, depth)
		var kids []int
		var centers []float64
		if multi {
			for _, c := range n.Children {
				id, cx := walk(c, depth+1)
				kids, centers = append(kids, id), append(centers, cx)
			}
			x := order + float64(len(n.Keys))/2
			if len(centers) > 0 {
				x = (centers[0] + centers[len(centers)-1]) / 2
			} else {
				order += float64(len(n.Keys)) + 0.6
			}
			c := uint8(6)
			for _, k := range n.Keys {
				if k == newest {
					c = 7
				}
			}
			spots = append(spots, spot{x: x, y: float64(depth), color: c, width: float64(len(n.Keys)) / 2})
		} else {
			l, _ := walk(n.Children[0], depth+1)
			x := order
			order++
			id := len(spots)
			spots = append(spots, spot{x: x, y: float64(depth), color: nodeColor(n, newest)})
			r, _ := walk(n.Children[1], depth+1)
			for _, k := range []int{l, r} {
				if k >= 0 {
					edges = append(edges, [2]int{id, k})
				}
			}
			return id, x
		}
		id := len(spots) - 1
		for _, k := range kids {
			edges = append(edges, [2]int{id, k})
		}
		return id, spots[id].x
	}
	walk(root, 0)

	// Scale into the lane, with a margin, and never bigger than a comfortable size.
	width := order
	if multi {
		width = order - 0.6
	}
	sx := min(22.0, float64(laneW-30)/max(1, width))
	sy := min(40.0, float64(laneH-30)/float64(max(1, depthMax)))
	r := max(2.5, min(8, sx*0.4, sy*0.3))
	px := func(s spot) (int, int) {
		x := float64(left) + 15 + (s.x)*sx
		if !multi {
			x += sx / 2
		}
		return int(x), top + 15 + int(s.y*sy)
	}
	for _, e := range edges {
		x1, y1 := px(spots[e[0]])
		x2, y2 := px(spots[e[1]])
		line(img, x1, y1, x2, y2, 1)
	}
	for _, s := range spots {
		x, y := px(s)
		if multi {
			box(img, x-int(s.width*sx), y-int(r), x+int(s.width*sx), y+int(r), s.color)
		} else {
			disc(img, x, y, r, s.color)
		}
	}
}

func nodeColor(n *trees.Node, newest int) uint8 {
	switch {
	case n.Keys[0] == newest:
		return 7
	case n.Color == "red":
		return 4
	case n.Color == "black":
		return 5
	}
	return 2
}

func line(img *image.Paletted, x1, y1, x2, y2 int, c uint8) {
	dx, dy := abs(x2-x1), -abs(y2-y1)
	sx, sy := sign(x2-x1), sign(y2-y1)
	e := dx + dy
	for {
		img.SetColorIndex(x1, y1, c)
		img.SetColorIndex(x1+1, y1, c)
		if x1 == x2 && y1 == y2 {
			return
		}
		e2 := 2 * e // once, before e changes
		if e2 >= dy {
			e += dy
			x1 += sx
		}
		if e2 <= dx {
			e += dx
			y1 += sy
		}
	}
}

func disc(img *image.Paletted, cx, cy int, r float64, c uint8) {
	ri := int(r + 1)
	for y := -ri; y <= ri; y++ {
		for x := -ri; x <= ri; x++ {
			if d := float64(x*x + y*y); d <= r*r {
				img.SetColorIndex(cx+x, cy+y, c)
			}
		}
	}
}

func box(img *image.Paletted, x1, y1, x2, y2 int, c uint8) {
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			edge := y == y1 || y == y2 || x == x1 || x == x2
			if edge {
				img.SetColorIndex(x, y, 1)
			} else {
				img.SetColorIndex(x, y, c)
			}
		}
	}
}

func abs(v int) int { return max(v, -v) }

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}
