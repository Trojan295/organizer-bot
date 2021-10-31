#!/bin/bash

set -eEu


helm::upgrade() {
  local release_name="${1}"
  local helm_flags="${2}"

  helm upgrade --install \
    --namespace discord-bots \
    --create-namespace \
    --atomic \
    --timeout 5m \
    ${helm_flags} \
    "${release_name}" \
    ./helm/chart
}

main() {
  local -r env="${1}"

  local release_name="organizer-bot-${env}"
  local helm_flags="-f helm/envs/values.yaml -f helm/envs/${env}/values.yaml"

  if [ -n "${DOCKER_IMAGE_TAG:-}" ]; then
    helm_flags="${helm_flags} --set image.tag=${DOCKER_IMAGE_TAG}"
  fi

  helm::upgrade "${release_name}" "${helm_flags}"
}

main $@
