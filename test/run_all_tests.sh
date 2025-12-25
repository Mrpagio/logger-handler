#!/usr/bin/env bash
# Run all go tests in the repository and write output to a timestamped file in this directory
# Filename format: <YY-MM-DD-hh-mm>_test
# Usage: ./run_all_tests.sh

set -uo pipefail

# Resolve script directory (this file's directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Write output into the "output" subdirectory under the test directory
OUT_DIR="$SCRIPT_DIR/output"
TIMESTAMP="$(date +'%y-%m-%d-%H-%M')"
OUTFILE="$OUT_DIR/${TIMESTAMP}_test"

# Ensure output dir exists (should be test/output)
mkdir -p "$OUT_DIR"

# Move to repository root (two levels up from test/output -> repo root)
cd "$SCRIPT_DIR/../.." || { echo "Failed to cd to repo root" >&2; exit 2; }

echo "Running: go test -v ./..." | tee "$OUTFILE"
# Run tests and append both stdout and stderr to the outfile
if go test -v ./... >> "$OUTFILE" 2>&1; then
  RC=0
else
  RC=$?
fi

echo "Exit code: $RC" >> "$OUTFILE"

echo "Wrote test output to: $OUTFILE"
exit $RC
