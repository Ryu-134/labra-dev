locals {
  #  let callers override the lock table name but keep a safe default for quick setup
  resolved_lock_table_name = coalesce(var.lock_table_name, "${var.name_prefix}-terraform-locks")
}

#  create shared state storage so we stop stepping on each other in local state files
resource "aws_s3_bucket" "state" {
  bucket        = var.state_bucket_name
  force_destroy = var.force_destroy
  tags = merge(var.tags, {
    Name    = var.state_bucket_name
    Purpose = "terraform-state"
  })
}

#  keep versioning on so we can recover from bad applies or accidental state edits
resource "aws_s3_bucket_versioning" "state" {
  bucket = aws_s3_bucket.state.id

  versioning_configuration {
    status = "Enabled"
  }
}

#  enforce SSE on state objects because plain-text state in S3 is asking for trouble
resource "aws_s3_bucket_server_side_encryption_configuration" "state" {
  bucket = aws_s3_bucket.state.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

#  hard block public access here because state includes sensitive material
resource "aws_s3_bucket_public_access_block" "state" {
  bucket = aws_s3_bucket.state.id

  block_public_acls       = true
  ignore_public_acls      = true
  block_public_policy     = true
  restrict_public_buckets = true
}

#  create a lock table so concurrent applies do not corrupt shared state
resource "aws_dynamodb_table" "locks" {
  name         = local.resolved_lock_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  #  have to keep this exact key name because Terraform backend locking expects LockID
  attribute {
    name = "LockID"
    type = "S"
  }

  tags = merge(var.tags, {
    Name    = local.resolved_lock_table_name
    Purpose = "terraform-locks"
  })
}
