#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <ictest_output_dir_1> <ictest_output_dir_2>"
    echo "Example: $0 interchaintest/coverage/TestBasicInjectiveStart/val0 interchaintest/coverage/TestBasicInjectiveStart/val1"
    exit 1
fi

coverage_path_1="$1"
coverage_path_2="$2"

dir_name1=$(basename "$coverage_path_1")
dir_name2=$(basename "$coverage_path_2")


mkdir -p "$(dirname "$coverage_path_1")-diff"
go tool covdata subtract -i "$coverage_path_1","$coverage_path_2" -o "$(dirname "$coverage_path_1")-diff"

go tool covdata textfmt -i "$(dirname "$coverage_path_1")-diff" -o "$coverage_path_1.diff.merged.out"

grep -v " 0$" "$coverage_path_1.diff.merged.out" > "$coverage_path_1.diff.merged.filtered.out"

output_file="$(dirname "$coverage_path_1")-diff.html"
go tool cover -html="$coverage_path_1.diff.merged.filtered.out" -o "$output_file"

rm "$coverage_path_1.diff.merged.out"
rm "$coverage_path_1.diff.merged.filtered.out"
rm -rf "$(dirname "$coverage_path_1")-diff"
