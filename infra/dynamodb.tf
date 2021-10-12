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
