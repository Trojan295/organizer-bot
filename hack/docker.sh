#!/bin/bash

set -eEu


main() {
  local -r action="$1"

  local -r docker_registry="rg.fr-par.scw.cloud"
  local -r docker_repository="discordbots/organizer-bot"
  local -r docker_tag="${DOCKER_TAG:-latest}"

  local -r docker_image="${docker_registry}/${docker_repository}:${docker_tag}"

  case "${action}" in
    build)
      docker build -t "${docker_image}" .
      ;;

    push)
      docker login "${docker_registry}" -u "${SCW_ACCESS_KEY}" -p "${SCW_SECRET_KEY}"
      docker push "${docker_image}"
      ;;
  esac

}

main $@
