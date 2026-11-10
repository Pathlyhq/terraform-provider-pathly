output "scenarios_outside_terraform" {
  description = "Scenarios that exist in Pathly but not in this code: created by hand, or left behind."
  value = [
    for s in data.pathly_scenarios.all.scenarios : {
      id     = s.id
      name   = s.name
      folder = s.folder
    }
    if !contains(
      concat(
        [for home in pathly_scenario.home : home.id],
        [for checkout in pathly_scenario.checkout : checkout.id],
      ),
      s.id,
    )
  ]
}

output "scenario_ids" {
  description = "Identifiers of the Ping and the Flow of each client, for a deployment gate or a dashboard."
  value = {
    for key, client in var.clients : key => {
      home     = pathly_scenario.home[key].id
      checkout = pathly_scenario.checkout[key].id
    }
  }
}

output "webhook_fingerprint" {
  description = "Fingerprint of the destination: changes if someone edits the URL outside Terraform."
  value       = pathly_webhook.alerts.url_fingerprint
}
