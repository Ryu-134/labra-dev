output "bucket_name" {
  description = "Static site S3 bucket name"
  value       = aws_s3_bucket.site.id
}

output "bucket_arn" {
  description = "Static site S3 bucket ARN"
  value       = aws_s3_bucket.site.arn
}

output "distribution_id" {
  description = "CloudFront distribution ID"
  value       = aws_cloudfront_distribution.site.id
}

output "distribution_arn" {
  description = "CloudFront distribution ARN"
  value       = aws_cloudfront_distribution.site.arn
}

output "distribution_domain_name" {
  description = "CloudFront domain name used as public endpoint"
  value       = aws_cloudfront_distribution.site.domain_name
}

output "site_url" {
  description = "Public HTTPS URL for static site"
  value       = "https://${aws_cloudfront_distribution.site.domain_name}"
}

output "release_prefix" {
  description = "S3 prefix used for versioned release artifacts"
  value       = var.release_prefix
}

output "alarm_names" {
  description = "Static runtime alarm names created by this module"
  value = compact([
    try(aws_cloudwatch_metric_alarm.cloudfront_5xx_rate[0].alarm_name, null)
  ])
}
