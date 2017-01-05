#!/bin/bash

set -e
set -o pipefail
set -u

curl https://api.buildkite.com/v2/organizations/macstadium/pipelines/vmkite-test/builds \
  -X POST \
  -H "Authorization: Bearer $VMKITE_BUILDKITE_API_TOKEN" \
  -F "commit=HEAD" \
  -F "branch=master" \
  -F "message=test build"
