/*
###############################################################################
# Regression test for GradedResponseModel.LogLikelihood.
#
# In May 2026 we discovered that the Python sibling implementation
# (libfabulouscatpy/irt/prediction/grm.py) had a bug in its batch
# log_likelihood path: it summed an N x N cross-indexed matrix
# (sum_i sum_j log P(y_j | item i, theta)) instead of the diagonal
# (sum_i log P(y_i | item i, theta)).  The Go implementation here was
# audited and found to be correct -- it iterates one response at a time,
# looks up prob[item_r][ability_i, category_ndx], and accumulates the log.
# This test pins that behaviour so a future refactor cannot regress to
# the cross-indexed bug.
###############################################################################
*/

package tests

import (
	"math"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/mederrata/ndvek"
)

func makeItem(name string, difficulties []float64, disc float64) irtcat.Item {
	return irtcat.Item{
		Name: name,
		ScaleLoadings: map[string]irtcat.Calibration{
			"default": {
				Difficulties:   difficulties,
				Discrimination: disc,
			},
		},
		ScoredValues: []int{1, 2, 3, 4},
	}
}

func TestLogLikelihoodBatchEqualsSumOfSingleItem(t *testing.T) {
	// 3 items with distinct calibrations + responses
	item1 := makeItem("item_A", []float64{-1.0, 0.0, 1.0}, 1.5)
	item2 := makeItem("item_B", []float64{-0.5, 0.2, 0.8}, 0.8)
	item3 := makeItem("item_C", []float64{-1.5, -0.2, 0.6}, 1.2)
	scale := irtcat.Scale{Loc: 0, Scale: 1, Name: "default"}
	grm := irtcat.NewGRM([]*irtcat.Item{&item1, &item2, &item3}, scale)

	abilities, err := ndvek.NewNdArray([]int{5}, []float64{-2, -1, 0, 1, 2})
	if err != nil {
		t.Fatalf("NewNdArray: %v", err)
	}

	resp1 := irtcat.Response{Value: 2, Item: &item1}
	resp2 := irtcat.Response{Value: 3, Item: &item2}
	resp3 := irtcat.Response{Value: 1, Item: &item3}

	// Batch call
	llBatch := grm.LogLikelihood(abilities, []irtcat.Response{resp1, resp2, resp3})

	// Sum of single-item calls
	llS1 := grm.LogLikelihood(abilities, []irtcat.Response{resp1})
	llS2 := grm.LogLikelihood(abilities, []irtcat.Response{resp2})
	llS3 := grm.LogLikelihood(abilities, []irtcat.Response{resp3})

	n := abilities.Shape()[0]
	const tol = 1e-12
	for i := 0; i < n; i++ {
		batch, _ := llBatch.Get([]int{i})
		s1, _ := llS1.Get([]int{i})
		s2, _ := llS2.Get([]int{i})
		s3, _ := llS3.Get([]int{i})
		sum := s1 + s2 + s3
		if math.Abs(batch-sum) > tol {
			t.Errorf("ability index %d: batch=%.12g vs single-sum=%.12g (diff %.2e)",
				i, batch, sum, batch-sum)
		}
	}
}

func TestLogLikelihoodMatchesDiagonalSumNotCrossSum(t *testing.T) {
	// Explicit test: verify the result is sum_i log P(y_i | item i, theta),
	// not sum_i sum_j log P(y_j | item i, theta).  With 3 distinct items
	// and distinct response values these two formulas give very different
	// numbers.
	item1 := makeItem("item_A", []float64{-1.0, 0.0, 1.0}, 1.5)
	item2 := makeItem("item_B", []float64{-0.5, 0.2, 0.8}, 0.8)
	item3 := makeItem("item_C", []float64{-1.5, -0.2, 0.6}, 1.2)
	scale := irtcat.Scale{Loc: 0, Scale: 1, Name: "default"}
	grm := irtcat.NewGRM([]*irtcat.Item{&item1, &item2, &item3}, scale)

	abilities, err := ndvek.NewNdArray([]int{5}, []float64{-2, -1, 0, 1, 2})
	if err != nil {
		t.Fatalf("NewNdArray: %v", err)
	}

	resp1 := irtcat.Response{Value: 2, Item: &item1}
	resp2 := irtcat.Response{Value: 3, Item: &item2}
	resp3 := irtcat.Response{Value: 1, Item: &item3}
	responses := []irtcat.Response{resp1, resp2, resp3}

	llBatch := grm.LogLikelihood(abilities, responses)

	// Manual diagonal sum: for each ability, sum_r log P(y_r | item_r)
	probs := grm.Prob(abilities)
	items := []*irtcat.Item{&item1, &item2, &item3}
	values := []int{2, 3, 1}
	n := abilities.Shape()[0]
	const tol = 1e-12
	for i := 0; i < n; i++ {
		var diag float64
		for r := range items {
			ndx := values[r] - 1
			p, err := probs[items[r].Name].Get([]int{i, ndx})
			if err != nil {
				t.Fatalf("probs Get: %v", err)
			}
			diag += math.Log(p)
		}
		batch, _ := llBatch.Get([]int{i})
		if math.Abs(batch-diag) > tol {
			t.Errorf("ability index %d: batch=%.12g vs manual diagonal=%.12g (diff %.2e)",
				i, batch, diag, batch-diag)
		}
	}
}
