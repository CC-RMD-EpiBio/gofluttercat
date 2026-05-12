package biascorrection

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestLibfabParity verifies that a BCM fitted by libfabulouscatpy
// (libfabulouscatpy.biascorrection.fit_bcm_set) and serialized to JSON
// is consumable by Go LoadSet, and that Apply reproduces libfab's own
// predictions to numerical tolerance — closing the loop on the shared
// JSON contract end-to-end.
//
// Generate the fixture with:
//
//	cd testdata && uv run --with numpy --with scikit-learn \
//	    python generate_libfab_fixture.py
func TestLibfabParity(t *testing.T) {
	fixturePath := filepath.Join("testdata", "libfab_bcm_fixture.json")
	predsPath := filepath.Join("testdata", "libfab_bcm_predictions.json")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("missing libfab_bcm_fixture.json — run generate_libfab_fixture.py")
	}
	if _, err := os.Stat(predsPath); os.IsNotExist(err) {
		t.Skip("missing libfab_bcm_predictions.json — run generate_libfab_fixture.py")
	}

	set, err := LoadSet(fixturePath)
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
	}
	if set.Scale != "libfab_synthetic" {
		t.Errorf("set.Scale = %q, want libfab_synthetic", set.Scale)
	}

	predRaw, err := os.ReadFile(predsPath)
	if err != nil {
		t.Fatal(err)
	}
	type probe struct {
		Input    float64 `json:"input"`
		Expected float64 `json:"expected"`
	}
	var preds map[string][]probe
	if err := json.Unmarshal(predRaw, &preds); err != nil {
		t.Fatalf("parse predictions: %v", err)
	}

	const tol = 1e-9
	checked := 0
	for jStr, probes := range preds {
		j, err := strconv.Atoi(jStr)
		if err != nil {
			t.Fatalf("bad probe key %q: %v", jStr, err)
		}
		bcm := set.For(j)
		if bcm == nil || bcm.SubsetSize != j {
			t.Fatalf("set.For(%d) returned nil or wrong size", j)
		}
		for _, p := range probes {
			got := bcm.Apply(p.Input)
			if math.Abs(got-p.Expected) > tol {
				t.Errorf("J=%d Apply(%g) Go=%.15g libfab=%.15g (diff %.2e)",
					j, p.Input, got, p.Expected, got-p.Expected)
			}
			checked++
		}
	}
	t.Logf("verified %d Go vs libfab BCM predictions to tol=%g", checked, tol)
}
