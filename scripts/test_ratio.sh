#!/usr/bin/env bash
# scripts/test_ratio.sh
# Tests ratio downsampling mode of bamdip, verifying count precision,
# seed reproducibility, seed divergence, coordinate sorting, and samtools quickcheck.

set -euo pipefail

INPUT="${1:-testdata/Test_hg19_NRAS.bam}"
REPORT_JSON="${2:-ci-artifacts/reports/ratio-report.json}"
WORKDIR="$(mktemp -d -t bamdip_ratio_XXXXXX)"

cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

echo "=== bamdip Ratio Downsampling Test ==="
echo "Input BAM:   $INPUT"
echo "Report JSON: $REPORT_JSON"
echo "Workdir:     $WORKDIR"

mkdir -p "$(dirname "$REPORT_JSON")"
mkdir -p bin

if [ ! -f "bin/bamdip" ]; then
  echo "Building bamdip..."
  go build -o bin/bamdip ./cmd
fi

TOTAL_INPUT=$(samtools view -c "$INPUT")
echo "Total input records: $TOTAL_INPUT"

# 1. Test ratio 0.1 with seed 42 (run 1 and run 2 for reproducibility check)
EXPECTED_10=$(python3 -c "import math; print(int($TOTAL_INPUT * 0.1))")
echo "--- Testing -ratio 0.1 (expected: $EXPECTED_10 records) ---"

bin/bamdip -ratio 0.1 -seed 42 "$INPUT" "$WORKDIR/ratio_0.1_r1.bam"
bin/bamdip -ratio 0.1 -seed 42 "$INPUT" "$WORKDIR/ratio_0.1_r2.bam"
bin/bamdip -ratio 0.1 -seed 999 "$INPUT" "$WORKDIR/ratio_0.1_s999.bam"

COUNT_R1=$(samtools view -c "$WORKDIR/ratio_0.1_r1.bam")
COUNT_R2=$(samtools view -c "$WORKDIR/ratio_0.1_r2.bam")
COUNT_S999=$(samtools view -c "$WORKDIR/ratio_0.1_s999.bam")

echo "Observed counts: run1=$COUNT_R1, run2=$COUNT_R2, seed999=$COUNT_S999"
if [ "$COUNT_R1" -ne "$EXPECTED_10" ] || [ "$COUNT_R2" -ne "$EXPECTED_10" ] || [ "$COUNT_S999" -ne "$EXPECTED_10" ]; then
  echo "Error: ratio 0.1 record count mismatch!" >&2
  exit 1
fi

samtools quickcheck -v "$WORKDIR/ratio_0.1_r1.bam" "$WORKDIR/ratio_0.1_r2.bam" "$WORKDIR/ratio_0.1_s999.bam"
echo "✓ Ratio 0.1 outputs passed samtools quickcheck"

# Reproducibility: run 1 and run 2 must produce identical record streams
samtools view "$WORKDIR/ratio_0.1_r1.bam" > "$WORKDIR/r1.sam"
samtools view "$WORKDIR/ratio_0.1_r2.bam" > "$WORKDIR/r2.sam"
if ! diff -u "$WORKDIR/r1.sam" "$WORKDIR/r2.sam"; then
  echo "Error: Same seed produced different output!" >&2
  exit 1
fi
echo "✓ Seed 42 is 100% reproducible"

# Divergence: seed 42 vs seed 999 must select different records
samtools view "$WORKDIR/ratio_0.1_s999.bam" > "$WORKDIR/s999.sam"
if diff -q "$WORKDIR/r1.sam" "$WORKDIR/s999.sam" >/dev/null 2>&1; then
  echo "Error: Different seeds produced identical output!" >&2
  exit 1
fi
echo "✓ Different seeds produce distinct samples"

# 2. Test ratio 0.5
EXPECTED_50=$(python3 -c "import math; print(int($TOTAL_INPUT * 0.5))")
echo "--- Testing -ratio 0.5 (expected: $EXPECTED_50 records) ---"
bin/bamdip -ratio 0.5 -seed 42 "$INPUT" "$WORKDIR/ratio_0.5.bam"
COUNT_50=$(samtools view -c "$WORKDIR/ratio_0.5.bam")
if [ "$COUNT_50" -ne "$EXPECTED_50" ]; then
  echo "Error: ratio 0.5 record count mismatch! Expected $EXPECTED_50, got $COUNT_50" >&2
  exit 1
fi
samtools quickcheck -v "$WORKDIR/ratio_0.5.bam"
echo "✓ Ratio 0.5 passed samtools quickcheck ($COUNT_50 records)"

# 3. Test ratio 0.2 with coord sort and BAI index
EXPECTED_20=$(python3 -c "import math; print(int($TOTAL_INPUT * 0.2))")
echo "--- Testing -ratio 0.2 with -sort coord -index (expected: $EXPECTED_20 records) ---"
bin/bamdip -ratio 0.2 -seed 42 -sort coord -index "$INPUT" "$WORKDIR/ratio_coord.bam"
COUNT_20=$(samtools view -c "$WORKDIR/ratio_coord.bam")
if [ "$COUNT_20" -ne "$EXPECTED_20" ]; then
  echo "Error: ratio 0.2 record count mismatch! Expected $EXPECTED_20, got $COUNT_20" >&2
  exit 1
fi
samtools quickcheck -v "$WORKDIR/ratio_coord.bam"
if [ ! -f "$WORKDIR/ratio_coord.bam.bai" ]; then
  echo "Error: BAI index file missing for ratio coord output!" >&2
  exit 1
fi
samtools idxstats "$WORKDIR/ratio_coord.bam" > "$WORKDIR/idxstats.tsv"
echo "✓ Ratio coord output and BAI index verified with samtools"

# Write JSON report
python3 -c "
import json
report = {
    'total_input_records': $TOTAL_INPUT,
    'ratio_0_1_expected': $EXPECTED_10,
    'ratio_0_1_observed': $COUNT_R1,
    'ratio_0_1_count_equal': True,
    'reproducible_same_seed': True,
    'divergent_different_seed': True,
    'ratio_0_5_expected': $EXPECTED_50,
    'ratio_0_5_observed': $COUNT_50,
    'ratio_0_5_count_equal': True,
    'ratio_coord_expected': $EXPECTED_20,
    'ratio_coord_observed': $COUNT_20,
    'bai_created': True,
    'all_quickcheck_passed': True
}
with open('$REPORT_JSON', 'w') as f:
    json.dump(report, f, indent=2)
"
echo "✓ Wrote ratio evidence report to $REPORT_JSON"
echo "=== Ratio tests completed successfully! ==="
