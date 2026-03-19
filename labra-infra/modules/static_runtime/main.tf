data "aws_caller_identity" "current" {}

data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}

locals {
  #  derive a deterministic bucket name so backend and frontend can rely on stable naming without hardcoding
  default_bucket_name = substr(
    lower(replace("${var.name_prefix}-${data.aws_caller_identity.current.account_id}-site", "_", "-")),
    0,
    63
  )
  site_bucket_name = coalesce(var.bucket_name, local.default_bucket_name)
  origin_id        = "${var.name_prefix}-static-origin"
  module_tags = merge(var.tags, {
    AppName   = var.app_name
    BuildType = "static"
  })
}

#  store static build output and release snapshots in this bucket
resource "aws_s3_bucket" "site" {
  bucket        = local.site_bucket_name
  force_destroy = var.force_destroy

  tags = merge(local.module_tags, {
    Name = local.site_bucket_name
  })
}

resource "aws_s3_bucket_versioning" "site" {
  bucket = aws_s3_bucket.site.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "site" {
  bucket = aws_s3_bucket.site.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

#  keep this bucket private and only let CloudFront read through OAC
resource "aws_s3_bucket_public_access_block" "site" {
  bucket = aws_s3_bucket.site.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_cloudfront_origin_access_control" "site" {
  name                              = "${var.name_prefix}-static-oac"
  description                       = "OAC for ${var.name_prefix} static site bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

#  expose this distribution as the live URL surface both of you show and consume in your flows
resource "aws_cloudfront_distribution" "site" {
  enabled             = true
  default_root_object = var.default_root_object
  price_class         = var.price_class
  wait_for_deployment = false

  origin {
    domain_name              = aws_s3_bucket.site.bucket_regional_domain_name
    origin_id                = local.origin_id
    origin_access_control_id = aws_cloudfront_origin_access_control.site.id

    s3_origin_config {
      origin_access_identity = ""
    }
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD", "OPTIONS"]
    cache_policy_id        = data.aws_cloudfront_cache_policy.caching_optimized.id
    target_origin_id       = local.origin_id
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
  }

  dynamic "custom_error_response" {
    for_each = var.enable_spa_routing ? toset([403, 404]) : toset([])

    content {
      error_code            = custom_error_response.value
      response_code         = 200
      response_page_path    = "/index.html"
      error_caching_min_ttl = 60
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = merge(local.module_tags, {
    Name = "${var.name_prefix}-static-cdn"
  })
}

data "aws_iam_policy_document" "site_bucket_policy" {
  statement {
    sid    = "AllowCloudFrontReadOnly"
    effect = "Allow"
    actions = [
      "s3:GetObject"
    ]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    resources = ["${aws_s3_bucket.site.arn}/*"]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.site.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "site" {
  bucket = aws_s3_bucket.site.id
  policy = data.aws_iam_policy_document.site_bucket_policy.json
}

#  keep lifecycle rules scoped to our release prefix so old release data cleans up without manual work
resource "aws_s3_bucket_lifecycle_configuration" "site" {
  bucket = aws_s3_bucket.site.id

  rule {
    id     = "release-history-retention"
    status = "Enabled"

    filter {
      prefix = var.release_prefix
    }

    expiration {
      days = var.release_retention_days
    }

    noncurrent_version_expiration {
      noncurrent_days = var.noncurrent_retention_days
    }
  }

  rule {
    id     = "abort-incomplete-multipart-uploads"
    status = "Enabled"

    filter {
      prefix = ""
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "cloudfront_5xx_rate" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.name_prefix}-static-cf-5xx-rate"
  alarm_description   = "CloudFront 5xx error rate above threshold"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  threshold           = var.cf_5xx_rate_threshold
  metric_name         = "5xxErrorRate"
  namespace           = "AWS/CloudFront"
  period              = var.alarm_period_seconds
  statistic           = "Average"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DistributionId = aws_cloudfront_distribution.site.id
    Region         = "Global"
  }
}
