# Pathly Terraform Provider

[English](README.md) · **Français** · [Español](README.es.md)


[![CI](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml/badge.svg)](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml)
[![Powered by Pathly](https://img.shields.io/badge/Powered%20by-Pathly-0B5FFF?style=flat-square)](https://pathlyhq.com)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827?style=flat-square)](https://pathlyhq.com)
[![API docs](https://img.shields.io/badge/API-developers-2563eb?style=flat-square)](https://pathlyhq.com/fr/developers)
[![Start free](https://img.shields.io/badge/Solo-start%20free-16a34a?style=flat-square)](https://pathlyhq.com/fr/login?mode=signup)

**Pour utiliser ce provider, vous avez besoin d’une clé API. Obtenez votre clé gratuite en vous inscrivant ici : [https://pathlyhq.com/fr/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta](https://pathlyhq.com/fr/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta).**

> **Démarrage en un clic.** Créez un compte gratuit sur [Pathly](https://pathlyhq.com) ([inscription](https://pathlyhq.com/fr/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta)), générez une clé API dans la console, puis exportez `PATHLY_API_TOKEN`. Ce dépôt est le pont officiel vers [la surveillance Pathly](https://pathlyhq.com) — contrôles HTTP et parcours navigateur (panier, connexion, disponibilité), données hébergées dans l’UE. Référence API : [pathlyhq.com/fr/developers](https://pathlyhq.com/fr/developers).

> **La version anglaise fait référence.** Ce document traduit
> [`README.md`](README.md), seule version lue par le registre Terraform. En cas
> d'écart entre les deux, la version anglaise prévaut. Une traduction périmée
> qui annoncerait de mauvaises portées d'API conduirait à créer une clé trop
> privilégiée.

Gérez la surveillance Pathly comme du code : scénarios, fenêtres de
maintenance, webhooks sortants et objectifs de disponibilité. Construit avec
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework),
au-dessus de l'API publique `/v1`, documentée sur
[pathlyhq.com/fr/developers](https://pathlyhq.com/fr/developers).

```hcl
terraform {
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

provider "pathly" {}

resource "pathly_scenario" "checkout" {
  name         = "Checkout"
  url          = "https://shop.example.com/cart"
  interval_sec = 300
  expect_text  = "Your cart"
  tags         = ["prod", "payment"]
}
```

## Authentification

La clé d'API est fournie par la variable d'environnement, jamais dans un
fichier `.tf` :

```sh
export PATHLY_API_TOKEN="sp_…"
terraform plan
```

| Variable | Rôle |
|---|---|
| `PATHLY_API_TOKEN` | Clé d'API de l'organisation, préfixe `sp_`. Obligatoire. |
| `PATHLY_API_URL` | Base de l'API. Vaut `https://api.pathlyhq.com` par défaut. |

L'attribut `api_token` du bloc `provider` existe bien, mais une clé lue depuis
une variable Terraform finit **en clair dans l'état**. L'environnement est le
seul chemin qui ne laisse aucune trace, ni dans le dépôt ni dans le fichier
d'état.

### Portées minimales pour la clé

Le provider n'appelle que ce dont il a besoin. Créez la clé avec les portées
strictement exigées par les ressources que vous gérez :

| Ressource gérée | Portées |
|---|---|
| `pathly_scenario`, `pathly_scenarios` | `scenarios:read`, `scenarios:write` |
| `pathly_settings` | `org:read`, `org:write` |
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |
| `pathly_incidents` | `incidents:read` |
| `pathly_members` | `members:read` |
| `pathly_runs`, `pathly_run` | `runs:read` |
| `pathly_usage` | `org:read` |
| `pathly_space`, `pathly_spaces`, `pathly_account` | `org:read`, `org:write` (`org:read` seul pour les deux data sources) |
| `pathly_status_page` (ressource) | `status:read`, `status:write` |

Pour un pipeline qui ne fait que `terraform plan`, les portées `:read`
suffisent.

Aucune portée d'organisation n'est nécessaire : le provider vérifie la clé au
moment de la configuration en appelant `/v1/usage`, et **tolère un refus** de
cet appel. Un 403 prouve que la clé est valide, il signale seulement l'absence
de `org:read`. Exiger cette portée forcerait une configuration limitée aux
scénarios à réclamer l'accès aux paramètres de l'organisation.

Il n'existe pas de portée `keys:*` ni de portée `members:write` : une clé d'API
ne peut pas en créer une autre, ni inviter un compte.

## Ressources et sources de données

| Nom | Ce qu'il gère |
|---|---|
| `pathly_scenario` | Contrôle HTTP ou parcours navigateur (étapes en écriture seule) |
| `pathly_settings` | Paramètres d'organisation (singleton, destroy ne les remet pas à zéro) |
| `pathly_maintenance_window` | Fenêtre de maintenance, ponctuelle ou hebdomadaire |
| `pathly_webhook` | Webhook sortant signé |
| `pathly_sla_target` | Objectif de disponibilité et budget d'erreur |
| `pathly_scenarios` | Inventaire des scénarios |
| `pathly_settings` (données) | Paramètres, lecture seule |
| `pathly_usage` | Consommation du plan |
| `pathly_incidents` | Incidents |
| `pathly_members` | Membres, lecture seule |
| `pathly_runs`, `pathly_run` | Exécutions |
| `pathly_sla`, `pathly_sla_targets` | Objectifs, avec ou sans mesure |
| `pathly_webhooks` | Webhooks sans leur destination |
| `pathly_maintenance_windows` | Fenêtres |
| `pathly_status_page` (données) | Composants de la page de statut publique |
| `pathly_space` | Espace : les scénarios d'un client, d'une marque ou d'une application (destroy archive) |
| `pathly_status_page` | Page de statut de l'organisation ou d'un Espace, avec son badge (destroy dépublie) |
| `pathly_spaces` | Espaces actifs et limite de l'offre |
| `pathly_account` | Organisation et Espace de la clé, offre, quotas utilisés et restants |

Documentation par ressource dans [`docs/`](docs/), traduite en français dans
[`translations/fr/`](translations/fr/), exemples complets dans
[`examples/`](examples/).

## Ce que le provider ne fait pas, et pourquoi

- **Ni organisations, ni utilisateurs, ni clés d'API.** Une clé capable
  d'émettre des clés ou d'inviter des comptes transforme une fuite de jeton en
  prise de contrôle de l'organisation. Ces actions restent dans la console,
  avec un compte nominatif et une piste d'audit.
- **Ni mise en sourdine d'incident, ni acquittement d'incident.** Ce sont des
  gestes opérationnels temporaires, pas un état désiré : les inscrire dans le
  code signifierait qu'un `apply` réveille un scénario délibérément mis en
  sourdine la veille au soir. `muted_until` est exposé en lecture seule.
- **Aucun destinataire de notification individuel** (adresses, numéros de
  téléphone). Ce sont des données personnelles : les écrire dans un fichier
  d'état Terraform, souvent partagé et rarement chiffré, crée une obligation
  RGPD que personne n'a demandée.

## Comportement quand les choses tournent mal

- **Ressource supprimée depuis la console** : la lecture la retire de l'état,
  le plan suivant la recrée. Un refus (403) ou une panne (5xx) n'est jamais
  pris pour une disparition, sans quoi un `apply` créerait un doublon à côté de
  l'objet existant.
- **Limite de débit** : le client honore l'en-tête `Retry-After`, avec un
  plafond de 90 secondes et quatre tentatives, puis échoue visiblement.
- **Réponse perdue après une création** : chaque création porte une clé
  d'idempotence, réutilisée par les nouvelles tentatives internes. La nouvelle
  tentative renvoie la ressource déjà créée au lieu d'en créer une seconde, que
  Terraform ne connaîtrait pas et ne détruirait jamais.
- **Webhook importé** : l'API ne renvoie jamais l'URL stockée ni le secret de
  signature. L'import émet un avertissement et vous demande de redéclarer `url`
  dans la configuration.

## Développement

```sh
go build ./...                                   # build
go test ./internal/... -coverprofile=covprofile   # tests, 100% of statements
go tool cover -func=covprofile                    # per-function detail
go vet ./...
```

`main.go` n'est pas couvert : il ne contient que l'appel de service du plugin,
qui bloque. La couverture est mesurée sur `./internal/...`, là où vit tout le
comportement.

### Essai local, sans publier

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "pathlyhq/pathly" = "/path/to/your/GOBIN"
  }
  direct {}
}
```

```sh
go install .   # drops the binary into $GOBIN
cd examples/complete && terraform plan
```

Avec un `dev_overrides`, `terraform init` est inutile et affiche un
avertissement : c'est attendu.

### Contre une API locale

```sh
export PATHLY_API_URL="http://localhost:8080"
export PATHLY_API_TOKEN="sp_…"
```

Le HTTP en clair n'est toléré que sur `localhost` : partout ailleurs, la clé
circulerait lisible dans l'en-tête `Authorization`.

## Publication

Ce dépôt est la source, et ses workflows GitHub Actions construisent, signent et
publient les archives. Il existe deux canaux. Ils sont indépendants, et les
mêmes archives signées alimentent les deux.

| Canal | Adresse de source | Qui peut l'installer |
|---|---|---|
| Registre Terraform public | `pathlyhq/pathly` | tout le monde |
| Registre privé HCP Terraform | `app.terraform.io/Pathly/pathly` | les membres de l'organisation `Pathly` |

Le registre public authentifie par GitHub, son espace de noms **est** le nom de
l'organisation GitHub, et il ingère les versions par un webhook sur les releases
de ce dépôt public. Il ne lit jamais une release non signée : une release sans
`SHA256SUMS.sig` est ignorée, et `terraform init` refuse alors le provider.

### Mise en place, une fois

**1. Clé de signature.** Les deux registres refusent les courbes elliptiques,
la clé doit donc être RSA :

```sh
gpg --full-generate-key            # type RSA, 4096 bits, avec une phrase de passe
gpg --list-secret-keys --keyid-format=long     # noter l'empreinte
gpg --armor --export <empreinte>               # partie publique, à coller dans le registre
gpg --armor --export-secret-keys <empreinte>   # partie privée, pour l'environnement release
```

**2. Environnement `release`** (*Settings → Environments*), limité à la branche
`main`, pour qu'aucune autre branche ne puisse lire la clé :

| Secret | Contenu |
|---|---|
| `GPG_PRIVATE_KEY` | Partie privée de la clé, en ASCII armé ou en base64 |
| `GPG_PASSPHRASE` | Phrase de passe de cette clé |
| `TFE_TOKEN` | Facultatif, registre privé HCP : jeton d'équipe avec *Manage private registry* |
| `TFE_GPG_KEY_ID` | Facultatif, registre privé HCP : identifiant rendu à l'enregistrement de la clé publique |

**3. Registre public.** Sur [registry.terraform.io](https://registry.terraform.io),
connecté avec GitHub : la clé publique sous *User settings → Signing keys*, puis
*Publish → Provider* sur ce dépôt (cela installe le webhook).

**4. Registre privé HCP** (facultatif). Enregistrer la clé publique une fois, et
garder l'identifiant rendu dans `TFE_GPG_KEY_ID` :

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <empreinte>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

Les utilisateurs du registre privé ont besoin d'un jeton pour l'hôte :
`terraform login app.terraform.io`, ou `TF_TOKEN_app_terraform_io` en CI.

### À chaque version

release-please tient à jour une pull request `chore(main): release x.y.z` à
partir des Conventional Commits fusionnés sur `main`. Tant que l'API est en v1,
les versions avancent d'un patch à la fois (`feat` compris). Fusionner cette pull
request est la décision de publier :

1. release-please crée une release GitHub en **brouillon** ;
2. le job `goreleaser` lance les tests, construit les onze cibles, signe les
   sommes de contrôle, joint les archives et le manifeste, puis publie la
   release, ce qui crée le tag `vX.Y.Z` et déclenche le webhook du registre ;
3. `publish-hcp` envoie les mêmes archives au registre privé quand `TFE_TOKEN`
   est renseigné.

Une version publiée est consommée aussitôt par `terraform init`, et ne peut
jamais être retirée d'un registre.

### Vérifier qu'une version a bien atterri

```sh
# Public registry
curl -s https://registry.terraform.io/v1/providers/pathlyhq/pathly/versions | jq '.versions[].version'

# HCP private registry
curl -s -H "Authorization: Bearer $TFE_TOKEN" \
  "https://app.terraform.io/api/v2/organizations/Pathly/registry-providers/private/Pathly/pathly/versions" \
  | jq -r '.data[].attributes.version'
```

Sur le registre public, une version absente alors que la release GitHub est
publiée signale presque toujours une signature rejetée : une archive sans
`SHA256SUMS.sig`, ou une empreinte déclarée au registre différente de celle qui
a réellement signé. Sur HCP, relancez `publish-hcp` : les appels peuvent être
répétés sans risque.

## Pathly + Terraform — checklist d’adoption

1. **Terraform Registry** — `source = "pathlyhq/pathly"` (registre public). Ne pas vendor le binaire.
2. **Auth** — uniquement `PATHLY_API_TOKEN` en CI. Jamais la clé dans Git.
3. **Import** — `terraform import pathly_scenario.name mon_…` pour reprendre les ressources créées dans la console.
4. **Docs** — pages sous [`docs/`](docs/) pour le Registry (`tfplugindocs`). L’anglais est canonique sur le Registry.
5. **Console Pathly** — créez la clé API sur [pathlyhq.com](https://pathlyhq.com), puis pilotez via ce provider. [Développeurs](https://pathlyhq.com/fr/developers).


## Packages associés

| Package | Role |
|---|---|
| [pathly-opentofu](https://github.com/pathlyhq/pathly-opentofu) | OpenTofu docs & examples |
| [pathly-cdktf](https://github.com/pathlyhq/pathly-cdktf) | CDK for Terraform |
| [pathly-pulumi](https://github.com/pathlyhq/pathly-pulumi) | Pulumi |
| [pathly-ansible](https://github.com/pathlyhq/pathly-ansible) | Ansible |
| [pathly-sdk-go](https://github.com/pathlyhq/pathly-sdk-go) | Go SDK |
| [pathly-sdk-python](https://github.com/pathlyhq/pathly-sdk-python) | Python SDK |
| [Pathly product](https://pathlyhq.com) | [Pathly monitoring](https://pathlyhq.com) |
## À propos de Pathly

[Pathly](https://pathlyhq.com) surveille les parcours clients des agences et e-commerçants : rejoue le tunnel, détecte un checkout cassé avant l’appel du client, et joint la preuve (capture, étape, consigne) à la facture de maintenance. Produit : [pathlyhq.com](https://pathlyhq.com) · Développeurs : [pathlyhq.com/fr/developers](https://pathlyhq.com/fr/developers) · Tarifs : [pathlyhq.com/fr/pricing](https://pathlyhq.com/fr/pricing).

## Auteur

| | |
|---|---|
| **Entreprise** | Pathly |
| **Auteur** | Simon Raynaud / keyral |

Voir [AUTHORS](AUTHORS).
