variable "clients" {
  description = <<-EOT
    Client sites, one entry each. The key is a short identifier: it ends up in
    the "client:<key>" tag and in the resource addresses, so keep it stable.
  EOT
  type = map(object({
    # Display name, also used as the Pathly folder.
    name = string
    # Home page watched by the Ping, and a text it must contain.
    home_url  = string
    home_text = optional(string)
    # Product page where the Flow starts, and the checkout page where it stops.
    product_url          = string
    checkout_url         = string
    add_to_cart_selector = optional(string, "button[name=add-to-cart]")
    checkout_text        = optional(string, "Payment")
    # Set only if the checkout needs a signed-in customer. The account itself
    # goes in var.test_logins, under the same key.
    login_url     = optional(string)
    severity      = optional(string, "major")
    objective_pct = optional(number, 99.9)
  }))

  validation {
    condition     = alltrue([for key, client in var.clients : can(regex("^[a-z0-9][a-z0-9-]*$", key))])
    error_message = "A client key uses lowercase letters, digits and hyphens: it becomes a tag."
  }

  validation {
    # The API refuses an unknown severity: saying it here names the rule.
    condition = alltrue([
      for key, client in var.clients : contains(["critical", "major", "minor"], client.severity)
    ])
    error_message = "severity is critical, major or minor."
  }

  validation {
    condition = alltrue([
      for key, client in var.clients : client.objective_pct >= 50 && client.objective_pct <= 100
    ])
    error_message = "objective_pct is a percentage, between 50 and 100."
  }

  validation {
    condition = alltrue([
      for key, client in var.clients : alltrue([
        for url in [client.home_url, client.product_url, client.checkout_url, client.login_url] :
        url == null || startswith(url, "https://")
      ])
    ])
    error_message = "Every URL of a client starts with https://."
  }
}

variable "test_logins" {
  description = <<-EOT
    Test accounts, keyed like var.clients. Only for the clients that set
    login_url. Use an account created for monitoring, never a real customer,
    and supply the values from a secret store (TF_VAR_test_logins), not from a
    committed file. Kept apart from var.clients because a sensitive map cannot
    drive for_each.
  EOT
  type = map(object({
    username = string
    password = string
  }))
  sensitive = true
  default   = {}
}

variable "webhook_url" {
  description = <<-EOT
    Alert destination shared by every client, over https and reachable from the
    public internet. The API refuses private addresses.
  EOT
  type        = string

  validation {
    condition     = startswith(var.webhook_url, "https://")
    error_message = "The webhook must use https: the payload is signed, not encrypted by the application."
  }
}

variable "ping_interval_sec" {
  description = "Period between two Pings of a home page, in seconds."
  type        = number
  default     = 300
}

variable "flow_interval_sec" {
  description = "Period between two runs of a checkout Flow, in seconds. A browser run costs more than a Ping: pick what your plan allows."
  type        = number
  default     = 3600
}

variable "max_latency_ms" {
  description = "Above this, a Ping of a home page is a performance failure."
  type        = number
  default     = 3000
}
