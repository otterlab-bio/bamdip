# bamdip

A fast, lightweight BAM alignment record subsampler in pure Go, powered by [otterlab-bio/bamdriver](https://github.com/otterlab-bio/bamdriver).

`bamdip` operates on individual BAM alignment records. It allows extracting the first N alignments in stream order, performing uniform reservoir downsampling with a reproducible seed, or sampling by ratio, without enforcing artificial read-pairing constraints. Multi-mapping alignments in RNA-seq (e.g., 3-4 alignments sharing the same QNAME) are naturally preserved without dropping or corruption.

## 🚀 Highlights

| Feature | Description |
| :--- | :--- |
| ⚡ First-N Streaming | Extracts the first N alignment records in input order with minimal memory overhead. |
| 🎯 Record-Level Sampling | Treats each alignment record as an independent entry; does not drop multi-mappers or orphan reads. |
| 🎲 Reproducible Random Sampling | Offers reservoir downsampling (`-random`) and ratio sampling (`-ratio`) with deterministic seeds (`-seed`). |
| 🧱 Samtools Compatible | Decoded and re-encoded BAM outputs pass `samtools quickcheck` and match reference bash pipelines. |

## 🏗️ Architecture

```text
┌─────────────────┐
│    Input BAM    │
└────────┬────────┘
         │
         ▼ (Pure Go Reader via bamdriver)
┌─────────────────────────────────────────┐
│             Sampling Engine             │
│  ┌─────────────────┬─────────────────┐  │
│  │     First-N     │    Reservoir    │  │
│  │ (Default Order) │ (Random / Ratio)│  │
│  └─────────────────┴─────────────────┘  │
└────────────────────┬────────────────────┘
                     │ (Optional: -sort coord / -index)
                     ▼
┌─────────────────────────────────────────┐
│       Output BAM & Optional BAI         │
└─────────────────────────────────────────┘
```

## 📦 Quick Install

Requires Go 1.23 or newer:

```bash
go install github.com/otterlab-bio/bamdip/cmd@latest
# or build locally from source:
git clone https://github.com/otterlab-bio/bamdip.git
cd bamdip
go build -o bin/bamdip ./cmd
```

## ⚡ Quick Start

```bash
# 1. Extract the first 1,000 alignment records
bamdip -n 1000 input.bam output.bam

# 2. Randomly sample 10,000 alignments with a fixed random seed
bamdip -n 10000 -random -seed 42 input.bam output.bam

# 3. Sample 10% of records
bamdip -ratio 0.1 input.bam output.bam

# 4. Extract first 5,000 records, sort by coordinate, and build BAI index
bamdip -n 5000 -sort coord -index input.bam output.bam
```

## ⚙️ Options & Usage

```text
Usage:
  bamdip [options] <input.bam> <output.bam>

Sampling options (choose exactly one):
  -n INT, -count INT    Number of alignment records to sample (default: first N records)
  -ratio FLOAT          Sampling ratio (0.0-1.0), e.g., 0.1 for 10%
  -random               Perform random downsampling instead of default first-N
  -seed INT             Random seed (optional, for reproducible sampling)

Output options:
  -sort string          Sort order for output: name, coord, or none (default: none)
  -index                Create BAI index for output (requires -sort coord)
```

## 🧪 Verification & Equivalence

In first-N mode, `bamdip` outputs the exact alignment stream as the canonical samtools bash pipeline:

```bash
(
  samtools view -H input.bam
  samtools view input.bam | head -n "$N"
) | samtools view -b -o bash_reference.bam

bamdip -n "$N" input.bam bamdip_out.bam

# Verifies 100% record equivalence and valid BAM structure:
diff -u <(samtools view bash_reference.bam) <(samtools view bamdip_out.bam)
samtools quickcheck -v bamdip_out.bam
```

The repository test suite and GitHub Actions workflow execute these comparisons automatically against upstream `bamdriver` fixtures.

## 📄 License

MIT
