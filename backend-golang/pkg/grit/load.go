package grit

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"log"

	gritmodel "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/grit"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
)

func LoadItems() []*irtcat.Item {
	cached, err := fs.ReadDir(gritmodel.FactorizedDir, "factorized")
	if err != nil {
		log.Fatal(err)
	}

	var items []*irtcat.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(gritmodel.FactorizedDir, "factorized/"+fn.Name())
		if err != nil {
			log.Fatal(err)
		}
		newItem := irtcat.LoadItemS(d, []int{0, 1, 2, 3, 4})
		if newItem != nil {
			items = append(items, newItem)
		}
	}
	return items
}

func LoadBaselineItems() []*irtcat.Item {
	cached, err := fs.ReadDir(gritmodel.BaselineFactorizedDir, "baseline_factorized")
	if err != nil {
		log.Printf("Warning: no baseline items for grit: %v", err)
		return nil
	}

	var items []*irtcat.Item
	for _, fn := range cached {
		if fn.Name() == ".gitkeep" {
			continue
		}
		d, err := fs.ReadFile(gritmodel.BaselineFactorizedDir, "baseline_factorized/"+fn.Name())
		if err != nil {
			log.Fatal(err)
		}
		newItem := irtcat.LoadItemS(d, []int{0, 1, 2, 3, 4})
		if newItem != nil {
			items = append(items, newItem)
		}
	}
	return items
}

func LoadImputationModel() (imputation.ImputationModel, error) {
	compressed, err := fs.ReadFile(gritmodel.ImputationModelDir, "imputation_model/config.yaml.gz")
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	pairwise, err := imputation.LoadFromYAML(data)
	if err != nil {
		return nil, err
	}
	if len(pairwise.MixedWeights) > 0 {
		return imputation.NewIrtMixedImputationModel(pairwise, pairwise.MixedWeights), nil
	}
	return pairwise, nil
}
