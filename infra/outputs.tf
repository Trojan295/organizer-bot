output "todo_list_dynamodb_table_name" {
  value = aws_dynamodb_table.todo_lists.id
}