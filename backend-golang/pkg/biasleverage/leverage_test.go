package biasleverage

import (
	"math"
	"testing"
)

func TestGRMPMFSumsToOne(t *testing.T) {
	item := GRMItem{A: 1.2, D0: -0.5, DDiff: []float64{0.6, 0.8}, K: 4}
	for _, theta := range []float64{-3, -1, 0, 1, 3} {
		pmf := GRMPMF(theta, item)
		var sum float64
		for _, p := range pmf {
			if p < 0 {
				t.Errorf("negative pmf entry at theta=%g: %v", theta, pmf)
			}
			sum += p
		}
		if math.Abs(sum-1.0) > 1e-12 {
			t.Errorf("pmf at theta=%g sums to %g, want 1", theta, sum)
		}
	}
}

func TestGRMPMFBinaryEdgeCase(t *testing.T) {
	item := GRMItem{A: 1.0, D0: 0.0, DDiff: nil, K: 2}
	pmf := GRMPMF(0.0, item)
	if math.Abs(pmf[0]-0.5) > 1e-12 || math.Abs(pmf[1]-0.5) > 1e-12 {
		t.Errorf("binary GRM at theta=0 d0=0: got %v, want [0.5, 0.5]", pmf)
	}
}

func TestFisherInfoNonNegative(t *testing.T) {
	item := GRMItem{A: 1.5, D0: -0.3, DDiff: []float64{0.5, 0.7}, K: 4}
	for theta := -3.0; theta <= 3.0; theta += 0.5 {
		fi := FisherInfo(theta, item)
		if fi < 0 {
			t.Errorf("Fisher info should be non-negative, got %g at theta=%g", fi, theta)
		}
	}
}

// constPMFProvider always returns a uniform PMF; B_i is then the mean
// |log(1/K) - log p_IRT(y_obs|theta)| across observations.
type constPMFProvider struct{ pmf []float64 }

func (c constPMFProvider) PredictPMF(items map[string]float64, target string, K int) ([]float64, error) {
	if len(c.pmf) != K {
		out := make([]float64, K)
		u := 1.0 / float64(K)
		for i := range out {
			out[i] = u
		}
		return out, nil
	}
	return c.pmf, nil
}

func TestComputeWithUniformProvider(t *testing.T) {
	K := 3
	itemKeys := []string{"a", "b"}
	items := map[string]GRMItem{
		"a": {A: 1.0, D0: -0.5, DDiff: []float64{1.0}, K: K},
		"b": {A: 1.0, D0: 0.5, DDiff: []float64{1.0}, K: K},
	}
	training := []Person{
		{"a": 0, "b": 1},
		{"a": 1, "b": 2},
		{"a": 2, "b": 0},
	}
	thetaHat := []float64{-1.0, 0.0, 1.0}
	results, err := Compute(items, itemKeys, training, thetaHat,
		constPMFProvider{}, 0.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.NEff != 3 {
			t.Errorf("%s: NEff = %d, want 3", r.Item, r.NEff)
		}
		if r.B < 0 {
			t.Errorf("%s: B = %g, want non-negative", r.Item, r.B)
		}
		if r.F <= 0 {
			t.Errorf("%s: F = %g, want > 0", r.Item, r.F)
		}
	}
}

func TestComputeRejectsLengthMismatch(t *testing.T) {
	_, err := Compute(map[string]GRMItem{}, []string{},
		[]Person{{}}, []float64{0, 0}, constPMFProvider{}, 0)
	if err == nil {
		t.Error("expected error on len(thetaHat) != len(training)")
	}
}

func TestComputeSortsByRatioDesc(t *testing.T) {
	K := 3
	itemKeys := []string{"low_lev", "high_lev"}
	items := map[string]GRMItem{
		"low_lev":  {A: 2.0, D0: 0.0, DDiff: []float64{0.5}, K: K}, // high F
		"high_lev": {A: 0.5, D0: 0.0, DDiff: []float64{0.5}, K: K}, // low F
	}
	training := []Person{
		{"low_lev": 0, "high_lev": 2},
		{"low_lev": 1, "high_lev": 0},
		{"low_lev": 2, "high_lev": 1},
	}
	thetaHat := []float64{-1, 0, 1}
	results, _ := Compute(items, itemKeys, training, thetaHat, constPMFProvider{}, 0.0)
	// "high_lev" has small F so larger Ratio = B/F → should sort first.
	if results[0].Item != "high_lev" {
		t.Errorf("expected high_lev first by ratio desc, got %v",
			[]string{results[0].Item, results[1].Item})
	}
}
