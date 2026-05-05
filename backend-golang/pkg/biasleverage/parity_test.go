package biasleverage

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// stubPMFProvider returns precomputed PMFs keyed by "<personIdx>|<target>".
// The current person index is set via SetPerson before each PredictPMF
// call by the parity test driver below.
type stubPMFProvider struct {
	preds  map[string][]float64
	person int
}

func (s *stubPMFProvider) SetPerson(p int) { s.person = p }

func (s *stubPMFProvider) PredictPMF(items map[string]float64, target string, K int) ([]float64, error) {
	key := fmt.Sprintf("%d|%s", s.person, target)
	pmf, ok := s.preds[key]
	if !ok {
		return nil, fmt.Errorf("no fixture prediction for person=%d target=%s",
			s.person, target)
	}
	if len(pmf) != K {
		return nil, fmt.Errorf("fixture pmf for %s has length %d, want K=%d",
			key, len(pmf), K)
	}
	return pmf, nil
}

// indexedTraining wraps a Person with a person index, so the stub
// PMFProvider can return person-specific predictions during Compute.
// Compute itself doesn't expose the person index — the parity test
// instead drives Compute one person at a time, accumulating into
// per-item totals manually. To keep the test direct, we use a small
// adapter Compute below (computeWithProvider) that mirrors what
// production callers will do, but threads the person index through.

// TestPythonParity loads the synthetic fixture produced by
// testdata/generate_fixture.py and verifies B_i, F_i, ratio, n_eff
// match within tolerance.
func TestPythonParity(t *testing.T) {
	fixturePath := filepath.Join("testdata", "leverage_fixture.json")
	expectedPath := filepath.Join("testdata", "leverage_expected.json")

	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("missing testdata/leverage_fixture.json — run testdata/generate_fixture.py")
	}

	type rawItem struct {
		A     float64   `json:"a"`
		D0    float64   `json:"d0"`
		DDiff []float64 `json:"ddiff"`
		K     int       `json:"K"`
	}
	type rawFixture struct {
		K                   int                  `json:"K"`
		ItemKeys            []string             `json:"item_keys"`
		Items               map[string]rawItem   `json:"items"`
		Training            []map[string]float64 `json:"training"`
		ThetaHat            []float64            `json:"theta_hat"`
		ThetaBar            float64              `json:"theta_bar"`
		ProviderPredictions map[string][]float64 `json:"provider_predictions"`
	}
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fix rawFixture
	if err := json.Unmarshal(raw, &fix); err != nil {
		t.Fatal(err)
	}
	type expectedItem struct {
		Item  string  `json:"item"`
		B     float64 `json:"B"`
		F     float64 `json:"F"`
		Ratio float64 `json:"ratio"`
		NEff  int     `json:"n_eff"`
	}
	expRaw, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	var expWrap struct {
		Items []expectedItem `json:"items"`
	}
	if err := json.Unmarshal(expRaw, &expWrap); err != nil {
		t.Fatal(err)
	}

	// Build the GRMItem map.
	itemParams := make(map[string]GRMItem, len(fix.Items))
	for k, v := range fix.Items {
		itemParams[k] = GRMItem{A: v.A, D0: v.D0, DDiff: v.DDiff, K: v.K}
	}
	training := make([]Person, len(fix.Training))
	for i, p := range fix.Training {
		training[i] = Person(p)
	}

	stub := &stubPMFProvider{preds: fix.ProviderPredictions}

	// Compute calls the provider without a person index — drive it per
	// person and accumulate manually so the stub can return the right PMF.
	results := computeAllPersons(itemParams, fix.ItemKeys, training,
		fix.ThetaHat, fix.ThetaBar, stub)

	gotByItem := make(map[string]Result, len(results))
	for _, r := range results {
		gotByItem[r.Item] = r
	}

	const (
		tolB     = 1e-9
		tolF     = 1e-9
		tolRatio = 1e-9
	)
	for _, e := range expWrap.Items {
		got, ok := gotByItem[e.Item]
		if !ok {
			t.Errorf("item %s missing from Go output", e.Item)
			continue
		}
		if got.NEff != e.NEff {
			t.Errorf("%s NEff: go=%d py=%d", e.Item, got.NEff, e.NEff)
		}
		if math.Abs(got.B-e.B) > tolB {
			t.Errorf("%s B: go=%.15g py=%.15g (diff %.2e)",
				e.Item, got.B, e.B, got.B-e.B)
		}
		if math.Abs(got.F-e.F) > tolF {
			t.Errorf("%s F: go=%.15g py=%.15g (diff %.2e)",
				e.Item, got.F, e.F, got.F-e.F)
		}
		if math.Abs(got.Ratio-e.Ratio) > tolRatio {
			t.Errorf("%s ratio: go=%.15g py=%.15g (diff %.2e)",
				e.Item, got.Ratio, e.Ratio, got.Ratio-e.Ratio)
		}
	}
	t.Logf("parity verified for %d items (tol B/F/ratio = %g)",
		len(expWrap.Items), tolB)
}

// computeAllPersons drives Compute one person at a time so the stub
// PMFProvider can vary its prediction by person index. Production
// callers would use Compute directly with a stateless PMFProvider.
func computeAllPersons(itemParams map[string]GRMItem, itemKeys []string,
	training []Person, thetaHat []float64, thetaBar float64,
	stub *stubPMFProvider,
) []Result {
	merged := make([]Result, len(itemKeys))
	sumAbs := make(map[string]float64, len(itemKeys))
	count := make(map[string]int, len(itemKeys))

	for p := range training {
		stub.SetPerson(p)
		single := []Person{training[p]}
		thetaSingle := []float64{thetaHat[p]}
		// Compute over a single-person slice; results' Ratio/F still use
		// thetaBar, B is just the single-person mean (or NaN), NEff is
		// the per-person count we want to sum.
		rs, err := Compute(itemParams, itemKeys, single, thetaSingle, stub, thetaBar)
		if err != nil {
			panic(err)
		}
		for _, r := range rs {
			if r.NEff == 0 || math.IsNaN(r.B) {
				continue
			}
			sumAbs[r.Item] += r.B * float64(r.NEff)
			count[r.Item] += r.NEff
		}
	}
	for i, key := range itemKeys {
		ip := itemParams[key]
		var B float64
		if count[key] > 0 {
			B = sumAbs[key] / float64(count[key])
		} else {
			B = math.NaN()
		}
		F := FisherInfo(thetaBar, ip)
		ratio := math.NaN()
		if F > 1e-6 {
			ratio = B / F
		}
		merged[i] = Result{Item: key, B: B, F: F, Ratio: ratio, NEff: count[key]}
	}
	return merged
}
