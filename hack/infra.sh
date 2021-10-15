#!/bin/bash

set -eEu

INFRA_DIR="infra"

help() {
  echo "Usage:"
  echo "  infra.sh plan <environment>"
  echo "  infra.sh apply <environment>"
  echo "  infra.sh destroy <environment>"
  exit 1
}

terraform::init() {
  local -r env="$1"
  local -r config="vars/${env}.backend.tfvars"

  terraform -chdir="${INFRA_DIR}" init -reconfigure -backend-config="${config}"
}

terraform::plan() {
  local -r env="$1"
  local -r config="vars/${env}.tfvars"

  terraform -chdir="${INFRA_DIR}" plan -var-file="${config}"
}

terraform::apply() {
  local -r env="$1"
  local -r config="vars/${env}.tfvars"

  terraform -chdir="${INFRA_DIR}" apply -auto-approve -var-file="${config}"
}

terraform::output() {
  local -r env="$1"
  local -r output_file="${2:-}"

  local -r cmd="terraform -chdir="${INFRA_DIR}" output -json"

  if [ -n "${output_file}" ]; then
    ${cmd} | tee "${output_file}"
  else
    ${cmd}
  fi
}

main() {
  local -r action="${1:-}"
  local -r env="${2:-devel}"

  case "${action}" in
    plan)
      terraform::init "${env}"
      terraform::plan "${env}"
      ;;

    apply)
      terraform::init "${env}"
      terraform::apply "${env}"
      ;;

    output)
      terraform::init "${env}"
      terraform::output "${env}" tf_output.json
      ;;

    *)
      help
      ;;

  esac
}


main $@
