#!/usr/bin/env python3
"""Convert mice_loo_model.yaml (v1.1 format) to v2.0 format for Go loader,
then gzip the output.

v1.1 format:
  - univariate_results: list of {predictor_idx, target_idx, result: {fields...}}
  - zero_predictor_results: dict keyed by target idx

v2.0 format (what Go's LoadFromYAML expects):
  - univariate_meta: flat list with fields hoisted from nested result
  - zero_predictor_meta: dict keyed by target idx string
  - intercept_mean: wrapped as single-element list (Go reads [0])
  - version: '2.0'

Usage:
    python convert_mice.py
"""

import gzip
import os

import yaml

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BQ_IRT = os.path.join(os.path.dirname(REPO_ROOT), "bayesianquilts", "notebooks", "irt")
BACKEND = os.path.join(REPO_ROOT, "backend-golang")

INSTRUMENTS = ["grit", "npi", "tma", "wpi", "rwa"]


def wrap_intercept(val):
    """Wrap scalar intercept_mean as single-element list."""
    if val is None:
        return []
    if isinstance(val, list):
        return val
    return [val]


def convert_univariate_result(entry):
    """Flatten v1.1 univariate_results entry to v2.0 univariate_meta."""
    result = entry.get("result", {})
    out = {}
    out["target_idx"] = entry.get("target_idx", result.get("target_idx"))
    out["predictor_idx"] = entry.get("predictor_idx", result.get("predictor_idx"))
    out["n_obs"] = result.get("n_obs", 0)
    out["elpd_loo"] = result.get("elpd_loo", 0.0)
    out["elpd_loo_per_obs"] = result.get("elpd_loo_per_obs", 0.0)
    out["elpd_loo_per_obs_se"] = result.get("elpd_loo_per_obs_se", 0.0)
    out["khat_max"] = result.get("khat_max", 0.0)
    out["khat_mean"] = result.get("khat_mean", 0.0)
    out["converged"] = result.get("converged", False)

    predictor_mean = result.get("predictor_mean")
    out["predictor_mean"] = predictor_mean if predictor_mean is not None else 0.0
    predictor_std = result.get("predictor_std")
    out["predictor_std"] = predictor_std if predictor_std is not None else 1.0

    if result.get("beta_mean") is not None:
        out["beta_mean"] = result["beta_mean"]
    out["intercept_mean"] = wrap_intercept(result.get("intercept_mean"))
    if result.get("cutpoints_mean") is not None:
        out["cutpoints_mean"] = result["cutpoints_mean"]

    return out


def convert_zero_predictor(entry):
    """Convert v1.1 zero_predictor_results entry to v2.0 zero_predictor_meta."""
    out = {}
    out["target_idx"] = entry.get("target_idx", 0)
    out["n_obs"] = entry.get("n_obs", 0)
    out["elpd_loo"] = entry.get("elpd_loo", 0.0)
    out["elpd_loo_per_obs"] = entry.get("elpd_loo_per_obs", 0.0)
    out["elpd_loo_per_obs_se"] = entry.get("elpd_loo_per_obs_se", 0.0)
    out["khat_max"] = entry.get("khat_max", 0.0)
    out["khat_mean"] = entry.get("khat_mean", 0.0)
    out["converged"] = entry.get("converged", False)

    if entry.get("beta_mean") is not None:
        out["beta_mean"] = entry["beta_mean"]
    out["intercept_mean"] = wrap_intercept(entry.get("intercept_mean"))
    if entry.get("cutpoints_mean") is not None:
        out["cutpoints_mean"] = entry["cutpoints_mean"]

    return out


def convert_v11_to_v20(data):
    """Convert full v1.1 MICE model YAML to v2.0 format."""
    out = {}
    out["version"] = "2.0"

    # Copy config as-is
    out["config"] = data.get("config", {})

    # Copy data section
    out["data"] = data.get("data", {})

    # Copy prediction_graph
    out["prediction_graph"] = data.get("prediction_graph", {})

    # Convert zero_predictor_results → zero_predictor_meta
    zpr = data.get("zero_predictor_results", {})
    out["zero_predictor_meta"] = {}
    for key, entry in zpr.items():
        out["zero_predictor_meta"][str(key)] = convert_zero_predictor(entry)

    # Convert univariate_results → univariate_meta
    ur = data.get("univariate_results", [])
    out["univariate_meta"] = [convert_univariate_result(e) for e in ur]

    return out


def main():
    for instrument in INSTRUMENTS:
        yaml_path = os.path.join(BQ_IRT, instrument, "mice_loo_model.yaml")
        if not os.path.exists(yaml_path):
            print(f"SKIP {instrument}: {yaml_path} not found")
            continue

        print(f"Loading {instrument} mice_loo_model.yaml...")
        with open(yaml_path) as f:
            data = yaml.safe_load(f)

        v20 = convert_v11_to_v20(data)

        out_dir = os.path.join(BACKEND, instrument, "imputation_model")
        os.makedirs(out_dir, exist_ok=True)
        out_path = os.path.join(out_dir, "config.yaml.gz")

        yaml_bytes = yaml.dump(v20, default_flow_style=False, allow_unicode=True).encode("utf-8")
        with gzip.open(out_path, "wb") as f:
            f.write(yaml_bytes)

        n_uni = len(v20["univariate_meta"])
        n_zero = len(v20["zero_predictor_meta"])
        print(f"  {instrument}: wrote {out_path} ({n_uni} univariate, {n_zero} zero-predictor models)")


if __name__ == "__main__":
    main()
