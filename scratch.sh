#!/bin/bash

set -euo pipefail

source .env

FOLDER="/MacStadium - Vegas/vm"

list_vms() {
  govc ls -json "$FOLDER/*" | jq '.elements[0]'
}

list_vms
