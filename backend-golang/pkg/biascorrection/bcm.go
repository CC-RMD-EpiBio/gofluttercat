// Package biascorrection implements the Bias-Correction Map (BCM) post-hoc
// correction described in the cat_optimalcontrol manuscript.
//
// A BCM is an isotonic (monotone non-decreasing) regression fit on
// pairs of (subset_score, gold_standard_score), one per (scale, subset_size)
// cell. At scoring time, the raw subset score is mapped through the BCM
// to obtain a corrected score with mean bias driven to zero by construction.
//
// The fit itself happens offline in Python (sklearn.isotonic.IsotonicRegression)
// and is serialized as JSON. This package only consumes the JSON and applies
// the mapping: piecewise-linear interpolation between adjacent (x, y)
// thresholds, with clipping at the ends.
package biascorrection

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// BCM is a single fitted isotonic mapping. XThresholds and YThresholds are
// monotone-non-decreasing breakpoints defining a piecewise-linear function.
// Both slices must have the same length and at least two points.
type BCM struct {
	// Scale is the IRT scale this BCM was fit on (e.g. "scs", "wpi").
	Scale string `json:"scale"`
	// SubsetSize is the number of administered items J this BCM covers.
	SubsetSize int `json:"subset_size"`
	// XThresholds are sorted raw subset-score breakpoints.
	XThresholds []float64 `json:"x_thresholds"`
	// YThresholds are corresponding fitted gold-standard scores
	// (monotone non-decreasing, same length as XThresholds).
	YThresholds []float64 `json:"y_thresholds"`
}

// Apply maps a raw subset score to the bias-corrected score by linear
// interpolation between adjacent thresholds. Inputs outside
// [XThresholds[0], XThresholds[n-1]] are clipped to the end values.
func (b *BCM) Apply(rawScore float64) float64 {
	n := len(b.XThresholds)
	if rawScore <= b.XThresholds[0] {
		return b.YThresholds[0]
	}
	if rawScore >= b.XThresholds[n-1] {
		return b.YThresholds[n-1]
	}
	// Binary search for the upper bracket.
	hi := sort.SearchFloat64s(b.XThresholds, rawScore)
	if hi >= n {
		return b.YThresholds[n-1]
	}
	if b.XThresholds[hi] == rawScore {
		return b.YThresholds[hi]
	}
	lo := hi - 1
	xlo, xhi := b.XThresholds[lo], b.XThresholds[hi]
	ylo, yhi := b.YThresholds[lo], b.YThresholds[hi]
	t := (rawScore - xlo) / (xhi - xlo)
	return ylo + t*(yhi-ylo)
}

// Validate checks that the BCM thresholds are well-formed. Returns nil
// if OK, otherwise a descriptive error.
func (b *BCM) Validate() error {
	if len(b.XThresholds) != len(b.YThresholds) {
		return fmt.Errorf("BCM threshold length mismatch: x=%d y=%d",
			len(b.XThresholds), len(b.YThresholds))
	}
	if len(b.XThresholds) < 2 {
		return fmt.Errorf("BCM needs at least 2 threshold points, got %d",
			len(b.XThresholds))
	}
	for i := 1; i < len(b.XThresholds); i++ {
		if b.XThresholds[i] < b.XThresholds[i-1] {
			return fmt.Errorf("BCM XThresholds not sorted at index %d", i)
		}
		if b.YThresholds[i] < b.YThresholds[i-1] {
			return fmt.Errorf("BCM YThresholds not monotone at index %d", i)
		}
	}
	return nil
}

// Set is a collection of BCMs keyed by subset size, intended for one IRT
// scale. Use For to retrieve the BCM that matches the number of
// administered items, or the closest available subset size.
type Set struct {
	Scale string       `json:"scale"`
	Maps  map[int]*BCM `json:"maps"` // key: subset size J
}

// For returns the BCM whose SubsetSize equals administered. If no exact
// match exists, the closest available SubsetSize is returned. If the Set
// is empty, returns nil.
func (s *Set) For(administered int) *BCM {
	if len(s.Maps) == 0 {
		return nil
	}
	if bcm, ok := s.Maps[administered]; ok {
		return bcm
	}
	bestKey := -1
	bestDist := -1
	for k := range s.Maps {
		d := administered - k
		if d < 0 {
			d = -d
		}
		if bestKey < 0 || d < bestDist {
			bestKey = k
			bestDist = d
		}
	}
	return s.Maps[bestKey]
}

// LoadSet reads a Set from a JSON file. Expected shape:
//
//	{
//	  "scale": "scs",
//	  "maps": {
//	    "5":  {"subset_size": 5,  "x_thresholds": [...], "y_thresholds": [...]},
//	    "10": {"subset_size": 10, "x_thresholds": [...], "y_thresholds": [...]}
//	  }
//	}
//
// Each map's Scale field is set to the parent Scale on load.
func LoadSet(path string) (*Set, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read BCM set %s: %w", path, err)
	}
	// Parse with string-keyed map first, then convert to int keys.
	var stub struct {
		Scale string          `json:"scale"`
		Maps  map[string]*BCM `json:"maps"`
	}
	if err := json.Unmarshal(raw, &stub); err != nil {
		return nil, fmt.Errorf("parse BCM set %s: %w", path, err)
	}
	out := &Set{Scale: stub.Scale, Maps: make(map[int]*BCM, len(stub.Maps))}
	for k, bcm := range stub.Maps {
		var subsetSize int
		if _, err := fmt.Sscanf(k, "%d", &subsetSize); err != nil {
			return nil, fmt.Errorf("BCM set key %q is not an integer", k)
		}
		bcm.Scale = stub.Scale
		if bcm.SubsetSize == 0 {
			bcm.SubsetSize = subsetSize
		}
		if err := bcm.Validate(); err != nil {
			return nil, fmt.Errorf("BCM map[%d]: %w", subsetSize, err)
		}
		out.Maps[subsetSize] = bcm
	}
	return out, nil
}

// LoadBCM reads a single BCM (no subset-size selection) from a JSON file.
// Expected shape: {"scale": "...", "subset_size": N, "x_thresholds": [...],
// "y_thresholds": [...]}.
func LoadBCM(path string) (*BCM, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read BCM %s: %w", path, err)
	}
	bcm := &BCM{}
	if err := json.Unmarshal(raw, bcm); err != nil {
		return nil, fmt.Errorf("parse BCM %s: %w", path, err)
	}
	if err := bcm.Validate(); err != nil {
		return nil, err
	}
	return bcm, nil
}
