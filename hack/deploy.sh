#!/bin/bash

set -eEu

TF_OUTPUT_FILE="tf_output.json"

terraform::output::todoDynamoDBTableName() {
  cat "${TF_OUTPUT_FILE}" | jq -r '.todo_list_dynamodb_table_name.value'
}

terraform::output::scheduleDynamoDBTableName() {
  cat "${TF_OUTPUT_FILE}" | jq -r '.schedules_dynamodb_table_name.value'
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
  local helm_flags="--set bot.todo.dynamoDBTableName=$(terraform::output::todoDynamoDBTableName)"
  helm_flags="${helm_flags} --set bot.schedule.dynamoDBTableName=$(terraform::output::scheduleDynamoDBTableName)"

  helm::upgrade "${helm_flags}"
}

main $@
