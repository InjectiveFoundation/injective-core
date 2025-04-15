#!/bin/bash

ulimit -n 120000

# Default INJHOME to current directory if not set
if [ -z "$INJHOME" ]; then
  INJHOME="$HOME/.injectived"
  echo "INJHOME not set, defaulting to HOME directory: $INJHOME"
fi

yes 12345678 | injectived \
--home "$INJHOME" \
$*
