#!/bin/bash
set -euo pipefail

# this script verifies, that all stylesheet identifiers (which certainly are
# subject to change, especially the addition of new identifiers) are present in
# the stylesheet internal data type (twice, for the declaration and assignment
# in constructor)

source .scripts/lint/helpers.sh

enumerate_stylesheet_identifiers \
  | while read identifier yaml_identifier
    do
      n_definitions=$(cat internal/styling/stylesheet.go | grep "^\s*${identifier}\s\+DrawStyling$" | wc -l)
      if [ "${n_definitions}" -ne "1" ]
      then
        echo "ERROR: stylesheet component '${identifier}' is not defined (exactly once, instead found ${n_definitions} times) in stylesheet.go"
        exit 1
      fi
      n_assigns=$(cat internal/styling/stylesheet.go | grep "^\s*stylesheet\.${identifier}\s*=\s*.*$" | wc -l)
      if [ "${n_assigns}" -ne "1" ]
      then
        # NOTE: Admittedly, this assumes the identifier used for an in-construction stylesheet is 'stylesheet'; brittle.
        echo "ERROR: stylesheet component '${identifier}' is not assigned (exactly once, instead found ${n_assigns} times) in stylesheet.go"
        exit 1
      fi
    done || exit 1

echo "SUCCESS: all stylesheet identifiers present in stylesheet.go"
exit 0
