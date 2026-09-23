variable "environment" {
  description = "Tag applied to every scenario, and filter for the data source."
  type        = string
  default     = "prod"
}

variable "folder" {
  description = "Pathly folder that groups the scenarios."
  type        = string
  default     = "Shop"
}

variable "objective_pct" {
  description = "Availability target over 30 days, for critical journeys."
  type        = number
  default     = 99.9

  validation {
    condition     = var.objective_pct >= 50 && var.objective_pct <= 100
    error_message = "The objective is a percentage, between 50 and 100."
  }
}

variable "journeys" {
  description = "Monitored journeys, described once and for all."
  type = map(object({
    name           = string
    url            = string
    interval_sec   = number
    expect_text    = optional(string)
    max_latency_ms = optional(number)
    severity       = optional(string, "major")
    tags           = optional(list(string), [])
    runbook        = optional(string)
  }))

  default = {
    home = {
      name           = "Home page"
      url            = "https://shop.example.com/"
      interval_sec   = 300
      expect_text    = "Our products"
      max_latency_ms = 2000
      severity       = "major"
    }
    checkout = {
      name           = "Checkout funnel"
      url            = "https://shop.example.com/cart"
      interval_sec   = 60
      expect_text    = "Your cart"
      max_latency_ms = 1500
      severity       = "critical"
      tags           = ["payment"]
      runbook        = "https://wiki.example.com/ops/checkout-unavailable"
    }
  }

  validation {
    # The API refuses an unknown severity: saying it here names the offending key.
    condition = alltrue([
      for key, j in var.journeys : contains(["critical", "major", "minor"], j.severity)
    ])
    error_message = "severity is critical, major or minor."
  }
}

variable "webhook_url" {
  description = <<-EOT
    Alert destination, over https and reachable from the public internet.
    Private addresses and cloud metadata IPs are refused by the API: the worker
    calls from the platform network, and a webhook pointing at 169.254.169.254
    would make it read its own credentials.
  EOT
  type        = string
  default     = "https://hooks.example.com/pathly"

  validation {
    condition     = startswith(var.webhook_url, "https://")
    error_message = "The webhook must use https: the payload is signed, not encrypted by the application."
  }
}

variable "migration_starts_at" {
  description = "Start of the migration window, in ISO 8601. Null to create none."
  type        = string
  default     = null
}

variable "migration_ends_at" {
  description = "End of the migration window, in ISO 8601."
  type        = string
  default     = null

  validation {
    condition     = var.migration_starts_at == null || var.migration_ends_at != null
    error_message = "A one-off window needs both of its bounds."
  }
}
