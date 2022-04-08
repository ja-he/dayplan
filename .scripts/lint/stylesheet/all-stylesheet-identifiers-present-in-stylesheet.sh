#!/bin/bash

# this script verifies, that all stylesheet identifiers (which certainly are
# subject to change, especially the addition of new identifiers) are present in
# the stylesheet internal data type (twice, for the declaration and assignment
# in constructor)

source .scripts/lint/helpers.sh

enumerate_stylesheet_identifiers \
  | while read identifier yaml_identifier
    do
      n_default_defs=$(cat src/styling/stylesheet.go | grep "\<${identifier}\>" | wc -l)
      if [ "${n_default_defs}" -ne "2" ]
      then
				echo "ERROR: stylesheet component '${identifier}' is not used (exactly twice) in stylesheet.go"
        exit 1
      fi
    done || exit 1

echo "SUCCESS: all stylesheet identifiers present in stylesheet.go"
exit 0
