#!/bin/bash

set -eEu

TF_OUTPUT_FILE="tf_output.json"

terraform::output::todoDynamoDBTableName() {
  cat "${TF_OUTPUT_FILE}" | jq -r '.todo_list_dynamodb_table_name.value'
}

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
  local -r todoDynamoDBTableName=$(terraform::output::todoDynamoDBTableName)

  local helm_flags="--set bot.todo.dynamoDBTableName=${todoDynamoDBTableName}"

  helm::upgrade "${helm_flags}"
}

main $@
