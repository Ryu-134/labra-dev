output "bucket_name" {
  value = aws_s3_bucket.site.id
}

output "bucket_arn" {
  value = aws_s3_bucket.site.arn
}

output "distribution_id" {
  value = aws_cloudfront_distribution.site.id
}

output "distribution_arn" {
  value = aws_cloudfront_distribution.site.arn
}

output "distribution_domain_name" {
  value = aws_cloudfront_distribution.site.domain_name
}

output "site_url" {
  value = "https://${aws_cloudfront_distribution.site.domain_name}"
}

output "release_prefix" {
  value = var.release_prefix
}

output "alarm_names" {
  value = compact([try(aws_cloudwatch_metric_alarm.cloudfront_5xx_rate[0].alarm_name, null)])
}
