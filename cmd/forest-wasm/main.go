//go:build js && wasm

// The WebAssembly build for the page. It sets a global object "forest"
// with functions that take and return JSON text:
//
//	forest.kinds()      → [{key, name, summary, restructures, search}]
//	forest.create(json) → an id, from {kind, degree}
//	forest.apply(json)  → {done, steps, tree, stats}, from {id, op, key, record}
//	forest.bench(json)  → a result, from {kind, keys, pattern}
//
// The ops are insert, delete, search, and clear. Every reply is
// {"ok": true, "result": ...} or {"ok": false, "error": "..."}.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"github.com/rahelmartim/tree-forest/trees"
)

type entry struct {
	kind   string
	degree int
	tree   trees.Tree
}

var forest = map[int]*entry{}
var nextID = 1

// maxTrees keeps a page that makes trees in a loop from using all memory.
const maxTrees = 64

func build(kind string, degree int) (trees.Tree, error) {
	if kind == "btree" {
		return trees.NewBTree(degree), nil
	}
	k, ok := trees.FindKind(kind)
	if !ok {
		return nil, fmt.Errorf("unknown tree %q", kind)
	}
	return k.New(), nil
}

func reply(result any, err error) string {
	out := map[string]any{"ok": err == nil}
	if err != nil {
		out["error"] = err.Error()
	} else {
		out["result"] = result
	}
	data, _ := json.Marshal(out)
	return string(data)
}

func handler[T any](fn func(T) (any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		var in T
		if len(args) > 0 {
			if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
				return reply(nil, err)
			}
		}
		return reply(fn(in))
	})
}

type createArgs struct {
	Kind   string `json:"kind"`
	Degree int    `json:"degree"`
}

type applyArgs struct {
	ID     int    `json:"id"`
	Op     string `json:"op"`
	Key    int    `json:"key"`
	Record bool   `json:"record"`
}

type applyResult struct {
	Done  bool         `json:"done"`
	Steps []trees.Step `json:"steps"`
	Tree  *trees.Node  `json:"tree"`
	Stats trees.Stats  `json:"stats"`
}

func create(in createArgs) (any, error) {
	if len(forest) >= maxTrees {
		return nil, errors.New("too many trees")
	}
	t, err := build(in.Kind, in.Degree)
	if err != nil {
		return nil, err
	}
	id := nextID
	nextID++
	forest[id] = &entry{kind: in.Kind, degree: in.Degree, tree: t}
	return id, nil
}

func apply(in applyArgs) (any, error) {
	e, ok := forest[in.ID]
	if !ok {
		return nil, fmt.Errorf("no tree %d", in.ID)
	}
	if in.Key < -999999 || in.Key > 999999 {
		return nil, errors.New("a key must be from -999999 to 999999")
	}
	rec := e.tree.Record(in.Record)
	defer e.tree.Record(false)
	var done bool
	switch in.Op {
	case "insert":
		done = e.tree.Insert(in.Key)
	case "delete":
		done = e.tree.Delete(in.Key)
	case "search":
		done = e.tree.Search(in.Key)
	case "clear":
		t, err := build(e.kind, e.degree)
		if err != nil {
			return nil, err
		}
		e.tree, done = t, true
	default:
		return nil, fmt.Errorf("unknown operation %q", in.Op)
	}
	res := applyResult{Done: done, Steps: []trees.Step{}, Tree: e.tree.Snapshot(), Stats: e.tree.Stats()}
	if rec != nil {
		res.Steps = rec.Steps
	}
	return res, nil
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("kinds", js.FuncOf(func(js.Value, []js.Value) any { return reply(trees.Kinds, nil) }))
	api.Set("create", handler(create))
	api.Set("apply", handler(apply))
	api.Set("bench", handler(func(in struct {
		Kind    string `json:"kind"`
		Keys    int    `json:"keys"`
		Pattern string `json:"pattern"`
	}) (any, error) {
		return trees.Bench(in.Kind, in.Keys, in.Pattern, 1)
	}))
	js.Global().Set("forest", api)
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("forest-ready"))
	select {}
}
