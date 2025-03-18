#!/bin/bash

if [[ "$DO_COVERAGE" != "true" && "$DO_COVERAGE" != "yes" ]]; then
    exit 0
fi

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <ictest_output_dir>"
    echo "Example: $0 interchaintest/coverage/TestBasicInjectiveStart"
    exit 1
fi

coverage_path="$1"

# Iterate through each directory in the coverage path
for dir in "$coverage_path"/*; do
    if [ -d "$dir" ]; then
        dir_name=$(basename "$dir")

        # Create merged coverage data for this directory
        go tool covdata textfmt -i "$dir" -o "$dir.merged.out"

        # Generate HTML coverage report
        output_file="$(dirname "$coverage_path")/${dir_name}.html"
        go tool cover -html="$dir.merged.out" -o "$output_file"

        # Cleanup temporary merged coverage file
        rm "$dir.merged.out"
    fi
done
