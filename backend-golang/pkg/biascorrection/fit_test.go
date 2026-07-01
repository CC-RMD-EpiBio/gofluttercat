package biascorrection

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPavMonotone(t *testing.T) {
	// Classic PAV example: input has a dip that must be pooled.
	in := []float64{1, 2, 4, 3, 5}
	got := pav(in)
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1]-1e-12 {
			t.Fatalf("pav output not monotone at %d: %v", i, got)
		}
	}
	// PAV preserves the sum (it is a least-squares projection with unit weights).
	var sumIn, sumOut float64
	for i := range in {
		sumIn += in[i]
		sumOut += got[i]
	}
	if math.Abs(sumIn-sumOut) > 1e-9 {
		t.Errorf("pav changed the total: in=%g out=%g", sumIn, sumOut)
	}
	// The pooled block {4,3} averages to 3.5.
	if math.Abs(got[2]-3.5) > 1e-9 || math.Abs(got[3]-3.5) > 1e-9 {
		t.Errorf("expected pooled 3.5 at idx 2,3, got %v", got)
	}
}

func TestFitBCMRecoversMonotoneRelationship(t *testing.T) {
	// gold = subset + fixed bias; a well-formed BCM should map subset back
	// onto gold to within interpolation error.
	var xs, ys []float64
	for i := 0; i < 50; i++ {
		x := -2.0 + 0.08*float64(i)
		xs = append(xs, x)
		ys = append(ys, x+0.5) // constant upward bias
	}
	bcm, err := FitBCM(xs, ys, 5, "synthetic")
	if err != nil {
		t.Fatalf("FitBCM: %v", err)
	}
	if err := bcm.Validate(); err != nil {
		t.Fatalf("fitted BCM invalid: %v", err)
	}
	// Apply at an interior point should recover subset+0.5.
	got := bcm.Apply(0.0)
	if math.Abs(got-0.5) > 1e-6 {
		t.Errorf("Apply(0) = %g, want ~0.5", got)
	}
}

func TestFitBCMHandlesTiesAndNoise(t *testing.T) {
	// Duplicate x values with noisy y must collapse to strictly increasing
	// breakpoints and stay monotone.
	xs := []float64{0, 0, 0, 1, 1, 2, 2, 2}
	ys := []float64{0.1, -0.2, 0.3, 1.2, 0.9, 2.5, 1.8, 2.2}
	bcm, err := FitBCM(xs, ys, 3, "s")
	if err != nil {
		t.Fatalf("FitBCM: %v", err)
	}
	for i := 1; i < len(bcm.XThresholds); i++ {
		if bcm.XThresholds[i] <= bcm.XThresholds[i-1] {
			t.Errorf("XThresholds not strictly increasing: %v", bcm.XThresholds)
		}
	}
	if err := bcm.Validate(); err != nil {
		t.Errorf("BCM with ties failed Validate: %v", err)
	}
}

func TestFitSetSaveRoundtrip(t *testing.T) {
	cells := map[int]Cell{
		5: {
			SubsetScores: []float64{-2, -1, 0, 1, 2},
			GoldScores:   []float64{-1.5, -0.5, 0.2, 0.9, 1.8},
		},
		10: {
			SubsetScores: []float64{-2, -1, 0, 1, 2},
			GoldScores:   []float64{-1.8, -0.9, 0.0, 0.9, 1.9},
		},
	}
	set, err := FitSet(cells, "scs")
	if err != nil {
		t.Fatalf("FitSet: %v", err)
	}
	if len(set.Maps) != 2 {
		t.Fatalf("expected 2 maps, got %d", len(set.Maps))
	}

	path := filepath.Join(t.TempDir(), "bcm.json")
	if err := set.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Round-trip through the consumer path.
	loaded, err := LoadSet(path)
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
	}
	if loaded.Scale != "scs" || len(loaded.Maps) != 2 {
		t.Fatalf("roundtrip mismatch: scale=%q maps=%d", loaded.Scale, len(loaded.Maps))
	}
	before := set.For(5).Apply(0.3)
	after := loaded.For(5).Apply(0.3)
	if math.Abs(before-after) > 1e-12 {
		t.Errorf("Apply differs after roundtrip: %g vs %g", before, after)
	}
	_ = os.Remove(path)
}
