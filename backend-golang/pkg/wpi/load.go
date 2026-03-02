package wpi

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"log"

	wpimodel "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/wpi"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
)

func LoadItems() []*irtcat.Item {
	cached, err := fs.ReadDir(wpimodel.FactorizedDir, "factorized")
	if err != nil {
		log.Fatal(err)
	}

	var items []*irtcat.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(wpimodel.FactorizedDir, "factorized/"+fn.Name())
		if err != nil {
			log.Fatal(err)
		}
		newItem := irtcat.LoadItemS(d, []int{0, 1})
		if newItem != nil {
			items = append(items, newItem)
		}
	}
	return items
}

func LoadImputationModel() (*imputation.MiceBayesianLoo, error) {
	compressed, err := fs.ReadFile(wpimodel.ImputationModelDir, "imputation_model/config.yaml.gz")
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
	return imputation.LoadFromYAML(data)
}
