package trees

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// Patterns of keys for a benchmark.
var Patterns = []string{"random", "sorted", "reversed", "zigzag"}

// MaxBenchKeys keeps a benchmark short. A plain BST of sorted keys takes
// n² steps, so its limit is lower.
const (
	MaxBenchKeys    = 200000
	MaxBenchBSTKeys = 5000
)

// BenchResult is the time of each operation, in nanoseconds for one key.
type BenchResult struct {
	Kind         string  `json:"kind"`
	Name         string  `json:"name"`
	Keys         int     `json:"keys"`
	InsertNs     float64 `json:"insertNs"`
	SearchNs     float64 `json:"searchNs"`
	DeleteNs     float64 `json:"deleteNs"`
	Height       int     `json:"height"`       // after all the inserts
	Restructures int     `json:"restructures"` // during the whole run
	Comparisons  float64 `json:"comparisons"`  // in one search, on average
}

// Order returns n keys in a pattern.
func Order(pattern string, n int, seed uint64) ([]int, error) {
	keys := make([]int, n)
	for i := range keys {
		keys[i] = i
	}
	switch pattern {
	case "random":
		rng := rand.New(rand.NewPCG(seed, seed+1))
		rng.Shuffle(n, func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	case "sorted":
	case "reversed":
		for i := range keys {
			keys[i] = n - 1 - i
		}
	case "zigzag": // 0, n-1, 1, n-2, ...
		for i := range keys {
			if i%2 == 0 {
				keys[i] = i / 2
			} else {
				keys[i] = n - 1 - i/2
			}
		}
	default:
		return nil, fmt.Errorf("unknown pattern %q", pattern)
	}
	return keys, nil
}

// Bench inserts n keys in a pattern, searches all of them, and deletes all
// of them, in the same pattern.
func Bench(kind string, n int, pattern string, seed uint64) (BenchResult, error) {
	k, ok := FindKind(kind)
	if !ok {
		return BenchResult{}, fmt.Errorf("unknown tree %q", kind)
	}
	limit := MaxBenchKeys
	if kind == "bst" && pattern != "random" {
		limit = MaxBenchBSTKeys
	}
	n = max(1, min(limit, n))
	keys, err := Order(pattern, n, seed)
	if err != nil {
		return BenchResult{}, err
	}
	tree := k.New()
	res := BenchResult{Kind: k.Key, Name: k.Name, Keys: n}
	per := func(d time.Duration) float64 { return float64(d.Nanoseconds()) / float64(n) }

	start := time.Now()
	for _, key := range keys {
		tree.Insert(key)
	}
	res.InsertNs = per(time.Since(start))
	res.Height = tree.Stats().Height

	comparisons := 0
	start = time.Now()
	for _, key := range keys {
		tree.Search(key)
		comparisons += tree.Stats().Comparisons
	}
	res.SearchNs = per(time.Since(start))
	res.Comparisons = float64(comparisons) / float64(n)

	start = time.Now()
	for _, key := range keys {
		tree.Delete(key)
	}
	res.DeleteNs = per(time.Since(start))
	res.Restructures = tree.Stats().Restructures
	if tree.Stats().Size != 0 {
		return res, fmt.Errorf("%s kept %d keys after the deletes", k.Name, tree.Stats().Size)
	}
	return res, nil
}
