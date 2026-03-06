# dr-evaluation

A CLI tool to evaluate and compare Velero backup/restore performance on OpenShift clusters. It connects to a Kubernetes cluster, fetches Velero `Backup` and `Restore` custom resources, and generates detailed Markdown reports.

## Features

- **Analysis reports**: Evaluate backup/restore operations for a given time range or the latest N samples.
- **Comparison reports**: Compare performance before and after a specific date (e.g., a plugin change), with delta percentages, duration trends, and automated key takeaways.
- Automatic classification of backup types: FVT, Daily Full, HC Daily, and Other.
- Duration statistics (min/max/avg) per category.
- Phase breakdown, error/warning summaries, and failure details.
- Markdown output ready to share or publish.

## Prerequisites

- Go 1.22+
- Access to a Kubernetes cluster with Velero installed (via kubeconfig)
- Velero `Backup` and `Restore` CRDs in the target namespace

## Build

```bash
make build
```

The binary is placed in `bin/dr-evaluation`. You can also run `make all` to format, vet, test, and build in one step.

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make all` | Format, vet, test, and build |
| `make build` | Build the binary into `bin/` |
| `make test` | Run unit tests |
| `make test-verbose` | Run unit tests with verbose output |
| `make test-cover` | Run tests with coverage report |
| `make fmt` | Format source code |
| `make vet` | Run `go vet` |
| `make lint` | Run `golangci-lint` (requires install) |
| `make clean` | Remove build artifacts |

## Usage

### `analysis` — Backup/Restore Analysis Report

Generates a report of Velero backups and restores. Dates are optional:

```bash
# Last 10 backups/restores per category (no dates needed)
bin/dr-evaluation analysis \
  --kubeconfig /path/to/kubeconfig \
  --sample 10

# From a start date until now
bin/dr-evaluation analysis \
  --start "2026-03-05T00:00:00Z" \
  --kubeconfig /path/to/kubeconfig

# Explicit date range
bin/dr-evaluation analysis \
  --start "2026-03-05T00:00:00Z" \
  --end "2026-03-06T23:59:59Z" \
  --kubeconfig /path/to/kubeconfig \
  --sample 5

# Custom output path
bin/dr-evaluation analysis \
  --kubeconfig /path/to/kubeconfig \
  --output my-report.md
```

#### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--kubeconfig` | Yes | — | Path to the kubeconfig file |
| `--start` | No | — | Start date in ISO8601 format. If omitted, reports the last N samples |
| `--end` | No | `now()` | End date in ISO8601 format |
| `--namespace` | No | `openshift-adp` | Namespace where Velero resources live |
| `--sample` | No | `5` | Number of backups/restores to display per category |
| `--output` | No | `reports/<cmd>_<timestamp>.md` | Output Markdown file path |

### `comparison` — Pre/Post Performance Comparison

Compares backup/restore performance before and after a cutoff date. Useful for evaluating plugin changes, upgrades, or configuration updates.

```bash
# Basic comparison
bin/dr-evaluation comparison \
  --date "2026-03-05T00:00:00Z" \
  --kubeconfig /path/to/kubeconfig

# With more trend samples
bin/dr-evaluation comparison \
  --date "2026-03-05T00:00:00Z" \
  --kubeconfig /path/to/kubeconfig \
  --sample 10

# Custom output path
bin/dr-evaluation comparison \
  --date "2026-03-05T00:00:00Z" \
  --kubeconfig /path/to/kubeconfig \
  --output comparison.md
```

#### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--kubeconfig` | Yes | — | Path to the kubeconfig file |
| `--date` | Yes | — | Cutoff date in ISO8601 format (separates pre/post periods) |
| `--namespace` | No | `openshift-adp` | Namespace where Velero resources live |
| `--sample` | No | `5` | Number of pre-change samples shown in the duration trend |
| `--output` | No | `reports/<cmd>_<timestamp>.md` | Output Markdown file path |

## Report Contents

### Analysis Report

- Metadata table (period, namespace, totals)
- Backup tables per category (FVT, Daily Full, HC Daily, Other) with name, timestamps, duration, items, warnings, errors, phase, and TTL
- FVT Restore table with corresponding backup matches
- Phase breakdown for backups and restores
- Aggregated error/warning summary
- Duration statistics (min/max/avg) for FVT backups and restores

### Comparison Report

- Pre/post metadata (period boundaries, counts)
- Per-category comparison tables with delta percentages for duration, items, success rate, and errors
- ASCII duration trend chart showing the transition from pre to post
- Pre-change failure details (for HC Daily or any category with failures)
- Auto-generated key takeaways section

## Project Structure

```
dr-evaluation/
├── main.go                     # Entry point
├── go.mod / go.sum              # Go module files
├── Makefile                     # Build, test, and lint targets
├── .gitignore
├── cmd/
│   ├── root.go                  # Root cobra command
│   ├── output.go                # Shared output/reports helpers
│   ├── analysis.go              # "analysis" subcommand
│   └── comparison.go            # "comparison" subcommand
├── pkg/
│   ├── velero/
│   │   ├── types.go             # Types, classification, filtering, stats
│   │   ├── types_test.go
│   │   ├── client.go            # Kubernetes dynamic client for Velero CRDs
│   │   └── client_test.go
│   └── report/
│       ├── common.go            # Shared formatting helpers
│       ├── common_test.go
│       ├── analysis.go          # Analysis report generator
│       ├── analysis_test.go
│       ├── comparison.go        # Comparison report generator
│       └── comparison_test.go
├── bin/                         # Build output (gitignored)
└── reports/                     # Default report output (gitignored)
```

## Example Output

### Analysis (no dates, last 3 samples)

```
# Velero Backup/Restore Report

| Field | Value |
|-------|-------|
| **Period** | Last 3 samples (up to `2026-03-06T09:59:03Z`) |
| **Namespace** | `openshift-adp` |
| **Total backups available** | 103 |

## FVT Backups (showing 3 of 74)

| # | Name | Start | End | Duration | Items | Warnings | Errors | Phase |
|---|------|-------|-----|----------|-------|----------|--------|-------|
| 1 | `2oroao31...-bkp-fvt` | 03-05 19:22:28 | 03-05 19:24:39 | **2m 11s** | 551 | 0 | 0 | Completed |
| 2 | `2orsq5mg...-bkp-fvt` | 03-06 00:28:48 | 03-06 00:31:11 | **2m 23s** | 551 | 0 | 0 | Completed |
| 3 | `2os22urs...-bkp-fvt` | 03-06 06:28:37 | 03-06 06:30:14 | **1m 37s** | 551 | 0 | 0 | Completed |
```

### Comparison (pre/post cutoff)

```
## FVT Backups

| Metric | Pre-change | Post-change | Delta |
|--------|------------|-------------|-------|
| **Avg duration** | 4m 42s | 3m 4s | **-34.7%** |
| **Success rate** | 100.0% (69/69) | 100.0% (5/5) | Same |

### Duration Trend

PRE  | 2026-03-04 12:28 |  314s | ###############################
POST | 2026-03-06 06:28 |   97s | ######### <<<
```
