#!/bin/bash

set -eEu


helm::upgrade() {
  local helm_flags="$1"

  helm upgrade --install \
    --namespace discord-bots \
    --create-namespace \
    --wait \
    --timeout 5m \
    ${helm_flags} \
    organizer-bot \
    ./helm
}

main() {
  local helm_flags="--set redis.auth.password=password"

  helm::upgrade "${helm_flags}"
}

main $@
