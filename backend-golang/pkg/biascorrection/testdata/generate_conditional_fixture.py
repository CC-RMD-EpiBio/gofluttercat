"""Generate Go-consumable fixtures for the conditional BCMs.

Trains libfab's BCMConditionalInfo and BCMConditional on synthetic data,
extracts the sklearn HistGradientBoostingRegressor internals (baseline +
predictor trees), and writes:

    bcm_conditional_info_fixture.json     -- model state
    bcm_conditional_info_predictions.json -- probe (input, expected)
    bcm_conditional_fixture.json
    bcm_conditional_predictions.json

The Go runtime walks the same trees on raw feature values, so parity
holds without going through joblib/pickle.

Run from this directory:

    uv run --no-project --python 3.11 --with scikit-learn \\
        python3 generate_conditional_fixture.py
"""
from __future__ import annotations

import json
import os
import sys

import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
LIBFAB_ROOT = os.path.abspath(os.path.join(
    HERE, "..", "..", "..", "..", "..", "libfabulouscatpy"))
if LIBFAB_ROOT not in sys.path:
    sys.path.insert(0, LIBFAB_ROOT)

from libfabulouscatpy.biascorrection import (  # noqa: E402
    BCMConditional,
    BCMConditionalInfo,
)


def extract_histgbr(model):
    """Pull baseline + predictor trees out of a fitted HistGBR into a
    Go-friendly dict. Assumes single-target regression (trees_per_iter=1)
    and numerical-only features (no categorical splits)."""
    baseline = float(np.asarray(model._baseline_prediction).reshape(-1)[0])
    trees = []
    for it_predictors in model._predictors:
        # Single-target regression -> exactly one tree per iteration.
        pred = it_predictors[0]
        nodes = pred.nodes
        if np.any(nodes["is_categorical"]):
            raise ValueError("categorical splits not supported by Go port")
        tree = [
            {
                "value": float(n["value"]),
                "feature_idx": int(n["feature_idx"]),
                "num_threshold": float(n["num_threshold"]),
                "left": int(n["left"]),
                "right": int(n["right"]),
                "is_leaf": bool(n["is_leaf"]),
                "missing_go_to_left": bool(n["missing_go_to_left"]),
            }
            for n in nodes
        ]
        trees.append(tree)
    return {
        "baseline_prediction": baseline,
        "n_features": int(model.n_features_in_),
        "trees": trees,
    }


def info_fixture():
    rng = np.random.default_rng(2026)
    n = 600
    score = rng.normal(0.0, 1.0, n)
    info = rng.uniform(0.2, 3.0, n)
    # Mild non-linear coupling so the trees do something interesting.
    gold = (
        0.7 * score
        + 0.15 * info
        - 0.05 * score * info
        + rng.normal(0.0, 0.35, n)
    )
    bcm = BCMConditionalInfo.fit(
        score, info, gold,
        scale_name="synthetic_info",
        n_folds=5,
        seed=2026,
        max_iter=60,
        learning_rate=0.08,
        max_depth=4,
    )
    fixture = {
        "scale_name": bcm.scale_name,
        "feature_names": list(bcm.feature_names),
        "model": extract_histgbr(bcm.model),
    }
    with open(os.path.join(HERE, "bcm_conditional_info_fixture.json"), "w") as f:
        json.dump(fixture, f, indent=2)

    probe_score = np.linspace(-3.0, 3.0, 13)
    probe_info = np.array([0.3, 0.8, 1.5, 2.5])
    rows = []
    for s in probe_score:
        for i in probe_info:
            y = float(bcm.apply(np.array([s]), np.array([i]))[0])
            rows.append({"score": float(s), "info": float(i), "expected": y})
    with open(os.path.join(HERE, "bcm_conditional_info_predictions.json"), "w") as f:
        json.dump({"probes": rows}, f, indent=2)
    print(f"Wrote bcm_conditional_info_*.json ({len(rows)} probes)")


def conditional_fixture():
    rng = np.random.default_rng(4242)
    item_keys = [f"item_{k:02d}" for k in range(8)]
    n = 700
    # Synthesize (score, indicators, gold).
    indicators = rng.integers(0, 2, size=(n, len(item_keys))).astype(float)
    # Each draw has a "subset size"; recompute score driven by indicators and
    # a latent ability so the relationship between score and gold is non-trivial.
    ability = rng.normal(0.0, 1.0, n)
    bias_per_item = np.array([0.0, 0.1, -0.05, 0.15, -0.1, 0.05, 0.0, 0.2])
    score = ability + indicators @ bias_per_item / np.maximum(indicators.sum(axis=1), 1)
    score += rng.normal(0.0, 0.2, n)
    gold = ability + rng.normal(0.0, 0.25, n)
    bcm = BCMConditional.fit(
        score, indicators, gold,
        item_keys=item_keys,
        scale_name="synthetic_cond",
        n_folds=5,
        seed=4242,
        max_iter=50,
        learning_rate=0.08,
        max_depth=3,
    )
    fixture = {
        "scale_name": bcm.scale_name,
        "item_keys": list(bcm.item_keys),
        "model": extract_histgbr(bcm.model),
    }
    with open(os.path.join(HERE, "bcm_conditional_fixture.json"), "w") as f:
        json.dump(fixture, f, indent=2)

    # Probes: a handful of (score, indicator-pattern) combinations.
    rng2 = np.random.default_rng(99)
    patterns = [
        np.array([1, 0, 1, 0, 1, 0, 1, 0], dtype=float),
        np.array([0, 1, 0, 1, 0, 1, 0, 1], dtype=float),
        np.array([1, 1, 0, 0, 1, 1, 0, 0], dtype=float),
        np.array([1, 1, 1, 1, 1, 1, 1, 1], dtype=float),
        np.array([0, 0, 0, 0, 0, 0, 0, 0], dtype=float),
        rng2.integers(0, 2, 8).astype(float),
        rng2.integers(0, 2, 8).astype(float),
    ]
    scores = np.linspace(-2.5, 2.5, 11)
    rows = []
    for s in scores:
        for pat in patterns:
            y = float(bcm.apply(np.array([s]), pat.reshape(1, -1))[0])
            rows.append({
                "score": float(s),
                "indicators": pat.tolist(),
                "expected": y,
            })
    with open(os.path.join(HERE, "bcm_conditional_predictions.json"), "w") as f:
        json.dump({"probes": rows}, f, indent=2)
    print(f"Wrote bcm_conditional_*.json ({len(rows)} probes)")


if __name__ == "__main__":
    info_fixture()
    conditional_fixture()
