#!/bin/bash

modulesDir="injective-chain/modules"

# Function to parse errors file and generate markdown
parse_errors_file() {
  local filePath=$1
  local moduleName=$2
  local specDir=$3
  local outputFile="${specDir}/99_errors.md"

  mkdir -p "$specDir"

  echo "# Error Codes" > "$outputFile"
  echo "" >> "$outputFile"
  echo "This document lists the error codes used in the module." >> "$outputFile"
  echo "" >> "$outputFile"
  echo "" >> "$outputFile"
  echo "| Module | Error Code | description |" >> "$outputFile"
  echo "|--------|------------|-------------|" >> "$outputFile"

  grep "errors.Register(" "$filePath" | while read -r line; do
    code=$(echo "$line" | sed -n 's/.*errors\.Register([^,]*,\s*\([^,]*\),.*"\([^"]*\)".*/\1/p')
    message=$(echo "$line" | sed -n 's/.*errors\.Register([^,]*,\s*\([^,]*\),.*"\([^"]*\)".*/\2/p')
    if [[ -n "$code" && -n "$message" ]]; then
      echo "| $moduleName | $code | $message |" >> "$outputFile"
    fi
  done
}

# Iterate over each module directory
for moduleDir in "$modulesDir"/*/; do
  if [[ -d "$moduleDir" ]]; then
    moduleName=$(basename "$moduleDir")
    errorsFile="${moduleDir}types/errors.go"
    specDir="${moduleDir}spec"

    if [[ -f "$errorsFile" ]]; then
      parse_errors_file "$errorsFile" "$moduleName" "$specDir"
    fi
  fi
done

echo "Markdown files for modules error codes generated successfully."