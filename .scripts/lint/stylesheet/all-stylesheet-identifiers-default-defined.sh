#!/bin/bash
set -euo pipefail

# this script verifies, that all stylesheet identifiers (which certainly are
# subject to change, especially the addition of new identifiers) have 2 defaults
# (2 == count(light, dark)) defined.

source .scripts/lint/helpers.sh

enumerate_stylesheet_identifiers \
  | while read identifier yaml_identifier
    do
      n_default_defs=$(cat internal/config/default.go | grep "\<${identifier}\>" | wc -l)
      if [ "${n_default_defs}" -ne "2" ]
      then
        echo "ERROR: stylesheet component '${identifier}' is not defined exactly twice in the defaults"
        exit 1
      fi
    done || exit 1

echo "SUCCESS: all stylesheet identifiers default-defined"
exit 0
