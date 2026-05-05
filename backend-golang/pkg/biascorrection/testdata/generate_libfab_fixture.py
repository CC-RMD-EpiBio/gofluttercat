"""Generate a Go-consumable BCM fixture using the libfab BCMSet (rather
than calling sklearn directly), so the Go parity test verifies that:

  libfab.fit_bcm_set(...).save(JSON)  ->  Go LoadSet(JSON).Apply  ==  sklearn.predict

This exercises the full libfab serialization path, not just sklearn's
threshold arrays. Outputs:

    libfab_bcm_fixture.json     -- BCMSet saved by libfab.
    libfab_bcm_predictions.json -- probe predictions computed by libfab.
"""

import json
import os
import sys

import numpy as np

# Make libfab importable when this script is run from inside the testdata dir.
HERE = os.path.dirname(os.path.abspath(__file__))
LIBFAB_ROOT = os.path.abspath(os.path.join(
    HERE, "..", "..", "..", "..", "..", "libfabulouscatpy"))
if LIBFAB_ROOT not in sys.path:
    sys.path.insert(0, LIBFAB_ROOT)

from libfabulouscatpy.biascorrection import fit_bcm_set  # noqa: E402


def synthesize(rng, n=400, slope=0.6, bias=-0.2, spread=0.4, x_lo=-3.0, x_hi=3.0):
    x = rng.uniform(x_lo, x_hi, size=n)
    y = slope * x + bias + rng.normal(0.0, spread * (1 + 0.2 * np.abs(x)))
    return x, y


def main():
    rng = np.random.default_rng(2026)
    cells = {}
    for J in (5, 10, 20):
        slope = 0.5 + 0.05 * J / 20
        b = -0.3 + 0.02 * J
        cells[J] = synthesize(rng, n=300 + 50 * J, slope=slope, bias=b, spread=0.5)

    bcm_set = fit_bcm_set(cells, scale="libfab_synthetic")
    bcm_set.save(os.path.join(HERE, "libfab_bcm_fixture.json"))

    probe_xs = np.concatenate([
        np.linspace(-4.0, 4.0, 41),
        np.array([-100.0, -3.0, -2.999, 0.0, 2.999, 3.0, 100.0]),
    ])
    probes = {}
    for J in cells:
        bcm = bcm_set.maps[J]
        preds = bcm.apply(probe_xs).tolist()
        probes[str(J)] = [
            {"input": float(x), "expected": float(y)}
            for x, y in zip(probe_xs.tolist(), preds)
        ]
    with open(os.path.join(HERE, "libfab_bcm_predictions.json"), "w") as f:
        json.dump(probes, f, indent=2)

    print(f"Wrote libfab_bcm_fixture.json (J={list(cells)})")
    print(f"Wrote libfab_bcm_predictions.json ({len(probe_xs)} probes per cell)")


if __name__ == "__main__":
    main()
