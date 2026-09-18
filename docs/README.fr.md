# ResumeGPT

<p align="center">
  <img src="./assets/resumegpt-logo.png" alt="ResumeGPT Ré logo" width="128" height="128">
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <b>Français</b> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.zh-CN.md">简体中文</a> ·
  <a href="./README.zh-TW.md">繁體中文</a>
</p>

<p align="center">
  <a href="https://resumegpt.tech-fun.net">Page d'accueil du projet</a>
</p>

ResumeGPT est un espace de travail auto-hébergé permettant d'adapter CV et lettres de motivation à des offres d'emploi précises. Il regroupe en une seule application la gestion de profils, le suivi des candidatures, la découverte planifiée d'offres, des modèles réutilisables, des agents LLM configurables, la génération de PDF, la relecture visuelle et le suivi des tâches en arrière-plan.

## En bref

Avec Docker Compose installé, démarrez l'application complète en une seule commande :

```bash
docker compose up -d
```

Cette commande récupère les images `latest` pré-construites publiées par la CI (voir [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml)) au lieu de les construire localement, de sorte que la pile démarre en quelques secondes. Si vous travaillez sur le code et souhaitez que Compose construise les images `api`, `worker`, `document-worker`, `web-worker` et `web` à partir de votre copie locale, ajoutez `--build` :

```bash
docker compose up -d --build
```

Dans les deux cas, ouvrez [http://localhost:5173](http://localhost:5173) une fois la pile prête. La pile Compose fournit des valeurs par défaut de développement, crée ses volumes de stockage, applique les migrations de base de données et attend que les dépendances deviennent saines.

## Fonctionnalités

### Profils

- Créer et gérer plusieurs profils avec un nom, un poste cible, une langue par défaut et un contenu compatible Markdown.
- Saisir le contenu du profil directement ou l'importer depuis des fichiers TXT, Markdown, TeX, DOC, DOCX, PDF, PNG et JPEG.
- Relire et modifier le texte extrait avant de l'enregistrer dans un profil.
- Ajouter un avatar JPEG ou PNG optionnel pour les mises en page de CV qui prennent en charge une photo.
- Rechercher, filtrer, paginer, modifier et supprimer des profils depuis l'interface web.

### Offres d'emploi

- Suivre le titre du poste, l'entreprise, la localisation, le pays, la ville, le mode de travail, le type de contrat, l'URL source, la description et le statut de candidature.
- Créer des offres manuellement ou en importer jusqu'à 50 URL publiques à la fois.
- Utiliser une extraction déterministe rapide pour les pages LinkedIn et Indeed prises en charge.
- Activer l'agent d'import d'offres pour une extraction assistée par IA depuis d'autres pages d'offres HTTPS publiques.
- Modifier les informations importées et mettre à jour le statut de candidature directement depuis la liste des offres.
- Filtrer les offres par origine afin de distinguer les résultats manuels, importés par URL et trouvés par le Job Hunter.
- Sélectionner plusieurs offres et créer des tâches de CV ou de lettre de motivation par lot.

### Job Hunter

- Planifier des recherches récurrentes par métier, localisation, mode de travail, type de contrat, expérience, mots-clés et une instruction supplémentaire.
- Utiliser éventuellement un profil enregistré comme contexte de correspondance.
- Limiter chaque exécution à 10 nouvelles offres au maximum.
- Lancer, mettre en pause, reprendre, modifier ou supprimer un Hunter depuis l'interface web.
- Dédupliquer les URL découvertes avant de créer des offres.
- Écarter les résultats bloqués ou inexploitables de la liste des offres et les rassembler dans une boîte de confirmation pour une relecture manuelle.

### Modèles

- Gérer des modèles séparés pour le CV et la lettre de motivation.
- Importer des fichiers DOC, DOCX, TeX en fichier unique ou des projets LaTeX en archive ZIP multi-fichiers.
- Spécifier un fichier d'entrée LaTeX lorsqu'une archive ZIP n'utilise pas de `main.tex` non ambigu.
- Analyser les sources importées, extraire un contenu lisible par un LLM et générer un aperçu PDF mis en cache.
- Consulter, mettre à jour, télécharger et supprimer les modèles personnalisés.
- Utiliser le modèle LaTeX Rezume fourni en lecture seule, avec attribution à sa [source Overleaf](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs) d'origine.

### Génération de CV et de lettres de motivation

- Générer un CV ou une lettre de motivation sur mesure à partir d'un profil enregistré et d'une offre d'emploi.
- Utiliser un modèle LaTeX optionnel, ou laisser le modèle vide et laisser l'agent Document Designer créer une mise en page HTML/CSS prête à l'impression.
- Configurer un LLM par défaut différent pour chaque agent, ou remplacer le routage par défaut par un seul modèle pour l'ensemble du flux d'une génération.
- Exécuter les rôles Writer, Template Applying ou Document Designer, et Visual Reviewer via des exécuteurs d'agents LangChainGo bornés.
- Générer des artefacts LaTeX et HTML/CSS, ainsi que des aperçus de modèles Word, via des outils de rendu isolés.
- Rastériser les pages PDF pour la relecture visuelle et effectuer jusqu'à deux cycles automatiques de réparation de mise en page.
- Conserver un PDF valide avec un avertissement visible lorsque le modèle sélectionné ne peut pas effectuer d'inspection visuelle.
- Basculer vers une mise en page de secours sûre si un document généré par IA ne peut pas être rendu de manière fiable.
- Suivre la progression de la génération et consulter les brouillons, les PDF intermédiaires, les retours de relecture, les avertissements et les instructions utilisateur dans une chronologie.
- Modifier les entrées d'une génération terminée ou échouée et la régénérer sans perdre les entrées précédentes de la chronologie.
- Envoyer une instruction complémentaire pour réviser un artefact terminé.

### Fournisseurs LLM et paramètres

- Configurer des fournisseurs OpenAI, compatibles OpenAI et Ollama en local.
- Stocker les jetons d'API chiffrés au repos et ne jamais renvoyer de jetons en clair au navigateur.
- Découvrir automatiquement les modèles disponibles via `/models` ou le point de terminaison `/api/tags` d'Ollama.
- Attribuer des fournisseurs et modèles par défaut indépendamment aux agents Writer, Template Applying, Document Designer, Visual Reviewer, Job Import et Job Hunter.
- Remplacer le routage par défaut par un seul modèle pour une génération individuelle.
- Configurer la langue de l'interface et le thème Système, Clair ou Sombre.
- Basculer l'interface entre l'anglais, l'allemand, le français, l'espagnol, le japonais, le chinois simplifié et le chinois traditionnel depuis les paramètres.

### Traitement en arrière-plan et exploitation

- Exécuter l'extraction de documents, la préparation des modèles, l'import d'offres, la recherche d'offres et la génération de documents via une file d'attente durable adossée à PostgreSQL.
- Récupérer les tâches en bail, réessayer les échecs transitoires et persister les événements de cycle de vie des tâches.
- Inspecter l'état des tâches, les tentatives, les entrées assainies, les erreurs et les journaux dans le Task Monitor.
- Stocker les fichiers source, avatars, aperçus et PDF générés dans un stockage objet compatible S3.
- Persister les enregistrements d'outbox transactionnelle et d'audit pour les mutations essentielles.
- Émettre des journaux structurés et des traces OpenTelemetry avec propagation du contexte de trace W3C.
- Créer des sauvegardes PostgreSQL et exécuter une vérification de restauration automatisée avec les scripts fournis.

## Architecture

Voir [Architecture](./architecture-design.md) pour le modèle de composants implémenté, l'organisation de la persistance, les flux de travail et les frontières de sécurité.

## Captures d'écran

<table>
<tr>
<td width="50%"><img src="./images/landing_page_overview.png" alt="Workspace overview"><br><sub>Vue d'ensemble de l'espace de travail — profils, offres d'emploi, modèles, fournisseurs et PDF générés en un coup d'œil.</sub></td>
<td width="50%"><img src="./images/profiles.png" alt="Profiles"><br><sub>Profils — un profil ciblé par poste, avec contenu enregistré et langue.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/opportunity_list.png" alt="Job Opportunities list"><br><sub>Offres d'emploi — suivre le statut et la source, et lancer un CV ou une lettre de motivation pour n'importe quelle offre.</sub></td>
<td width="50%"><img src="./images/opportunity_detail.png" alt="Job opportunity detail"><br><sub>Détail d'une offre d'emploi — description complète, localisation, mode de travail et statut de suivi.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/job_hunter.png" alt="Job Hunter"><br><sub>Job Hunter — recherches planifiées qui font remonter automatiquement de nouvelles offres.</sub></td>
<td width="50%"><img src="./images/templates.png" alt="Templates"><br><sub>Modèles — le modèle LaTeX Rezume intégré aux côtés d'un modèle personnalisé importé.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv.png" alt="Create CVs and Cover Letters"><br><sub>Créer des CV et lettres de motivation — chaque génération avec son statut, son modèle et son gabarit.</sub></td>
<td width="50%"><img src="./images/opportunity_create_cv.png" alt="Quick CV generation dialog"><br><sub>Génération rapide — démarrer un CV ou une lettre de motivation directement depuis une offre d'emploi.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv_stages.png" alt="Generation progress and PDF preview"><br><sub>Progression de génération en direct avec aperçu PDF par étape, ici pour un CV en japonais.</sub></td>
<td width="50%"><img src="./images/setting-1.png" alt="Settings: LLM providers"><br><sub>Paramètres — fournisseurs LLM configurés, cloud et local.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/setting-2.png" alt="Settings: default models"><br><sub>Paramètres — routage des modèles par défaut selon le rôle de l'agent.</sub></td>
<td width="50%"><img src="./images/setting-3.png" alt="Settings: interface language"><br><sub>Paramètres — sélecteur de langue de l'interface, ici en chinois simplifié.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/task_mng.png" alt="Task Monitor"><br><sub>Task Monitor — historique des tâches en arrière-plan et détails par tâche.</sub></td>
<td width="50%"></td>
</tr>
</table>

## Pile locale

Le déploiement Compose par défaut exécute :

| Service | Rôle | Accès local |
|---|---|---|
| `web` | Application Vue 3 servie par Nginx | `http://localhost:5173` |
| `api` | API HTTP Go | `http://localhost:8080` |
| `worker` | Processeurs et planificateurs d'arrière-plan durables | Interne |
| `document-worker` | Analyse antivirus, extraction, OCR, aperçu et rendu PDF | Interne |
| `web-worker` | Rendu de pages Playwright isolé pour les imports assistés par IA | Interne |
| `postgres` | Données applicatives, file d'attente, événements et paramètres | `localhost:5432` |
| `redis` | Cache à expiration pour la découverte des modèles des fournisseurs | `localhost:6379` |
| `minio` | Stockage objet compatible S3 | API `localhost:9000`, console `localhost:9001` |
| `migrate` | Processus de migration de base de données à usage unique | Interne |
| `pgadmin` | Interface web pour inspecter/déboguer la base Postgres | `http://localhost:5050` |

Vérifier l'état ou suivre les journaux avec :

```bash
docker compose ps
docker compose logs -f
```

Arrêter la pile sans supprimer les données :

```bash
docker compose down
```

Les données PostgreSQL, Redis et MinIO restent dans des volumes nommés. N'utilisez `docker compose down --volumes` que lorsque vous souhaitez délibérément effacer les données applicatives locales.

## Première utilisation

1. Ouvrez `http://localhost:5173/settings`.
2. Ajoutez un fournisseur OpenAI, compatible OpenAI ou Ollama et testez-le.
3. Choisissez les modèles par défaut pour les agents que vous prévoyez d'utiliser.
4. Créez un profil et enregistrez votre contenu source.
5. Ajoutez ou importez une offre d'emploi.
6. Créez un CV ou une lettre de motivation, en choisissant éventuellement un modèle.

Pour un Ollama exécuté sur l'hôte Docker, utilisez `http://host.docker.internal:11434` comme URL de base.

## Configuration

Compose fournit des valeurs par défaut de développement, un fichier `.env` est donc optionnel. Copiez `.env.example` si vous souhaitez personnaliser les ports, les identifiants, le stockage, l'authentification ou la traçabilité :

```bash
cp .env.example .env
```

Les paramètres importants incluent :

| Variable | Rôle |
|---|---|
| `WEB_PORT` | Port web côté navigateur ; par défaut `5173` |
| `API_PORT` | Port API côté navigateur ; par défaut `8080` |
| `POSTGRES_*` | Base de données PostgreSQL locale et identifiants |
| `REDIS_PORT`, `MODEL_CACHE_TTL` | Port Redis et durée de vie du cache des modèles des fournisseurs LLM |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Identifiants locaux du stockage objet |
| `SETTINGS_ENCRYPTION_KEY` | Chiffre les jetons d'API LLM stockés |
| `AUTH_MODE` | `development` ou `oidc` |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID` | Requis lorsque `AUTH_MODE=oidc` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Point de terminaison OpenTelemetry optionnel |

La clé de chiffrement intégrée et les identifiants de stockage sont des valeurs par défaut réservées au développement. Définissez des valeurs privées avant d'utiliser l'application en dehors d'une machine locale.

## Développement

Le développement en local nécessite Go 1.26.8 ou plus récent, Node.js 22 ou plus récent, Python 3.12, `uv` et GNU Make.

Installer les dépendances frontend et Python :

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

Démarrer PostgreSQL et MinIO, appliquer les migrations et exécuter les processus principaux dans des terminaux séparés :

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

Le worker complet nécessite également les workers document et web. Pour un développement de bout en bout, la pile Compose complète est l'option la plus simple.

Exécuter la validation :

```bash
make test
make build
```

Commandes utiles :

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## Organisation du dépôt

```text
apps/web/                   Frontend Vue 3 et TypeScript
cmd/api/                    Point d'entrée de l'API Go
cmd/worker/                 Point d'entrée du worker d'arrière-plan Go
cmd/migrate/                Exécuteur de migrations embarqué
internal/                   Modules de domaine, services, ports et adaptateurs
migrations/                 Migrations PostgreSQL versionnées
services/document-worker/   Service isolé de traitement et de rendu de documents
services/web-worker/        Service isolé de navigateur Playwright
scripts/                    Utilitaires de sauvegarde et de vérification de restauration
docs/                       Documentation d'architecture actuelle
```

## Licence

ResumeGPT est distribué selon les termes du fichier [LICENSE](../LICENSE). Le modèle Rezume fourni conserve ses propres métadonnées d'attribution et de licence dans l'application.
