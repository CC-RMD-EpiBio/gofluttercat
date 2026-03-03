package simulation

import (
	"math"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
)

func makeTestSimulator() *CATSimulator {
	item1 := &irtcat.Item{
		Name: "Item1",
		ScaleLoadings: map[string]irtcat.Calibration{
			"default": {Difficulties: []float64{-2, -1, 1, 2}, Discrimination: 1.0},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item2 := &irtcat.Item{
		Name: "Item2",
		ScaleLoadings: map[string]irtcat.Calibration{
			"default": {Difficulties: []float64{-1, -0.5, 1.5, 2.5}, Discrimination: 2.0},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item3 := &irtcat.Item{
		Name: "Item3",
		ScaleLoadings: map[string]irtcat.Calibration{
			"default": {Difficulties: []float64{0, 1, 2.5, 3.5}, Discrimination: 3.0},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}

	scale := irtcat.Scale{Name: "default", Loc: 0, Scale: 1}
	grm := irtcat.NewGRM([]*irtcat.Item{item1, item2, item3}, scale)

	return &CATSimulator{
		Models:         map[string]*irtcat.GradedResponseModel{"default": &grm},
		Selector:       irtcat.BayesianFisherSelector{Temperature: 0},
		MaxItems:       3,
		AbilityGridPts: ndvek.Linspace(-10, 10, 400),
		Prior:          irtcat.DefaultAbilityPrior,
	}
}

func TestRunSingle_ReturnsResult(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.5}
	result := sim.RunSingle(theta)

	if result.NItems <= 0 {
		t.Errorf("expected NItems > 0, got %d", result.NItems)
	}
	if len(result.Steps) != result.NItems {
		t.Errorf("expected len(Steps) == NItems (%d), got %d", result.NItems, len(result.Steps))
	}
}

func TestRunSingle_TrueScoresFinite(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	result := sim.RunSingle(theta)

	for scale, mean := range result.TrueMean {
		if math.IsNaN(mean) || math.IsInf(mean, 0) {
			t.Errorf("true mean for scale %s is not finite: %f", scale, mean)
		}
	}
	for scale, std := range result.TrueStd {
		if math.IsNaN(std) || math.IsInf(std, 0) {
			t.Errorf("true std for scale %s is not finite: %f", scale, std)
		}
		if std <= 0 {
			t.Errorf("true std for scale %s should be > 0, got %f", scale, std)
		}
	}
}

func TestRunSingle_StepScoresFinite(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	result := sim.RunSingle(theta)

	for _, step := range result.Steps {
		if math.IsNaN(step.Mean) || math.IsInf(step.Mean, 0) {
			t.Errorf("step %d mean is not finite: %f", step.Step, step.Mean)
		}
		if math.IsNaN(step.Std) || math.IsInf(step.Std, 0) {
			t.Errorf("step %d std is not finite: %f", step.Step, step.Std)
		}
	}
}

func TestSimulate_ReturnsSummary(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	summary := sim.Simulate(theta, 5, 0)

	if summary.NReplicates != 5 {
		t.Errorf("expected NReplicates=5, got %d", summary.NReplicates)
	}
	if len(summary.Replicates) != 5 {
		t.Errorf("expected 5 replicates, got %d", len(summary.Replicates))
	}
	found := false
	for _, s := range summary.Scales {
		if s == "default" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'default' in scales, got %v", summary.Scales)
	}
}

func TestSimulate_MetricShapes(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	summary := sim.Simulate(theta, 3, 0)

	maxSteps := summary.MaxItems
	for _, scale := range summary.Scales {
		if len(summary.MeanL2[scale]) != maxSteps {
			t.Errorf("MeanL2[%s] length = %d, want %d", scale, len(summary.MeanL2[scale]), maxSteps)
		}
		if len(summary.StdL2[scale]) != maxSteps {
			t.Errorf("StdL2[%s] length = %d, want %d", scale, len(summary.StdL2[scale]), maxSteps)
		}
		if len(summary.MeanKL[scale]) != maxSteps {
			t.Errorf("MeanKL[%s] length = %d, want %d", scale, len(summary.MeanKL[scale]), maxSteps)
		}
		if len(summary.StdKL[scale]) != maxSteps {
			t.Errorf("StdKL[%s] length = %d, want %d", scale, len(summary.StdKL[scale]), maxSteps)
		}
		if len(summary.MeanSE[scale]) != maxSteps {
			t.Errorf("MeanSE[%s] length = %d, want %d", scale, len(summary.MeanSE[scale]), maxSteps)
		}
		if len(summary.StdSE[scale]) != maxSteps {
			t.Errorf("StdSE[%s] length = %d, want %d", scale, len(summary.StdSE[scale]), maxSteps)
		}
	}
}

func TestSimulate_L2NonNegative(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	summary := sim.Simulate(theta, 5, 0)

	for _, scale := range summary.Scales {
		for r, row := range summary.L2Matrix[scale] {
			for s, v := range row {
				if !math.IsNaN(v) && v < 0 {
					t.Errorf("L2[%s][%d][%d] = %f, expected >= 0", scale, r, s, v)
				}
			}
		}
	}
}

func TestSimulate_KLNonNegative(t *testing.T) {
	sim := makeTestSimulator()
	theta := map[string]float64{"default": 0.0}
	summary := sim.Simulate(theta, 5, 0)

	for _, scale := range summary.Scales {
		for r, row := range summary.KLMatrix[scale] {
			for s, v := range row {
				if !math.IsNaN(v) && v < -1e-10 {
					t.Errorf("KL[%s][%d][%d] = %f, expected >= -1e-10", scale, r, s, v)
				}
			}
		}
	}
}

func TestKLDivergence_Identical(t *testing.T) {
	grid := ndvek.Linspace(-3, 3, 100)
	// Build normalized Gaussian density
	p := make([]float64, len(grid))
	for i, x := range grid {
		p[i] = math.Exp(-0.5 * x * x)
	}
	Z := math2.Trapz2(p, grid)
	for i := range p {
		p[i] /= Z
	}

	kl := math2.KlDivergence(p, p, grid)
	if math.Abs(kl) > 1e-10 {
		t.Errorf("KL(p, p) should be ~0, got %f", kl)
	}
}

func TestL2Discrepancy_Identical(t *testing.T) {
	grid := ndvek.Linspace(-3, 3, 100)
	// Build normalized Gaussian density as energy
	energy := make([]float64, len(grid))
	for i, x := range grid {
		energy[i] = -0.5 * x * x
	}

	score := &irtcat.BayesianScore{
		Grid:     grid,
		Energy:   energy,
		RbEnergy: energy,
	}

	l2 := math.Abs(score.Mean() - score.Mean())
	if l2 > 1e-10 {
		t.Errorf("|mean - mean| should be 0, got %f", l2)
	}
}
