package biascorrection

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

const condTol = 1e-9

type condInfoProbe struct {
	Score    float64 `json:"score"`
	Info     float64 `json:"info"`
	Expected float64 `json:"expected"`
}

type condInfoProbes struct {
	Probes []condInfoProbe `json:"probes"`
}

// TestBCMConditionalInfoParity verifies that walking the exported HistGBR
// trees in Go produces the same predictions as libfab's BCMConditionalInfo
// (which delegates to sklearn HistGradientBoostingRegressor.predict).
func TestBCMConditionalInfoParity(t *testing.T) {
	bcm, err := LoadBCMConditionalInfo(
		filepath.Join("testdata", "bcm_conditional_info_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bcm.ScaleName != "synthetic_info" {
		t.Errorf("scale_name = %q, want synthetic_info", bcm.ScaleName)
	}
	if got := bcm.FeatureNames; len(got) != 2 || got[0] != "score" || got[1] != "info" {
		t.Errorf("feature_names = %v, want [score info]", got)
	}

	raw, err := os.ReadFile(
		filepath.Join("testdata", "bcm_conditional_info_predictions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var probes condInfoProbes
	if err := json.Unmarshal(raw, &probes); err != nil {
		t.Fatal(err)
	}
	if len(probes.Probes) == 0 {
		t.Fatal("no probes")
	}

	maxAbs := 0.0
	for _, p := range probes.Probes {
		got := bcm.Apply(p.Score, p.Info)
		diff := math.Abs(got - p.Expected)
		if diff > maxAbs {
			maxAbs = diff
		}
		if diff > condTol {
			t.Errorf("Apply(score=%v info=%v) = %v want %v (|diff|=%v > %v)",
				p.Score, p.Info, got, p.Expected, diff, condTol)
		}
	}
	t.Logf("BCMConditionalInfo: %d probes, max |diff| = %.3e",
		len(probes.Probes), maxAbs)
}

type condProbe struct {
	Indicators []float64 `json:"indicators"`
	Score      float64   `json:"score"`
	Expected   float64   `json:"expected"`
}

type condProbes struct {
	Probes []condProbe `json:"probes"`
}

// TestBCMConditionalParity verifies parity with libfab's BCMConditional
// across a grid of (score, item-indicator-pattern) probes.
func TestBCMConditionalParity(t *testing.T) {
	bcm, err := LoadBCMConditional(
		filepath.Join("testdata", "bcm_conditional_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bcm.ScaleName != "synthetic_cond" {
		t.Errorf("scale_name = %q, want synthetic_cond", bcm.ScaleName)
	}
	if len(bcm.ItemKeys) != 8 {
		t.Fatalf("item_keys length = %d, want 8", len(bcm.ItemKeys))
	}

	raw, err := os.ReadFile(
		filepath.Join("testdata", "bcm_conditional_predictions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var probes condProbes
	if err := json.Unmarshal(raw, &probes); err != nil {
		t.Fatal(err)
	}
	if len(probes.Probes) == 0 {
		t.Fatal("no probes")
	}

	maxAbs := 0.0
	for i, p := range probes.Probes {
		got, err := bcm.Apply(p.Score, p.Indicators)
		if err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
		diff := math.Abs(got - p.Expected)
		if diff > maxAbs {
			maxAbs = diff
		}
		if diff > condTol {
			t.Errorf("probe %d Apply(score=%v indicators=%v) = %v want %v (|diff|=%v > %v)",
				i, p.Score, p.Indicators, got, p.Expected, diff, condTol)
		}
	}
	t.Logf("BCMConditional: %d probes, max |diff| = %.3e",
		len(probes.Probes), maxAbs)
}

// TestBCMConditionalApplyByKey checks that the map-based convenience wrapper
// agrees with the slice-based Apply, since callers in the CAT runtime have
// item membership as a set of keys rather than an aligned vector.
func TestBCMConditionalApplyByKey(t *testing.T) {
	bcm, err := LoadBCMConditional(
		filepath.Join("testdata", "bcm_conditional_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	administered := map[string]bool{
		bcm.ItemKeys[0]: true,
		bcm.ItemKeys[2]: true,
		bcm.ItemKeys[5]: true,
	}
	indicators := make([]float64, len(bcm.ItemKeys))
	for i, k := range bcm.ItemKeys {
		if administered[k] {
			indicators[i] = 1
		}
	}
	for _, s := range []float64{-1.5, 0.0, 1.2} {
		want, err := bcm.Apply(s, indicators)
		if err != nil {
			t.Fatal(err)
		}
		got := bcm.ApplyByKey(s, administered)
		if math.Abs(got-want) > 1e-12 {
			t.Errorf("ApplyByKey(score=%v) = %v want %v", s, got, want)
		}
	}
}
