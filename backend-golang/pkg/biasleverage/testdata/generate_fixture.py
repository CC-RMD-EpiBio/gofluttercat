"""Generate Python <-> Go parity fixture for the biasleverage package.

Mirrors the math of compute_item_bias_leverage.py from cat_optimalcontrol
on a fully synthetic GRM dataset, so the Go and Python implementations
can be cross-validated without depending on libfabulouscatpy or any IRT
fit artifacts.

Outputs:
    leverage_fixture.json   — items, training data, theta_hat, theta_bar,
                              and the synthetic PMFProvider's predictions
                              keyed by (person_idx, target_item).
    leverage_expected.json  — per-item B_i, F_i, ratio, n_eff that the Go
                              implementation must reproduce to tolerance.

The synthetic PMFProvider is a deterministic perturbation of the GRM PMF
with a known additive shift in logit space, so its outputs differ from
the IRT model in a controlled way (giving non-zero B_i).
"""

import json
import os

import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))


def grm_pmf(theta, a, d0, ddiff, K):
    """GRM PMF using the bayesianquilts production convention:
        P(Y >= k) = sigmoid(a * (theta - tau_k)),
    with monotone-increasing thresholds tau."""
    thresholds = np.empty(K - 1)
    thresholds[0] = d0
    for k in range(1, K - 1):
        thresholds[k] = thresholds[k - 1] + ddiff[k - 1]
    cdf = np.empty(K + 1)
    cdf[0] = 1.0
    cdf[-1] = 0.0
    for k in range(1, K):
        eta = a * (theta - thresholds[k - 1])
        cdf[k] = 1.0 / (1.0 + np.exp(-eta))
    return cdf[:-1] - cdf[1:]


def fisher_info(theta, a, d0, ddiff, K):
    eps = 1e-3
    floor = 1e-12
    p = grm_pmf(theta, a, d0, ddiff, K)
    pp = grm_pmf(theta + eps, a, d0, ddiff, K)
    pm = grm_pmf(theta - eps, a, d0, ddiff, K)
    p = np.maximum(p, floor)
    pp = np.maximum(pp, floor)
    pm = np.maximum(pm, floor)
    dlog = (np.log(pp) - np.log(pm)) / (2 * eps)
    return float(np.sum(p * dlog ** 2))


def synth_pw_pmf(theta_self, item, others, perturb_amplitude=0.6):
    """Deterministic 'imputation' PMF: GRM PMF at a perturbed theta.

    The perturbation depends on the sum of other responses, so different
    observation patterns yield different predictions — ensuring Go must
    pass the same `others` map for the parity test to succeed.
    """
    K = item["K"]
    others_sum = sum(int(v) for v in others.values())
    shift = perturb_amplitude * np.tanh(0.1 * (others_sum - 5))
    return grm_pmf(theta_self + shift, item["a"], item["d0"],
                   np.array(item["ddiff"]), K)


def main():
    rng = np.random.default_rng(2026)

    # 6 items, K=4 categories.
    K = 4
    item_keys = [f"q{i+1}" for i in range(6)]
    items = {}
    for i, key in enumerate(item_keys):
        items[key] = {
            "a": float(rng.uniform(0.6, 1.4)),
            "d0": float(rng.uniform(-1.5, 0.0)),
            "ddiff": rng.uniform(0.4, 1.2, size=K - 2).tolist(),
            "K": K,
        }

    # 30 synthetic respondents with abilities ~ N(0, 1).
    N = 30
    theta_true = rng.normal(0.0, 1.0, size=N)
    training = []
    for n in range(N):
        person = {}
        for key in item_keys:
            it = items[key]
            pmf = grm_pmf(theta_true[n], it["a"], it["d0"],
                          np.array(it["ddiff"]), K)
            pmf = np.clip(pmf, 0.0, None)
            pmf = pmf / pmf.sum()
            # Inject ~10% missingness.
            if rng.uniform() < 0.10:
                continue
            person[key] = float(rng.choice(K, p=pmf))
        training.append(person)

    # Simulate baseline EAP scores: noisy version of theta_true.
    theta_hat = (theta_true + rng.normal(0.0, 0.15, size=N)).tolist()
    theta_bar = float(np.mean(theta_hat))

    # Synthetic PMFProvider: precompute the prediction for every
    # (person_idx, target_item) where target is observed and at least one
    # other item is observed. The Go test loads these directly.
    provider_predictions = {}
    for p, person in enumerate(training):
        obs = {k: int(v) for k, v in person.items() if 0 <= int(v) < K}
        for target in obs:
            others = {k: v for k, v in obs.items() if k != target}
            if not others:
                continue
            pmf = synth_pw_pmf(theta_hat[p], items[target], others)
            provider_predictions[f"{p}|{target}"] = pmf.tolist()

    # Compute expected B_i, F_i, ratio, n_eff.
    floor = 1e-12
    sums = {k: 0.0 for k in item_keys}
    counts = {k: 0 for k in item_keys}
    for p, person in enumerate(training):
        theta_p = theta_hat[p]
        if not np.isfinite(theta_p):
            continue
        obs = {k: int(v) for k, v in person.items() if 0 <= int(v) < K}
        for target, yi in obs.items():
            others = {k: v for k, v in obs.items() if k != target}
            if not others:
                continue
            pmf_pw = np.array(provider_predictions[f"{p}|{target}"])
            it = items[target]
            pmf_irt = grm_pmf(theta_p, it["a"], it["d0"],
                              np.array(it["ddiff"]), K)
            log_pw = np.log(max(float(pmf_pw[yi]), floor))
            log_irt = np.log(max(float(pmf_irt[yi]), floor))
            sums[target] += abs(log_pw - log_irt)
            counts[target] += 1

    expected = []
    for key in item_keys:
        it = items[key]
        c = counts[key]
        B = sums[key] / c if c > 0 else float("nan")
        F = fisher_info(theta_bar, it["a"], it["d0"],
                        np.array(it["ddiff"]), K)
        ratio = (B / F) if F > 1e-6 else float("nan")
        expected.append({
            "item": key, "B": B, "F": F, "ratio": ratio, "n_eff": c,
        })

    fixture = {
        "K": K,
        "item_keys": item_keys,
        "items": items,
        "training": training,
        "theta_hat": theta_hat,
        "theta_bar": theta_bar,
        "provider_predictions": provider_predictions,
    }
    with open(os.path.join(HERE, "leverage_fixture.json"), "w") as f:
        json.dump(fixture, f, indent=2)
    with open(os.path.join(HERE, "leverage_expected.json"), "w") as f:
        json.dump({"items": expected}, f, indent=2)

    print(f"Wrote leverage_fixture.json (N={N}, items={len(item_keys)}, "
          f"provider_predictions={len(provider_predictions)})")
    print("Per-item expected:")
    for e in expected:
        print(f"  {e['item']}  B={e['B']:.4f}  F={e['F']:.4f}  "
              f"B/F={e['ratio']:.4f}  n={e['n_eff']}")


if __name__ == "__main__":
    main()
