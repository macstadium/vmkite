#!/bin/bash

set -e
set -o pipefail
set -u

export org_slug=macstadium
export pipeline_slug=vmkite-test

curl -X POST "https://api.buildkite.com/v2/organizations/${org_slug}/pipelines/${pipeline_slug}/builds" \
  --show-error \
  --fail \
  -H "Authorization: Bearer $VMKITE_BUILDKITE_API_TOKEN" \
  -d @- << JSON
{
  "commit": "HEAD",
  "branch": "master",
  "message": "Testing vmkite build :rocket: :llama:",
  "meta_data": {
    "vmkite-vmdk": "${VMKITE_SOURCE_PATH}",
    "vmkite-guestid": "${VMKITE_VM_GUEST_TYPE}"
  }
}
JSON