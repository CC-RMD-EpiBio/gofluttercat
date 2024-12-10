package rwas

import (
	"embed"
)

//go:embed factorized
var FactorizedDir embed.FS

//go:embed autoencoded
var AutoencodedDir embed.FS

// DistDirFS contains the embedded dist directory files (without the "dist" prefix)

// var FactorizedDirFS, _ = fs.Sub(factorizedDir, "factorized")
// var AutoencodedDirFS, _ = fs.Sub(autoencodedDir, "autoencoded")
