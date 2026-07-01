package cmd

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/biascorrection"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web"
	"github.com/mederrata/ndvek"
	"github.com/spf13/cobra"
)

// BcmCmd fits and evaluates a Bias-Correction Map for a single instrument
// scale, entirely in Go. It mirrors the offline Python pipeline
// (bayesianquilts/examples/irt/fit_bcm_with_imputation.py): simulate CAT
// sessions to build (subset_score, gold_score) triples, fit a per-subset-size
// isotonic BCM, then report how much the BCM shrinks the subset->gold bias on
// held-out sessions.
var BcmCmd = &cobra.Command{
	Use:   "bcm",
	Short: "Fit and evaluate a Bias-Correction Map (BCM) for an instrument",
	Long: `Fit a Bias-Correction Map (BCM) for one IRT instrument scale.

A CAT session that stops after J items yields a subset ability estimate (EAP)
that is biased relative to the full-bank "gold" estimate. The BCM is an
isotonic mapping subset_score -> gold_score, fit per subset size, that drives
that mean bias to zero.

This command simulates sessions across a grid of true abilities, splits them
into train/test halves, fits the BCM on the training half (see the
biascorrection package's Go-native isotonic fitter), and reports the held-out
L2 bias reduction. Use --out to save the fitted Set as JSON consumable by
biascorrection.LoadSet.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		instrument, _ := cmd.Flags().GetString("instrument")
		scaleFlag, _ := cmd.Flags().GetString("scale")
		replicates, _ := cmd.Flags().GetInt("replicates")
		subsetSizes, _ := cmd.Flags().GetIntSlice("subset-sizes")
		useImputation, _ := cmd.Flags().GetBool("imputation")
		outPath, _ := cmd.Flags().GetString("out")

		config := conf.GetConfig()
		instruments := web.LoadAllInstruments(config)

		reg, ok := instruments[instrument]
		if !ok {
			available := make([]string, 0, len(instruments))
			for k := range instruments {
				available = append(available, k)
			}
			sort.Strings(available)
			return fmt.Errorf("instrument %q not found; available: %v", instrument, available)
		}

		// Resolve the scale to correct. Default to the (deterministically
		// chosen) first scale, which is the whole instrument for single-scale
		// banks like grit.
		scaleName := scaleFlag
		if scaleName == "" {
			names := make([]string, 0, len(reg.Models))
			for k := range reg.Models {
				names = append(names, k)
			}
			sort.Strings(names)
			if len(names) == 0 {
				return fmt.Errorf("instrument %q has no scales", instrument)
			}
			scaleName = names[0]
		}
		if _, ok := reg.Models[scaleName]; !ok {
			return fmt.Errorf("scale %q not found in instrument %q", scaleName, instrument)
		}

		model := reg.Models[scaleName]
		items := model.GetItems()
		nItems := len(items)
		sort.Ints(subsetSizes)

		grid := ndvek.Linspace(-10, 10, 400)

		// newScorer builds a fresh scorer that scores through the same full
		// Score() path used for the gold estimate, so subset and gold EAPs are
		// on the same footing. When imputation is enabled and a mixed model is
		// available, subset scores are Rao-Blackwellized over the unadministered
		// items (the "imputation-blended" regime from the Python example).
		// Imputation is optional: with --imputation=false (or when the
		// instrument ships no imputation model) the BCM is fit on plain IRT
		// subset scores, matching the naive regime in libfab's companion
		// examples.
		imEnabled := useImputation && reg.ImputationModel != nil
		newScorer := func() *irtcat.BayesianScorer {
			if imEnabled {
				var baseline irtcat.IrtModel
				if bm, ok := reg.BaselineModels[scaleName]; ok {
					baseline = *bm
				}
				return irtcat.NewBayesianScorerWithBaseline(
					grid, irtcat.DefaultAbilityPrior, *model, reg.ImputationModel, baseline)
			}
			return irtcat.NewBayesianScorerWithBaseline(
				grid, irtcat.DefaultAbilityPrior, *model, nil, nil)
		}

		scoreSubset := func(resps []irtcat.Response) float64 {
			scorer := newScorer()
			r := &irtcat.Responses{Responses: resps}
			if err := scorer.Score(r); err != nil {
				return math.NaN()
			}
			return scorer.Running.Mean()
		}

		fmt.Printf("Fitting BCM for %s / scale %q (%d items, imputation=%v)\n",
			instrument, scaleName, nItems, imEnabled)
		fmt.Printf("  subset sizes: %v, replicates/theta: %d\n", subsetSizes, replicates)

		type pair struct{ subset, gold float64 }
		train := make(map[int][]pair)
		test := make(map[int][]pair)

		// Build (subset_score, gold_score) triples the way the offline pipeline
		// does: sample a full response vector at a true theta, then for each
		// subset size mask a random subset of items and re-score. Gold is the
		// same model scored on all items. Sweep a grid of true abilities so the
		// BCM sees the whole score range.
		thetaGrid := ndvek.Linspace(-2, 2, 9)
		session := 0
		for _, theta := range thetaGrid {
			ab, err := ndvek.NewNdArray([]int{1}, []float64{theta})
			if err != nil {
				return err
			}
			for r := 0; r < replicates; r++ {
				// Sample a full response vector at this theta.
				samples := model.Sample(ab)
				full := make([]irtcat.Response, 0, nItems)
				for name, sv := range samples {
					if len(sv) == 0 {
						continue
					}
					item := irtcat.GetItemByName(name, items)
					if item == nil || len(item.ScoredValues) == 0 {
						continue
					}
					idx := sv[0]
					if idx < 0 || idx >= len(item.ScoredValues) {
						idx = len(item.ScoredValues) - 1
					}
					full = append(full, irtcat.Response{Item: item, Value: item.ScoredValues[idx]})
				}
				if len(full) < 2 {
					continue
				}

				gold := scoreSubset(full)
				if math.IsNaN(gold) {
					continue
				}

				bucket := train
				if session%2 == 1 {
					bucket = test
				}
				for _, j := range subsetSizes {
					if j < 1 || j > len(full) {
						continue
					}
					// Random subset of j items.
					perm := rand.Perm(len(full))
					subset := make([]irtcat.Response, j)
					for i := 0; i < j; i++ {
						subset[i] = full[perm[i]]
					}
					s := scoreSubset(subset)
					if math.IsNaN(s) {
						continue
					}
					bucket[j] = append(bucket[j], pair{s, gold})
				}
				session++
			}
		}

		// Fit the BCM on the training half.
		cells := make(map[int]biascorrection.Cell)
		for _, j := range subsetSizes {
			ps := train[j]
			if len(ps) < 2 {
				continue
			}
			xs := make([]float64, len(ps))
			ys := make([]float64, len(ps))
			for i, p := range ps {
				xs[i], ys[i] = p.subset, p.gold
			}
			cells[j] = biascorrection.Cell{SubsetScores: xs, GoldScores: ys}
		}
		set, err := biascorrection.FitSet(cells, scaleName)
		if err != nil {
			return fmt.Errorf("fit BCM: %w", err)
		}

		// Evaluate held-out bias reduction, mirroring Step 6 of the Python
		// example.
		fmt.Printf("\nHeld-out bias (test half): raw subset vs BCM-corrected, both against gold\n")
		fmt.Printf("%-6s %-8s %-12s %-12s %-10s\n", "J", "n", "L2(raw)", "L2(bcm)", "reduction")

		var sumRawSq, sumBcmSq float64
		var totalN int
		for _, j := range subsetSizes {
			bcm := set.For(j)
			if bcm == nil {
				continue
			}
			ps := test[j]
			if len(ps) == 0 {
				continue
			}
			var rawSq, bcmSq float64
			for _, p := range ps {
				rawErr := p.subset - p.gold
				bcmErr := bcm.Apply(p.subset) - p.gold
				rawSq += rawErr * rawErr
				bcmSq += bcmErr * bcmErr
			}
			l2Raw := math.Sqrt(rawSq / float64(len(ps)))
			l2Bcm := math.Sqrt(bcmSq / float64(len(ps)))
			reduction := 0.0
			if l2Raw > 0 {
				reduction = 100 * (l2Raw - l2Bcm) / l2Raw
			}
			fmt.Printf("%-6d %-8d %-12.4f %-12.4f %+.1f%%\n", j, len(ps), l2Raw, l2Bcm, reduction)

			sumRawSq += rawSq
			sumBcmSq += bcmSq
			totalN += len(ps)
		}

		if totalN > 0 {
			l2Raw := math.Sqrt(sumRawSq / float64(totalN))
			l2Bcm := math.Sqrt(sumBcmSq / float64(totalN))
			reduction := 0.0
			if l2Raw > 0 {
				reduction = 100 * (l2Raw - l2Bcm) / l2Raw
			}
			fmt.Printf("%-6s %-8d %-12.4f %-12.4f %+.1f%%\n", "all", totalN, l2Raw, l2Bcm, reduction)
		}

		if outPath != "" {
			if err := set.Save(outPath); err != nil {
				return fmt.Errorf("save BCM set: %w", err)
			}
			fmt.Printf("\nSaved BCM set -> %s (load with biascorrection.LoadSet)\n", outPath)
		}

		return nil
	},
}

func init() {
	BcmCmd.Flags().String("instrument", "grit", "Instrument to fit a BCM for")
	BcmCmd.Flags().String("scale", "", "Scale name (default: first scale of the instrument)")
	BcmCmd.Flags().Int("replicates", 200, "Simulated sessions per theta grid point")
	BcmCmd.Flags().IntSlice("subset-sizes", []int{3, 5, 8}, "Item subset sizes to fit BCMs for")
	BcmCmd.Flags().Bool("imputation", true, "Use the instrument's imputation model for subset scoring (if available)")
	BcmCmd.Flags().String("out", "", "Optional path to save the fitted BCM Set as JSON")
}
