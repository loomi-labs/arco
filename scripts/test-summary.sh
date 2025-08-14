#!/bin/bash

# Test Summary Generator - Generate final test summary from go test -json output
# Usage: go test -json ... | ./scripts/test-summary.sh
# 
# Processes JSON test output and generates a comprehensive final summary
# with test counts, timing, and overall coverage statistics.

jq -sr '
# Calculate final summary
. as $all_events |
($all_events | map(select(.Action == "pass" or .Action == "fail") | select(.Test != null))) as $test_events |
($test_events | map(select(.Action == "pass")) | length) as $passed_tests |
($test_events | map(select(.Action == "fail")) | length) as $failed_tests |
($test_events | map(.Elapsed // 0) | add) as $total_time |

# Extract coverage information per package
($all_events | map(select(.Action == "output" and (.Output // "" | type == "string") and (.Output | test("coverage: [0-9.]+%")))) | 
 group_by(.Package) |
 map({
   package: .[0].Package,
   coverage: (.[0].Output | match("coverage: ([0-9.]+)%") | .captures[0].string | tonumber)
 })) as $package_coverage |

# Calculate overall coverage
($package_coverage | map(.coverage) | if length > 0 then (add / length) else null end) as $overall_coverage |

"\n" + "=" * 60 + "\n" +
"📊 Final Summary: " + ($passed_tests | tostring) + " passed, " + 
($failed_tests | tostring) + " failed (" + ($total_time | . * 100 | round / 100 | tostring) + "s total)" +
(if $overall_coverage then " | Coverage: " + ($overall_coverage | . * 100 | round / 100 | tostring) + "%" else "" end) +
"\n" + "=" * 60
'