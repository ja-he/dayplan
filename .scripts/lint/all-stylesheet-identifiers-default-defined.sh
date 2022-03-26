#!/bin/bash

# this script verifies, that all stylesheet identifiers (which certainly are
# subject to change, especially the addition of new identifiers) have 2 defaults
# (2 == count(light, dark)) defined.

cat src/config/config.go \
  | sed -n '/type Stylesheet struct/,/}/p' \
  | grep 'Styling' \
  | sed 's/^\s*\([A-Z][a-zA-Z]\+\)\s\+Styling\s\+`yaml:"\([a-z-]*\)"`\s*$/\1 \2/' \
  | while read identifier yaml_identifier
    do
      n_default_defs=$(cat src/config/default.go | grep "\<${identifier}\>" | wc -l)
      if [ "${n_default_defs}" -ne "2" ]
      then
        echo "ERROR: stylesheet component '${identifier}' is not defined exactly twice in the defaults"
        exit 1
      fi
    done || exit 1

echo "SUCCESS: all stylesheet identifiers default-defined"
exit 0
