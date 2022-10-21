#!/bin/bash
set -euo pipefail

# this script verifies, that all stylesheet identifiers (which certainly are
# subject to change, especially the addition of new identifiers) have 2 defaults
# (2 == count(light, dark)) defined.

source .scripts/lint/helpers.sh

enumerate_stylesheet_identifiers \
  | while read identifier yaml_identifier
    do
      n_usages_in_assignment=$(cat internal/styling/stylesheet.go | grep "config\.${identifier}\>" | wc -l)
      if [ "${n_usages_in_assignment}" -ne "1" ]
      then
        echo "ERROR: stylesheet component '${identifier}' is not used in exactly one assignment"
        exit 1
      fi
    done || exit 1

echo "SUCCESS: all stylesheet identifiers used in assignments as expected"
exit 0
