#!/bin/bash

# Test Output Processor - Process go test -json output for real-time display
# Usage: go test -json ... | ./scripts/test-output-processor.sh
# 
# Processes streaming JSON test output and displays clean, formatted results
# with correct package-to-coverage association.

# Process JSON events and convert to structured format, then handle with bash
jq --unbuffered -r '
  if .Action == "pass" and .Test == null then
    "PACKAGE_PASS:" + .Package + ":" + (.Elapsed // 0 | . * 100 | round / 100 | tostring)
  elif .Action == "fail" and .Test == null then
    "PACKAGE_FAIL:" + .Package + ":" + (.Elapsed // 0 | . * 100 | round / 100 | tostring)
  elif .Action == "output" and (.Output // "" | type == "string") and (.Output | test("^coverage: [0-9.]+% of statements$")) then
    "COVERAGE:" + .Package + ":" + (.Output | match("coverage: ([0-9.]+)%") | .captures[0].string)
  else
    empty
  end
' | {
  # Bash associative arrays
  declare -A packages
  declare -A coverage
  declare -A printed
  
  while IFS= read -r line; do
    if [[ "$line" == "PACKAGE_PASS:"* ]]; then
      # Extract package name and elapsed time
      pkg_info="${line#PACKAGE_PASS:}"
      pkg_name="${pkg_info%:*}"
      elapsed="${pkg_info##*:}"
      
      # Skip if already printed
      if [[ -n "${printed[$pkg_name]}" ]]; then
        continue
      fi
      
      package_line="✅ $pkg_name (${elapsed}s)"
      
      # Check if we have coverage for this package
      if [[ -n "${coverage[$pkg_name]}" ]]; then
        printf "%s | %s%%\n" "$package_line" "${coverage[$pkg_name]}"
        unset coverage["$pkg_name"]
      else
        printf "%s\n" "$package_line"
      fi
      
      # Mark as printed
      printed["$pkg_name"]=1
      
    elif [[ "$line" == "PACKAGE_FAIL:"* ]]; then
      # Extract package name and elapsed time
      pkg_info="${line#PACKAGE_FAIL:}"
      pkg_name="${pkg_info%:*}"
      elapsed="${pkg_info##*:}"
      
      # Skip if already printed
      if [[ -n "${printed[$pkg_name]}" ]]; then
        continue
      fi
      
      package_line="❌ $pkg_name - FAILED (${elapsed}s)"
      
      # Check if we have coverage for this package
      if [[ -n "${coverage[$pkg_name]}" ]]; then
        printf "%s | %s%%\n" "$package_line" "${coverage[$pkg_name]}"
        unset coverage["$pkg_name"]
      else
        printf "%s\n" "$package_line"
      fi
      
      # Mark as printed
      printed["$pkg_name"]=1
      
    elif [[ "$line" == "COVERAGE:"* ]]; then
      # Extract package name and coverage percentage
      cov_info="${line#COVERAGE:}"
      pkg_name="${cov_info%:*}"
      cov_pct="${cov_info##*:}"
      
      # Skip if already printed
      if [[ -n "${printed[$pkg_name]}" ]]; then
        continue
      fi
      
      # Check if we have package completion for this coverage
      if [[ -n "${packages[$pkg_name]}" ]]; then
        printf "%s | %s%%\n" "${packages[$pkg_name]}" "$cov_pct"
        unset packages["$pkg_name"]
        printed["$pkg_name"]=1
      else
        # Store coverage for later
        coverage["$pkg_name"]="$cov_pct"
      fi
    fi
  done
  
  # Print any remaining packages without coverage
  for pkg_name in "${!packages[@]}"; do
    if [[ -z "${printed[$pkg_name]}" ]]; then
      printf "%s\n" "${packages[$pkg_name]}"
    fi
  done
}