#!/bin/bash

set -euo pipefail

export GOVC_HOST="10.92.157.10"
export GOVC_USERNAME="administrator@ljd.cc"
export GOVC_PASSWORD="$GOVC_PASSWORD"
export GOVC_INSECURE="true"

export GOVC_URL="https://$GOVC_HOST/sdk"

govc 2>&1 ls -l
