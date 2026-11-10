variable "webhook_url" {
  description = "Webhook destination (must be restated after import; API never returns it)."
  type        = string
  sensitive   = true
  default     = "https://hooks.example.com/pathly"
}

variable "alert_email" {
  description = "Org alert recipient for pathly_settings."
  type        = string
  default     = "ops@example.com"
}

variable "import_scenario_id" {
  description = "Existing scenario id (UUID)."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "import_webhook_id" {
  description = "Existing webhook id (UUID)."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "import_maintenance_window_id" {
  description = "Existing maintenance window id (UUID)."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "import_sla_target_id" {
  description = "Existing SLA target id (UUID)."
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}
