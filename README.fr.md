# Pathly Terraform Provider

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
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |

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
| `pathly_scenario` | Scénario de surveillance HTTP ou navigateur |
| `pathly_maintenance_window` | Fenêtre de maintenance, ponctuelle ou hebdomadaire |
| `pathly_webhook` | Webhook sortant signé |
| `pathly_sla_target` | Objectif de disponibilité et budget d'erreur |
| `pathly_scenarios` (source de données) | Inventaire des scénarios, filtrable par étiquette ou par dossier |

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

Le développement vit sur GitLab, et le pipeline y construit et y signe les
archives. La destination de ces archives est une décision distincte, et il
existe deux canaux. Ils sont indépendants : chacun fonctionne seul, et les
mêmes archives signées alimentent les deux.

| Canal | Adresse de source | Qui peut l'installer | GitHub nécessaire |
|---|---|---|---|
| Registre Terraform public | `pathlyhq/pathly` | tout le monde | oui, un miroir public |
| Registre privé HCP Terraform | `app.terraform.io/Pathly/pathly` | les membres de l'organisation `Pathly` | non |

Le registre public est le seul à offrir la découvrabilité, un site de
documentation rendu et une installation sans identifiants. Il s'accompagne
aussi d'une exigence ferme : il authentifie par GitHub, son espace de noms
**est** un nom d'organisation GitHub, et il ingère les versions par un webhook
sur les releases d'un dépôt GitHub public. Il n'existe pas d'import depuis
GitLab, pas de téléversement manuel et pas d'API pour pousser une version.
GitHub sert donc de vitrine en lecture seule, alimentée par un miroir, tandis
que la clé de signature ne quitte jamais GitLab.

### Mise en place commune

**1. Clé de signature.** Les deux registres rejettent les courbes elliptiques,
la clé doit donc être RSA :

```sh
gpg --full-generate-key            # RSA type, 4096 bits, with a passphrase
gpg --list-secret-keys --keyid-format=long     # note the fingerprint
gpg --armor --export <fingerprint>             # public part, to paste into the registry
gpg --armor --export-secret-keys <fingerprint> | base64 -w0   # private part, for CI
```

**2. Variables CI/CD GitLab**, toutes **masked (masquées) et protected
(protégées)**. « Protected » n'est pas un confort : sans cet attribut, la clé
de signature est lisible depuis n'importe quelle branche, et donc exfiltrable
par un simple push.

| Variable | Contenu |
|---|---|
| `GPG_PRIVATE_KEY` | Partie privée de la clé, encodée en base64 |
| `GPG_PASSPHRASE` | Phrase de passe de cette clé |
| `GPG_FINGERPRINT` | Empreinte de la clé |

### Canal A — registre Terraform public

Le nom du dépôt GitHub ne se choisit pas librement : la source
`pathlyhq/pathly` impose une organisation `pathlyhq` et un dépôt
`terraform-provider-pathly`.

**1. Dépôt GitHub.** Créez l'organisation `pathlyhq` et le dépôt **public**
`terraform-provider-pathly`, vide, sans README généré et sans licence générée.
Le miroir refuserait de pousser sur un historique divergent.

**2. Miroir GitLab → GitHub.** Sous **Settings → Repository → Mirroring
repositories** (Paramètres → Dépôt → Mise en miroir des dépôts), direction
*Push*, URL `https://github.com/pathlyhq/terraform-provider-pathly.git`, avec
un jeton GitHub comme mot de passe. Laissez « Mirror only protected branches »
(ne mettre en miroir que les branches protégées) **décoché** : sans les tags,
le registre n'a rien à lire.

**3. Variable CI/CD** `GITHUB_TOKEN`, masked (masquée) et protected
(protégée) : un jeton **fine-grained (granulaire)** avec la permission
`contents: write` sur le seul dépôt `terraform-provider-pathly`, et une
expiration courte. Un jeton classic (classique) donnerait accès à toute
l'organisation.

**4. Déclaration au registre.** Connectez-vous à
[registry.terraform.io](https://registry.terraform.io) avec le compte GitHub,
déclarez la clé publique sous *User settings → Signing keys* (Paramètres
utilisateur → Clés de signature), puis *Publish → Provider* (Publier →
Provider) et choisissez le dépôt. Le webhook est installé à ce moment-là.

### Canal B — registre privé HCP Terraform

Aucun GitHub sur ce chemin. Un provider est publié directement dans le registre
de l'organisation par son API — qui est d'ailleurs la seule voie, puisque la
console HCP et la connexion VCS ne gèrent que les modules, jamais les
providers.

**1. Variable CI/CD** `TFE_TOKEN`, masked (masquée) et protected (protégée) :
un jeton HCP Terraform appartenant à une équipe qui détient la permission
*Manage private registry* (Gérer le registre privé). Préférez un jeton d'équipe
à un jeton personnel, pour que le pipeline ne cesse pas de fonctionner le jour
où son auteur part.

**2. Enregistrez la clé publique** une fois, et conservez l'id qu'elle renvoie.
Pour un registre privé, l'espace de noms est le nom de l'organisation :

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <fingerprint>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

**3. Variable CI/CD** `TFE_GPG_KEY_ID` avec cet id. C'est elle qui relie une
version publiée à la clé qui l'a signée.

**4. Les consommateurs** ont besoin d'un jeton pour l'hôte, puisqu'un registre
privé est authentifié. `terraform login app.terraform.io` en écrit un, ou
définissez `TF_TOKEN_app_terraform_io` dans la CI :

```hcl
terraform {
  required_providers {
    pathly = {
      source  = "app.terraform.io/Pathly/pathly"
      version = "~> 0.1"
    }
  }
}
```

### À chaque version

```sh
git tag v0.1.0 && git push origin v0.1.0
```

Le pipeline teste, valide les exemples, puis s'arrête. Trois jobs manuels
suivent, dans cet ordre : `archives` construit les onze cibles et signe les
sommes de contrôle, après quoi `publish-github` et `publish-hcp` téléversent
ces mêmes archives vers les canaux que vous utilisez. Aucun des deux jobs de
publication n'a besoin de l'autre.

`publish-github` crée la release GitHub en **brouillon**, le registre n'ingère
donc rien tant que vous ne l'avez pas publiée à la main. Deux gestes
explicites, et c'est délibéré : une version publiée est immédiatement consommée
par les `terraform init` des clients, et ne peut jamais être retirée d'un
registre.

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
a réellement signé.

Sur HCP, une version dont les fichiers de sommes de contrôle ou les binaires de
plateforme n'ont pas tous été téléversés reste en place mais inutilisable, et
`terraform init` la signale comme indisponible plutôt qu'absente. Relancez
`publish-hcp` sur le tag : les appels peuvent être répétés sans risque.
