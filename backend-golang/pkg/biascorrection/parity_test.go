package biascorrection

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestPythonParity loads a BCM Set fitted by sklearn IsotonicRegression
// (testdata/generate_fixture.py) and verifies that Go Apply matches
// sklearn predict to numerical tolerance on a fixed probe set.
//
// The fixtures must be generated first by running:
//
//	cd testdata && uv run --with numpy --with scikit-learn python generate_fixture.py
func TestPythonParity(t *testing.T) {
	fixturePath := filepath.Join("testdata", "bcm_fixture.json")
	predsPath := filepath.Join("testdata", "bcm_predictions.json")

	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("missing testdata/bcm_fixture.json — run testdata/generate_fixture.py")
	}
	if _, err := os.Stat(predsPath); os.IsNotExist(err) {
		t.Skip("missing testdata/bcm_predictions.json — run testdata/generate_fixture.py")
	}

	set, err := LoadSet(fixturePath)
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
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
				t.Errorf("J=%d Apply(%g) = %.15g, sklearn = %.15g (diff %.2e)",
					j, p.Input, got, p.Expected, got-p.Expected)
			}
			checked++
		}
	}
	t.Logf("verified %d Go vs sklearn IsotonicRegression predictions to tol=%g", checked, tol)
}
