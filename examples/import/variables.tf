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
  description = "Existing scenario id (mon_…)."
  type        = string
  default     = "mon_REPLACE_ME"
}

variable "import_webhook_id" {
  description = "Existing webhook id (wh_…)."
  type        = string
  default     = "wh_REPLACE_ME"
}

variable "import_maintenance_window_id" {
  description = "Existing maintenance window id (mw_…)."
  type        = string
  default     = "mw_REPLACE_ME"
}

variable "import_sla_target_id" {
  description = "Existing SLA target id (sla_…)."
  type        = string
  default     = "sla_REPLACE_ME"
}
