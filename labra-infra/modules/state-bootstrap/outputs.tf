output "state_bucket_name" {
  description = "S3 bucket name used for Terraform remote state."
  value       = aws_s3_bucket.state.id
}

output "state_bucket_arn" {
  description = "S3 bucket ARN for IAM policy wiring."
  value       = aws_s3_bucket.state.arn
}

output "lock_table_name" {
  description = "DynamoDB table name used for Terraform state locking."
  value       = aws_dynamodb_table.locks.name
}

output "lock_table_arn" {
  description = "DynamoDB lock table ARN."
  value       = aws_dynamodb_table.locks.arn
}
