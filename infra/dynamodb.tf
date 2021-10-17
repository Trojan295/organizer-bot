resource "aws_dynamodb_table" "todo_lists" {
  name         = "${local.namespace}-todo-lists"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "ListId"

  attribute {
    name = "ListId"
    type = "S"
  }

  ttl {
    attribute_name = "UpdatedAt"
    enabled        = true
  }
}

resource "aws_dynamodb_table" "reminders" {
  name         = "${local.namespace}-reminders"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "ChannelId"

  attribute {
    name = "ChannelId"
    type = "S"
  }

  ttl {
    attribute_name = "UpdatedAt"
    enabled        = true
  }
}
