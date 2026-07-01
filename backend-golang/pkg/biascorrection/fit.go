package biascorrection

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// This file adds the "fit" side of the Bias-Correction Map. The rest of the
// package (bcm.go) only consumes JSON produced offline by Python
// (sklearn.isotonic.IsotonicRegression). FitBCM/FitSet reproduce that
// isotonic fit natively in Go via the Pool-Adjacent-Violators (PAV)
// algorithm, so a BCM can be trained end-to-end inside gofluttercat from
// (subset_score, gold_score) training pairs — see cmd/bcm.go for a runnable
// example on the grit instrument.

// pav runs the Pool-Adjacent-Violators algorithm and returns the isotonic
// (monotone non-decreasing) least-squares fit of y under unit weights. The
// input is assumed already sorted by the independent variable x. This is the
// same estimator sklearn.isotonic.IsotonicRegression uses by default.
func pav(y []float64) []float64 {
	// Each block tracks its pooled mean, total weight, and how many original
	// points it spans so we can expand back to per-point fitted values.
	vals := make([]float64, 0, len(y))
	weights := make([]float64, 0, len(y))
	counts := make([]int, 0, len(y))

	for _, yi := range y {
		v, w, c := yi, 1.0, 1
		// Merge with the previous block while it violates monotonicity.
		for len(vals) > 0 && vals[len(vals)-1] > v {
			last := len(vals) - 1
			pv, pw, pc := vals[last], weights[last], counts[last]
			vals = vals[:last]
			weights = weights[:last]
			counts = counts[:last]
			v = (pw*pv + w*v) / (pw + w)
			w += pw
			c += pc
		}
		vals = append(vals, v)
		weights = append(weights, w)
		counts = append(counts, c)
	}

	out := make([]float64, 0, len(y))
	for i := range vals {
		for j := 0; j < counts[i]; j++ {
			out = append(out, vals[i])
		}
	}
	return out
}

// FitBCM fits a single isotonic BCM mapping raw subset scores x to gold
// scores y. It sorts by x, runs PAV to obtain a monotone fit, then collapses
// duplicate x values into strictly-increasing breakpoints (averaging their
// fitted y) so the resulting piecewise-linear map is well-formed for Apply.
// Returns an error if fewer than two distinct x values survive.
func FitBCM(x, y []float64, subsetSize int, scale string) (*BCM, error) {
	if len(x) != len(y) {
		return nil, fmt.Errorf("FitBCM: x/y length mismatch %d != %d", len(x), len(y))
	}
	if len(x) < 2 {
		return nil, fmt.Errorf("FitBCM: need at least 2 points, got %d", len(x))
	}

	order := make([]int, len(x))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return x[order[i]] < x[order[j]] })

	xs := make([]float64, len(x))
	ys := make([]float64, len(y))
	for k, i := range order {
		xs[k], ys[k] = x[i], y[i]
	}

	fit := pav(ys)

	// Collapse ties in x (Apply divides by the x-gap, so breakpoints must be
	// strictly increasing) by averaging the isotonic fit within each x group.
	var xt, yt []float64
	for i := 0; i < len(xs); {
		j := i
		var sum float64
		for j < len(xs) && xs[j] == xs[i] {
			sum += fit[j]
			j++
		}
		xt = append(xt, xs[i])
		yt = append(yt, sum/float64(j-i))
		i = j
	}
	if len(xt) < 2 {
		return nil, fmt.Errorf("FitBCM: fewer than 2 distinct x values after collapsing ties")
	}
	// Averaging within groups can only preserve monotonicity, but guard against
	// floating-point drift so Validate is satisfied.
	for k := 1; k < len(yt); k++ {
		if yt[k] < yt[k-1] {
			yt[k] = yt[k-1]
		}
	}

	b := &BCM{
		Scale:       scale,
		XThresholds: xt,
		YThresholds: yt,
		SubsetSize:  subsetSize,
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

// Cell holds the (subset_score, gold_score) training pairs for one subset
// size. SubsetScores and GoldScores must be the same length.
type Cell struct {
	SubsetScores []float64
	GoldScores   []float64
}

// FitSet fits a per-subset-size Set of isotonic BCMs from training cells
// keyed by subset size J. Cells with fewer than two distinct subset scores
// are skipped. Returns an error if no cell yields a usable BCM.
func FitSet(cells map[int]Cell, scale string) (*Set, error) {
	sizes := make([]int, 0, len(cells))
	for j := range cells {
		sizes = append(sizes, j)
	}
	sort.Ints(sizes)

	set := &Set{Scale: scale, Maps: make(map[int]*BCM)}
	for _, j := range sizes {
		c := cells[j]
		bcm, err := FitBCM(c.SubsetScores, c.GoldScores, j, scale)
		if err != nil {
			// Skip degenerate cells rather than failing the whole fit; the
			// caller can inspect which sizes made it into the Set.
			continue
		}
		set.Maps[j] = bcm
	}
	if len(set.Maps) == 0 {
		return nil, fmt.Errorf("FitSet: no subset size produced a valid BCM")
	}
	return set, nil
}

// Save writes the Set to a JSON file in the same shape LoadSet reads.
func (s *Set) Save(path string) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal BCM set: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write BCM set %s: %w", path, err)
	}
	return nil
}
