#!/bin/bash

set -e
set -o pipefail
set -u

method="$1"
subpath="$2"
shift
shift

url="https://api.buildkite.com/v2/organizations/macstadium$subpath"

echo >&2 "$method $url $@"

curl "$url" \
  --silent \
  --show-error \
  --globoff \
  -X "$method" \
  -H "Authorization: Bearer $BUILDKITE_TOKEN" \
  "$@"
