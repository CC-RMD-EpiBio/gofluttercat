package simulation

import (
	"math"
	"math/rand/v2"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
)

// StepRecord captures the state after one CAT item administration.
type StepRecord struct {
	Step     int
	ItemName string
	Response int
	Scale    string
	Energy   []float64 // posterior energy after this step
	Mean     float64
	Std      float64
}

// ReplicateResult holds the outcome of a single simulated CAT session.
type ReplicateResult struct {
	ReplicateID   int
	TrueResponses map[string]int            // item_name -> response value
	TrueMean      map[string]float64        // scale -> true posterior mean
	TrueStd       map[string]float64        // scale -> true posterior std
	TrueEnergy    map[string][]float64      // scale -> true posterior energy
	Steps         []StepRecord
	NItems        int
}

// SimulationSummary aggregates results across Monte Carlo replicates.
type SimulationSummary struct {
	NReplicates int
	MaxItems    int
	Scales      []string
	// Per-scale arrays indexed by step
	MeanL2 map[string][]float64
	StdL2  map[string][]float64
	MeanKL map[string][]float64
	StdKL  map[string][]float64
	MeanSE map[string][]float64
	StdSE  map[string][]float64
	// Raw matrices: [replicate][step], NaN-padded
	L2Matrix   map[string][][]float64
	KLMatrix   map[string][][]float64
	Replicates []ReplicateResult
}

// CATSimulator runs Monte Carlo simulations of adaptive testing sessions.
type CATSimulator struct {
	Models          map[string]*irtcat.GradedResponseModel
	BaselineModels  map[string]*irtcat.GradedResponseModel
	Selector        irtcat.ItemSelector
	ImputationModel imputation.ImputationModel
	MaxItems        int
	AbilityGridPts  []float64
	Prior           func(float64) float64
}

// computeTrueScores scores ALL responses to get the "ground truth" posterior for each scale.
func (sim *CATSimulator) computeTrueScores(responses map[string]int) map[string]*irtcat.BayesianScore {
	scores := make(map[string]*irtcat.BayesianScore)
	for scaleName, model := range sim.Models {
		var baselineModel irtcat.IrtModel
		if bm, ok := sim.BaselineModels[scaleName]; ok {
			baselineModel = *bm
		}
		scorer := irtcat.NewBayesianScorerWithBaseline(
			sim.AbilityGridPts, sim.Prior, model,
			sim.ImputationModel, baselineModel,
		)
		var resps []irtcat.Response
		for _, item := range model.GetItems() {
			if val, ok := responses[item.Name]; ok {
				resps = append(resps, irtcat.Response{Item: item, Value: val})
			}
		}
		if len(resps) > 0 {
			scorer.AddResponses(resps)
		}
		// Deep copy the score so it's independent of the scorer
		scoreCopy := &irtcat.BayesianScore{
			Grid:     scorer.Running.Grid,
			Energy:   make([]float64, len(scorer.Running.Energy)),
			RbEnergy: make([]float64, len(scorer.Running.RbEnergy)),
		}
		copy(scoreCopy.Energy, scorer.Running.Energy)
		copy(scoreCopy.RbEnergy, scorer.Running.RbEnergy)
		scores[scaleName] = scoreCopy
	}
	return scores
}

// RunSingle runs one simulated CAT session at the given true ability.
func (sim *CATSimulator) RunSingle(theta map[string]float64) *ReplicateResult {
	// 1. Sample responses from each scale's GRM
	trueResponses := make(map[string]int)
	for scaleName, model := range sim.Models {
		thetaVal := theta[scaleName]
		abilities, err := ndvek.NewNdArray([]int{1}, []float64{thetaVal})
		if err != nil {
			panic(err)
		}
		samples := model.Sample(abilities) // map[itemName][]int, 0-indexed
		for itemName, sampleVals := range samples {
			if len(sampleVals) == 0 {
				continue
			}
			item := irtcat.GetItemByName(itemName, model.GetItems())
			if item == nil || len(item.ScoredValues) == 0 {
				continue
			}
			idx := sampleVals[0]
			if idx < 0 || idx >= len(item.ScoredValues) {
				idx = len(item.ScoredValues) - 1
			}
			trueResponses[itemName] = item.ScoredValues[idx]
		}
	}

	// 2. Compute true scores by scoring ALL responses
	trueScores := sim.computeTrueScores(trueResponses)

	// 3. Run CAT loop
	scorers := make(map[string]*irtcat.BayesianScorer)
	for scaleName, model := range sim.Models {
		var baselineModel irtcat.IrtModel
		if bm, ok := sim.BaselineModels[scaleName]; ok {
			baselineModel = *bm
		}
		scorer := irtcat.NewBayesianScorerWithBaseline(
			sim.AbilityGridPts, sim.Prior, model,
			sim.ImputationModel, baselineModel,
		)
		scorers[scaleName] = scorer
	}

	// Determine max items for this session
	totalItems := 0
	for _, model := range sim.Models {
		totalItems += len(model.GetItems())
	}
	maxItems := totalItems
	if sim.MaxItems > 0 && sim.MaxItems < maxItems {
		maxItems = sim.MaxItems
	}

	var steps []StepRecord
	for step := 0; step < maxItems; step++ {
		// Find scales with remaining admissible items
		var availableScales []string
		for scaleName := range sim.Models {
			admissible := irtcat.AdmissibleItems(scorers[scaleName])
			if len(admissible) > 0 {
				availableScales = append(availableScales, scaleName)
			}
		}
		if len(availableScales) == 0 {
			break
		}

		// Pick a scale (uniform random; deterministic for single-scale)
		scaleName := availableScales[rand.IntN(len(availableScales))]
		scorer := scorers[scaleName]

		// Select next item
		item := sim.Selector.NextItem(scorer)
		if item == nil {
			continue
		}

		// Look up the sampled response
		response, ok := trueResponses[item.Name]
		if !ok {
			continue
		}

		// Add response to the scale's scorer
		resp := irtcat.Response{Item: item, Value: response}
		scorer.AddResponses([]irtcat.Response{resp})

		// Exclude this item from all other scales' scorers
		for otherScale, otherScorer := range scorers {
			if otherScale != scaleName {
				otherScorer.Exclusions = append(otherScorer.Exclusions, item.Name)
			}
		}

		// Record step
		energyCopy := make([]float64, len(scorer.Running.Energy))
		copy(energyCopy, scorer.Running.Energy)

		steps = append(steps, StepRecord{
			Step:     step,
			ItemName: item.Name,
			Response: response,
			Scale:    scaleName,
			Energy:   energyCopy,
			Mean:     scorer.Running.Mean(),
			Std:      scorer.Running.Std(),
		})
	}

	// Build result
	result := &ReplicateResult{
		TrueResponses: trueResponses,
		TrueMean:      make(map[string]float64),
		TrueStd:       make(map[string]float64),
		TrueEnergy:    make(map[string][]float64),
		Steps:         steps,
		NItems:        len(steps),
	}
	for scaleName, score := range trueScores {
		result.TrueMean[scaleName] = score.Mean()
		result.TrueStd[scaleName] = score.Std()
		energyCopy := make([]float64, len(score.Energy))
		copy(energyCopy, score.Energy)
		result.TrueEnergy[scaleName] = energyCopy
	}

	return result
}

// Simulate runs nReplicates simulated CAT sessions and aggregates metrics.
func (sim *CATSimulator) Simulate(theta map[string]float64, nReplicates int, seed int64) *SimulationSummary {
	replicates := make([]ReplicateResult, nReplicates)

	for r := range nReplicates {
		result := sim.RunSingle(theta)
		result.ReplicateID = r
		replicates[r] = *result
	}

	// Determine scales and max steps
	scales := make([]string, 0, len(sim.Models))
	for scaleName := range sim.Models {
		scales = append(scales, scaleName)
	}

	maxSteps := 0
	for _, rep := range replicates {
		if rep.NItems > maxSteps {
			maxSteps = rep.NItems
		}
	}

	// Build NaN-padded L2 and KL matrices
	l2Matrices := make(map[string][][]float64)
	klMatrices := make(map[string][][]float64)

	for _, scale := range scales {
		l2Mat := makeNaNMatrix(nReplicates, maxSteps)
		klMat := makeNaNMatrix(nReplicates, maxSteps)

		for r, rep := range replicates {
			trueMean, hasTrueScore := rep.TrueMean[scale]
			trueEnergy, hasTrueEnergy := rep.TrueEnergy[scale]
			if !hasTrueScore || !hasTrueEnergy {
				continue
			}

			trueDensity := math2.EnergyToDensity(trueEnergy, sim.AbilityGridPts)

			for _, step := range rep.Steps {
				if step.Scale != scale {
					continue
				}
				idx := step.Step
				if idx >= maxSteps {
					continue
				}
				l2Mat[r][idx] = math.Abs(trueMean - step.Mean)

				stepDensity := math2.EnergyToDensity(step.Energy, sim.AbilityGridPts)
				klMat[r][idx] = math2.KlDivergence(trueDensity, stepDensity, sim.AbilityGridPts)
			}
		}

		l2Matrices[scale] = l2Mat
		klMatrices[scale] = klMat
	}

	// Build SE matrices and compute summary stats
	summary := &SimulationSummary{
		NReplicates: nReplicates,
		MaxItems:    maxSteps,
		Scales:      scales,
		MeanL2:      make(map[string][]float64),
		StdL2:       make(map[string][]float64),
		MeanKL:      make(map[string][]float64),
		StdKL:       make(map[string][]float64),
		MeanSE:      make(map[string][]float64),
		StdSE:       make(map[string][]float64),
		L2Matrix:    l2Matrices,
		KLMatrix:    klMatrices,
		Replicates:  replicates,
	}

	for _, scale := range scales {
		summary.MeanL2[scale] = nanMean(l2Matrices[scale], maxSteps)
		summary.StdL2[scale] = nanStd(l2Matrices[scale], maxSteps)
		summary.MeanKL[scale] = nanMean(klMatrices[scale], maxSteps)
		summary.StdKL[scale] = nanStd(klMatrices[scale], maxSteps)

		// SE matrix from step standard deviations
		seMat := makeNaNMatrix(nReplicates, maxSteps)
		for r, rep := range replicates {
			for _, step := range rep.Steps {
				if step.Scale == scale && step.Step < maxSteps {
					seMat[r][step.Step] = step.Std
				}
			}
		}
		summary.MeanSE[scale] = nanMean(seMat, maxSteps)
		summary.StdSE[scale] = nanStd(seMat, maxSteps)
	}

	return summary
}

// makeNaNMatrix creates a rows x cols matrix filled with NaN.
func makeNaNMatrix(rows, cols int) [][]float64 {
	mat := make([][]float64, rows)
	for r := range rows {
		mat[r] = make([]float64, cols)
		for c := range cols {
			mat[r][c] = math.NaN()
		}
	}
	return mat
}

// nanMean computes the mean of each column, ignoring NaN values.
func nanMean(matrix [][]float64, cols int) []float64 {
	result := make([]float64, cols)
	for j := range cols {
		sum := 0.0
		count := 0
		for i := range matrix {
			v := matrix[i][j]
			if !math.IsNaN(v) {
				sum += v
				count++
			}
		}
		if count > 0 {
			result[j] = sum / float64(count)
		} else {
			result[j] = math.NaN()
		}
	}
	return result
}

// nanStd computes the population standard deviation of each column, ignoring NaN values.
func nanStd(matrix [][]float64, cols int) []float64 {
	means := nanMean(matrix, cols)
	result := make([]float64, cols)
	for j := range cols {
		sumSq := 0.0
		count := 0
		for i := range matrix {
			v := matrix[i][j]
			if !math.IsNaN(v) {
				d := v - means[j]
				sumSq += d * d
				count++
			}
		}
		if count > 0 {
			result[j] = math.Sqrt(sumSq / float64(count))
		} else {
			result[j] = math.NaN()
		}
	}
	return result
}
