package rwas

import (
	"embed"
	"io/fs"
)

var factorizedDir embed.FS
var autoencodedDir embed.FS

// DistDirFS contains the embedded dist directory files (without the "dist" prefix)
var FactorizedDirFS, _ = fs.Sub(factorizedDir, "factorized")
var AutoencodedDirFS, _ = fs.Sub(autoencodedDir, "autoencoded")
