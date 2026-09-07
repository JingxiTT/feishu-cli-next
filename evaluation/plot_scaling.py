#!/usr/bin/env python3
"""Export the completed development size comparison as PNG and SVG.

Chart contract: compare actual provider tokens by tested row count and arm.
Six static Matplotlib dot panels show Base/Sheets x three arms, with nine
attempts per panel (three fresh trials at each of 100, 1,000 and 10,000 rows).
Every known total is shown, including failed/inconclusive attempts. Unknown
totals are disclosed rather than plotted at zero. No fitted curve or asymptotic
claim; final interpretation depends on the completed evidence, not this script.
Single blue root plus neutrals; marker shapes distinguish outcome without color.
Shared zero-based y scales within each task, log x ticks at the tested sizes.
Output: usage-by-size.png/.svg beside the input batches; inspect the exported
image in the accompanying Markdown report before shipping it.
"""
import argparse
import hashlib
import json
from pathlib import Path

ARMS = ("official", "official_concise", "next")
SIZES = (100, 1000, 10000)
TITLES = ("Official + official guidance", "Official + concise guidance", "Next + concise guide")


def read_batch(path, section):
    raw = path.read_bytes()
    batch = json.loads(raw)
    if batch.get("mode") != "real" or batch.get("split") != "development":
        raise ValueError("A real development comparison is required")
    runs = batch["runs"]
    if batch.get("completed") is not True or batch.get("planned_runs") != 27 or len(runs) != 27:
        raise ValueError(f"{section} requires a completed 27-attempt batch")
    seen = set()
    rows = []
    for run in runs:
        task_ids = {size: f"base-scale-dev-{size:05d}" if section == "base" else f"sheets-size-dev-{size}"
                    for size in SIZES}
        matches = [size for size, task_id in task_ids.items() if run["task_id"] == task_id]
        if len(matches) != 1 or run["arm"] not in ARMS or type(run["trial"]) is not int or run["trial"] not in (0, 1, 2):
            raise ValueError("Unexpected size, arm or trial identity")
        size = matches[0]
        key = run["arm"], size, run["trial"]
        if key in seen:
            raise ValueError("Duplicate trial")
        seen.add(key)
        if type(run["success"]) is not bool or type(run["outcome_inconclusive"]) is not bool:
            raise ValueError("Missing outcome classification")
        usage = run["usage"]
        if usage is not None:
            keys = ("input_tokens", "cached_input_tokens", "output_tokens", "total_tokens")
            if any(type(usage.get(k)) is not int or usage[k] < 0 for k in keys):
                raise ValueError("Invalid provider usage")
            if usage["cached_input_tokens"] > usage["input_tokens"] or usage["total_tokens"] != usage["input_tokens"] + usage["output_tokens"]:
                raise ValueError("Provider usage must include cached input in full")
        rows.append({**run, "row_count": size})
    return rows, hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("evidence_directory", type=Path)
    args = parser.parse_args()
    data, hashes = {}, {}
    for section in ("base", "sheets"):
        data[section], hashes[section] = read_batch(args.evidence_directory / section / "batch-01.json", section)

    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    from matplotlib.lines import Line2D
    from matplotlib.ticker import FuncFormatter

    plt.rcParams.update({"font.family": "DejaVu Sans", "font.size": 10,
                         "text.color": "#20252B", "axes.labelcolor": "#20252B",
                         "axes.edgecolor": "#A0A5AB", "axes.linewidth": 0.7,
                         "xtick.color": "#555B63", "ytick.color": "#555B63",
                         "svg.fonttype": "none"})
    styles = {"verified": {"marker": "o", "color": "#2563EB"},
              "nonverified": {"marker": "x", "color": "#343A40"},
              "inconclusive": {"marker": "D", "edgecolors": "#6B7280", "facecolors": "none"}}
    fig, axes = plt.subplots(2, 3, figsize=(12, 7.2), sharey="row")
    fig.subplots_adjust(left=0.09, right=0.985, top=0.81, bottom=0.23, hspace=0.52, wspace=0.20)
    fig.suptitle("Provider tokens by data size", x=0.09, y=0.975, ha="left", fontsize=20)
    fig.text(0.09, 0.915, "54 fresh attempts · three trials per arm and size · all recorded outcomes retained", fontsize=11)
    for row_index, section in enumerate(("base", "sheets")):
        observed = [r["usage"]["total_tokens"] for r in data[section] if r["usage"] is not None]
        upper = max(observed, default=1) * 1.15 or 1
        for col_index, arm in enumerate(ARMS):
            ax = axes[row_index, col_index]
            selected = [r for r in data[section] if r["arm"] == arm]
            for run in selected:
                if run["usage"] is None:
                    continue
                state = "inconclusive" if run["outcome_inconclusive"] else "verified" if run["success"] else "nonverified"
                x = run["row_count"] * 10 ** ((run["trial"] - 1) * 0.045)
                ax.scatter(x, run["usage"]["total_tokens"], s=48, linewidths=1.3, zorder=3, **styles[state])
            ax.set_xscale("log")
            ax.set_xlim(50, 20000)
            ax.set_ylim(0, upper)
            labels = [f'{size:,}\n{sum(r["success"] for r in selected if r["row_count"] == size)}/3 verified' for size in SIZES]
            ax.set_xticks(SIZES, labels, fontsize=9)
            ax.minorticks_off()
            ax.grid(axis="y", color="#E5E7EB", linewidth=0.7)
            ax.spines[["top", "right"]].set_visible(False)
            ax.yaxis.set_major_formatter(FuncFormatter(lambda value, _: f"{value / 1000:,.0f}"))
            if row_index == 0:
                ax.set_title(TITLES[col_index], fontsize=11, loc="left", pad=12)
            if col_index == 0:
                ax.set_ylabel(("Base import" if section == "base" else "Sheets export") + "\nProvider tokens (thousands)")
            unknown = [str(size) + ":" + str(sum(r["usage"] is None for r in selected if r["row_count"] == size)) for size in SIZES]
            ax.text(0.02, 0.965, "Unknown totals by size: " + ", ".join(unknown), transform=ax.transAxes,
                    va="top", fontsize=8, color="#555B63")
    handles = [Line2D([], [], marker="o", color="#2563EB", linestyle="none", label="Verified"),
               Line2D([], [], marker="x", color="#343A40", linestyle="none", label="Nonverified"),
               Line2D([], [], marker="D", markerfacecolor="none", color="#6B7280", linestyle="none", label="Inconclusive")]
    fig.legend(handles=handles, loc="lower left", bbox_to_anchor=(0.08, 0.095), ncol=3, frameon=False)
    fig.text(0.09, 0.075, "X: tested row count (log scale); horizontal offsets separate repeated trials. Y scales are shared within each task.", fontsize=9)
    fig.text(0.09, 0.045, "Provider input includes cached input in full. Unknown totals are not zero and are not plotted. No fitted scaling curve.", fontsize=9)
    description = "Source batch SHA-256: " + json.dumps(hashes, sort_keys=True)
    for extension in ("png", "svg"):
        fig.savefig(args.evidence_directory / f"usage-by-size.{extension}", dpi=180,
                    facecolor="white", metadata={"Description": description})
    plt.close(fig)
    print("Exported usage-by-size.png and usage-by-size.svg; final visual QA required.")


if __name__ == "__main__":
    main()
