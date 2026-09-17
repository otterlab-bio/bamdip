#!/usr/bin/env bash
# scripts/compare_bash.sh
# Compare bamdip output with reference bash/samtools implementation.
# Verifies exact record consistency on first-N sampling and samtools quickcheck.

set -euo pipefail

INPUT="${1:-testdata/Test_hg19_NRAS.bam}"
N="${2:-100}"
WORKDIR="$(mktemp -d -t bamdip_test_XXXXXX)"

cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

echo "=== bamdip vs Bash Comparison Test ==="
echo "Input BAM: $INPUT"
echo "Target N:  $N"
echo "Workdir:   $WORKDIR"

if ! command -v samtools >/dev/null 2>&1; then
  echo "Error: samtools is required for this comparison test." >&2
  exit 1
fi

if [ ! -f "$INPUT" ]; then
  echo "Error: input BAM not found: $INPUT" >&2
  exit 1
fi

echo "--- 1. Building bamdip ---"
mkdir -p bin
go build -o bin/bamdip ./cmd

echo "--- 2. Running bamdip (-n $N) ---"
bin/bamdip -n "$N" "$INPUT" "$WORKDIR/bamdip.bam"

echo "--- 3. Running reference Bash pipeline ---"
(
  samtools view -H "$INPUT"
  samtools view "$INPUT" | head -n "$N"
) | samtools view -b -o "$WORKDIR/bash.bam"

echo "--- 4. Validating with samtools quickcheck ---"
samtools quickcheck -v "$WORKDIR/bamdip.bam"
samtools quickcheck -v "$WORKDIR/bash.bam"
echo "✓ Both BAM files passed samtools quickcheck"

echo "--- 5. Comparing record counts ---"
COUNT_BAMDIP=$(samtools view -c "$WORKDIR/bamdip.bam")
COUNT_BASH=$(samtools view -c "$WORKDIR/bash.bam")
echo "bamdip record count: $COUNT_BAMDIP"
echo "bash   record count: $COUNT_BASH"

if [ "$COUNT_BAMDIP" -ne "$N" ] || [ "$COUNT_BASH" -ne "$N" ]; then
  echo "Error: Record count mismatch! Expected $N, got bamdip=$COUNT_BAMDIP, bash=$COUNT_BASH" >&2
  exit 1
fi
echo "✓ Record counts match expected N ($N)"

echo "--- 6. Verifying alignment records field-by-field ---"
samtools view "$WORKDIR/bamdip.bam" > "$WORKDIR/bamdip.sam"
samtools view "$WORKDIR/bash.bam" > "$WORKDIR/bash.sam"

if ! diff -u "$WORKDIR/bash.sam" "$WORKDIR/bamdip.sam"; then
  echo "Error: Alignment records differ between bamdip and bash implementation!" >&2
  exit 1
fi
echo "✓ Alignment records are 100% IDENTICAL"

echo "--- 7. Testing Coord-sorted with BAI and samtools quickcheck ---"
bin/bamdip -n "$N" -sort coord -index "$INPUT" "$WORKDIR/bamdip_coord.bam"
samtools quickcheck -v "$WORKDIR/bamdip_coord.bam"
if [ ! -f "$WORKDIR/bamdip_coord.bam.bai" ]; then
  echo "Error: BAI index file missing!" >&2
  exit 1
fi
echo "✓ Coordinate sorted output and BAI index verified with samtools"

echo "=== All checks passed successfully! ==="
