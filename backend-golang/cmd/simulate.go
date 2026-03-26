package cmd

import (
	"fmt"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/simulation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web"
	"github.com/mederrata/ndvek"
	"github.com/spf13/cobra"
)

var SimulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Run CAT session simulation",
	Long:  "Run Monte Carlo simulation of CAT sessions to evaluate convergence",
	RunE: func(cmd *cobra.Command, args []string) error {
		instrument, _ := cmd.Flags().GetString("instrument")
		theta, _ := cmd.Flags().GetFloat64("theta")
		replicates, _ := cmd.Flags().GetInt("replicates")
		maxItems, _ := cmd.Flags().GetInt("max-items")
		seed, _ := cmd.Flags().GetInt64("seed")
		selectorName, _ := cmd.Flags().GetString("selector")

		config := conf.GetConfig()
		instruments := web.LoadAllInstruments(config)

		reg, ok := instruments[instrument]
		if !ok {
			available := make([]string, 0, len(instruments))
			for k := range instruments {
				available = append(available, k)
			}
			return fmt.Errorf("instrument %q not found; available: %v", instrument, available)
		}

		// Build theta map (same theta for all scales)
		thetaMap := make(map[string]float64)
		for scaleName := range reg.Models {
			thetaMap[scaleName] = theta
		}

		// Create selector
		var selector irtcat.ItemSelector
		switch selectorName {
		case "cross-entropy":
			selector = irtcat.NewEntropySelector(0)
		case "bayesian-fisher":
			selector = irtcat.BayesianFisherSelector{Temperature: 0}
		case "fisher":
			selector = irtcat.FisherSelector{Temperature: 0}
		default:
			return fmt.Errorf("unknown selector: %s (options: cross-entropy, bayesian-fisher, fisher)", selectorName)
		}

		sim := &simulation.CATSimulator{
			Models:          reg.Models,
			Selector:        selector,
			ImputationModel: reg.ImputationModel,
			MaxItems:        maxItems,
			AbilityGridPts:  ndvek.Linspace(-10, 10, 400),
			Prior:           irtcat.DefaultAbilityPrior,
		}

		summary := sim.Simulate(thetaMap, replicates, seed)

		for _, scale := range summary.Scales {
			fmt.Printf("\nScale: %s\n", scale)
			fmt.Printf("%-6s %-10s %-10s %-10s %-10s %-10s %-10s\n",
				"Step", "MeanL2", "StdL2", "MeanKL", "StdKL", "MeanSE", "StdSE")
			for i := range summary.MaxItems {
				fmt.Printf("%-6d %-10.4f %-10.4f %-10.4f %-10.4f %-10.4f %-10.4f\n",
					i,
					summary.MeanL2[scale][i],
					summary.StdL2[scale][i],
					summary.MeanKL[scale][i],
					summary.StdKL[scale][i],
					summary.MeanSE[scale][i],
					summary.StdSE[scale][i])
			}
		}

		return nil
	},
}

func init() {
	SimulateCmd.Flags().String("instrument", "rwa", "Instrument to simulate")
	SimulateCmd.Flags().Float64("theta", 0.0, "True ability level")
	SimulateCmd.Flags().Int("replicates", 100, "Number of Monte Carlo replicates")
	SimulateCmd.Flags().Int("max-items", 0, "Max items per session (0 = all)")
	SimulateCmd.Flags().Int64("seed", 0, "RNG seed (0 = no seed)")
	SimulateCmd.Flags().String("selector", "cross-entropy", "Item selection method")
}
