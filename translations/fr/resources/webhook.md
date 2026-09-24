---
page_title: "pathly_webhook"
description: |-
  Webhook sortant signé, appelé quand un scénario échoue et quand il se rétablit.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/webhook.md`](../../../docs/resources/webhook.md), seule
> version lue par le registre Terraform. En cas d'écart entre les deux, la
> version anglaise prévaut. Une traduction périmée qui annoncerait de mauvaises
> portées d'API conduirait à créer une clé trop privilégiée.

# pathly_webhook

Webhook sortant. Chaque livraison est signée en HMAC avec le secret renvoyé à
la création : vérifiez cette signature côté réception, faute de quoi n'importe
qui peut forger une fausse alerte.

## Exemple

```terraform
resource "pathly_webhook" "alerts" {
  url    = "https://hooks.example.com/pathly"
  events = ["run.failed", "run.recovered"]
}
```

## Schéma

### Obligatoire

- `url` (String, sensible) — Destination, en https et publique. **Force le
  remplacement.** Les adresses privées, les domaines internes et les IP de
  métadonnées cloud sont refusés : le worker appelle depuis le réseau de la
  plateforme, et un webhook pointant sur `169.254.169.254` lui ferait lire ses
  propres identifiants.

### Optionnel

- `events` (List of String) — `run.failed` et `run.recovered`. Les deux par
  défaut. **Force le remplacement.**

### Calculé

- `id` (String) — Identifiant attribué par Pathly.
- `enabled` (Boolean) — Un webhook désactivé depuis la console reste dans
  l'état, sans être appelé.
- `url_fingerprint` (String) — Empreinte de la destination. Elle change si
  l'URL a été modifiée ailleurs : c'est le seul moyen de le voir, puisque l'API
  ne renvoie jamais l'URL stockée.
- `secret` (String, sensible) — Secret de signature HMAC, renvoyé **une seule
  fois**, à la création.
- `created_at` (String) — Date de création.

## Le secret et le fichier d'état

Puisque le secret n'est renvoyé qu'à la création, Terraform doit le conserver :
il est donc dans l'état. Trois conséquences à traiter :

1. **Chiffrez le backend** (`azurerm` avec chiffrement au repos, `gcs` avec
   CMEK, ou un équivalent). Un état en clair dans un bucket public expose le
   secret.
2. **Ne le publiez pas dans un `output` non sensible** : il apparaîtrait dans
   les logs de CI.
3. **Pour le déplacer dans un coffre**, lisez-le une fois puis retirez
   l'output :

```terraform
output "webhook_secret" {
  value     = pathly_webhook.alerts.secret
  sensitive = true
}
```

```sh
terraform output -raw webhook_secret | az keyvault secret set --vault-name … --name pathly-webhook --value @-
```

## Import

Préférez un bloc `import` (Terraform / OpenTofu ≥ 1.5) ; la forme CLI est
équivalente.

```terraform
import {
  to = pathly_webhook.alerts
  id = "wh_01H8ZK…"
}
```

```sh
terraform import pathly_webhook.alerts wh_01H8ZK…
```

~> L'import émet un avertissement : l'API ne renvoie ni l'URL ni le secret.
Redéclarez `url` dans la configuration, sinon le plan proposerait un
remplacement, et récupérez le secret depuis votre coffre, puisqu'il n'a été
affiché qu'à la création.
