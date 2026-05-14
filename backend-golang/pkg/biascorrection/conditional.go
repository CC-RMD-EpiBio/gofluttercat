package biascorrection

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// histNode is one node in a HistGradientBoostingRegressor predictor tree
// (sklearn's TreePredictor). Inference walks the binary tree comparing
// the raw feature value to NumThreshold; the categorical path is not
// supported by this port, mirroring the constraint enforced at fixture
// generation time.
type histNode struct {
	Value           float64 `json:"value"`
	NumThreshold    float64 `json:"num_threshold"`
	FeatureIdx      int     `json:"feature_idx"`
	Left            uint32  `json:"left"`
	Right           uint32  `json:"right"`
	IsLeaf          bool    `json:"is_leaf"`
	MissingGoToLeft bool    `json:"missing_go_to_left"`
}

// histTree is the per-iteration regression tree exported from sklearn.
type histTree []histNode

// predict walks the tree on x and returns the leaf value.
func (t histTree) predict(x []float64) float64 {
	idx := uint32(0)
	for {
		n := t[idx]
		if n.IsLeaf {
			return n.Value
		}
		v := x[n.FeatureIdx]
		if math.IsNaN(v) {
			if n.MissingGoToLeft {
				idx = n.Left
			} else {
				idx = n.Right
			}
			continue
		}
		if v <= n.NumThreshold {
			idx = n.Left
		} else {
			idx = n.Right
		}
	}
}

// histGBR is a fitted sklearn HistGradientBoostingRegressor with a
// squared-error loss (identity inverse link), serialized for the Go
// runtime. Single-target regression only.
type histGBR struct {
	Trees              []histTree `json:"trees"`
	BaselinePrediction float64    `json:"baseline_prediction"`
	NFeatures          int        `json:"n_features"`
}

func (m *histGBR) predict(x []float64) float64 {
	if len(x) != m.NFeatures {
		// Caller is expected to size the feature vector correctly; panic
		// here would surface a programmer error loudly during tests.
		return math.NaN()
	}
	y := m.BaselinePrediction
	for i := range m.Trees {
		y += m.Trees[i].predict(x)
	}
	return y
}

func (m *histGBR) validate() error {
	if m.NFeatures <= 0 {
		return fmt.Errorf("histGBR: n_features must be positive, got %d", m.NFeatures)
	}
	if len(m.Trees) == 0 {
		return fmt.Errorf("histGBR: no trees")
	}
	for ti, tr := range m.Trees {
		if len(tr) == 0 {
			return fmt.Errorf("histGBR: tree %d is empty", ti)
		}
		n := uint32(len(tr))
		for ni := range tr {
			node := tr[ni]
			if node.IsLeaf {
				continue
			}
			if node.Left >= n || node.Right >= n {
				return fmt.Errorf("histGBR: tree %d node %d has child index out of range", ti, ni)
			}
		}
	}
	return nil
}

// BCMConditionalInfo is the Go-runtime port of libfab's bivariate BCM
// that conditions on subset score (monotone) and subset Fisher
// information at theta=0 (unconstrained). It applies the
// HistGradientBoostingRegressor trained in Python by walking the
// exported trees on raw feature values; sklearn's prediction with
// squared-error loss is identity, so Predict and sklearn.predict agree
// to within floating-point error.
type BCMConditionalInfo struct {
	ScaleName    string   `json:"scale_name"`
	FeatureNames []string `json:"feature_names"`
	Model        histGBR  `json:"model"`
}

// Apply returns the bias-corrected score for (rawScore, info).
func (b *BCMConditionalInfo) Apply(rawScore, info float64) float64 {
	return b.Model.predict([]float64{rawScore, info})
}

// Validate sanity-checks the loaded model.
func (b *BCMConditionalInfo) Validate() error {
	if b.Model.NFeatures != 2 {
		return fmt.Errorf("BCMConditionalInfo: expected 2 features, got %d", b.Model.NFeatures)
	}
	return b.Model.validate()
}

// LoadBCMConditionalInfo reads a JSON-serialized BCMConditionalInfo
// produced by testdata/generate_conditional_fixture.py.
func LoadBCMConditionalInfo(path string) (*BCMConditionalInfo, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read BCMConditionalInfo %s: %w", path, err)
	}
	b := &BCMConditionalInfo{}
	if err := json.Unmarshal(raw, b); err != nil {
		return nil, fmt.Errorf("parse BCMConditionalInfo %s: %w", path, err)
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

// BCMConditional is the per-item-indicator BCM. Features are
// [score, indicator_0, ..., indicator_{I-1}]; column order matches
// ItemKeys.
type BCMConditional struct {
	ScaleName string   `json:"scale_name"`
	ItemKeys  []string `json:"item_keys"`
	Model     histGBR  `json:"model"`
}

// Apply returns the bias-corrected score for (rawScore, indicators).
// indicators must have len(b.ItemKeys) entries, in ItemKeys order.
func (b *BCMConditional) Apply(rawScore float64, indicators []float64) (float64, error) {
	if len(indicators) != len(b.ItemKeys) {
		return 0, fmt.Errorf("BCMConditional: indicators length %d != item_keys length %d",
			len(indicators), len(b.ItemKeys))
	}
	x := make([]float64, 1+len(indicators))
	x[0] = rawScore
	copy(x[1:], indicators)
	return b.Model.predict(x), nil
}

// ApplyByKey is a convenience wrapper that builds the indicator vector
// from a map of item-key -> {0,1}. Unknown keys in the map are ignored;
// missing keys default to zero.
func (b *BCMConditional) ApplyByKey(rawScore float64, administered map[string]bool) float64 {
	indicators := make([]float64, len(b.ItemKeys))
	for i, k := range b.ItemKeys {
		if administered[k] {
			indicators[i] = 1
		}
	}
	y, _ := b.Apply(rawScore, indicators)
	return y
}

// Validate sanity-checks the loaded model.
func (b *BCMConditional) Validate() error {
	if b.Model.NFeatures != 1+len(b.ItemKeys) {
		return fmt.Errorf("BCMConditional: model n_features=%d but item_keys=%d (expected %d)",
			b.Model.NFeatures, len(b.ItemKeys), 1+len(b.ItemKeys))
	}
	return b.Model.validate()
}

// LoadBCMConditional reads a JSON-serialized BCMConditional produced by
// testdata/generate_conditional_fixture.py.
func LoadBCMConditional(path string) (*BCMConditional, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read BCMConditional %s: %w", path, err)
	}
	b := &BCMConditional{}
	if err := json.Unmarshal(raw, b); err != nil {
		return nil, fmt.Errorf("parse BCMConditional %s: %w", path, err)
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}
