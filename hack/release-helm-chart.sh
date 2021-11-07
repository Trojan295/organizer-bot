#!/bin/bash

set -eEu

SNAPSHOT="${SNAPSHOT:-}"
CHARTMUSEUM_ENDPOINT="https://charts.myhightech.org"

chart::version() {
  helm show chart helm/chart | grep -E '^version:' | awk -F ' ' '{print $2}'
}

helm::package() {
  local helm_opts=""

  if [ -n "${SNAPSHOT}" ]; then
    local -r version="$(chart::version)"
    helm_opts="${helm_opts} --version ${version}-SNAPSHOT"
  fi

  helm package helm/chart $helm_opts
}

chartmuseum::push() {
  local version="$(chart::version)"
  if [ -n "${SNAPSHOT}" ]; then
    version="${version}-SNAPSHOT"

    curl ${CHARTMUSEUM_ENDPOINT}/api/charts/organizer-bot/${version} \
      -u "${CHARTMUSEUM_AUTH_USER}:${CHARTMUSEUM_AUTH_PASS}" \
      -XDELETE
  fi

  local -r post_resp=$(curl ${CHARTMUSEUM_ENDPOINT}/api/charts \
    -u "${CHARTMUSEUM_AUTH_USER}:${CHARTMUSEUM_AUTH_PASS}" \
    --data-binary "@organizer-bot-${version}.tgz")
  echo $post_resp
  echo "$post_resp" | jq -er ".saved"
}

main() {
  helm::package
  chartmuseum::push
}

main $@
