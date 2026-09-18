# ResumeGPT

<p align="center">
  <img src="./assets/resumegpt-logo.png" alt="ResumeGPT Ré logo" width="128" height="128">
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <b>Deutsch</b> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.zh-CN.md">简体中文</a> ·
  <a href="./README.zh-TW.md">繁體中文</a>
</p>

ResumeGPT ist ein selbst gehosteter Arbeitsbereich zum Zuschneiden von Lebensläufen und Anschreiben auf konkrete Stellenangebote. Die Anwendung vereint Profilverwaltung, Job-Tracking, geplante Job-Suche, wiederverwendbare Vorlagen, konfigurierbare LLM-Agenten, PDF-Erzeugung, visuelle Prüfung und Hintergrundaufgaben-Monitoring an einem Ort.

## TL;DR

Mit installiertem Docker Compose startest du die komplette Anwendung mit einem einzigen Befehl:

```bash
docker compose up -d
```

Dabei werden die von CI vorgebauten `latest`-Images geladen (siehe [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml)), statt sie lokal zu bauen — der Stack ist so in Sekunden startklar. Wenn du am Code arbeitest und Compose die Images `api`, `worker`, `document-worker`, `web-worker` und `web` aus deinem lokalen Checkout bauen lassen willst, füge `--build` hinzu:

```bash
docker compose up -d --build
```

Öffne in beiden Fällen [http://localhost:5173](http://localhost:5173), sobald der Stack bereit ist. Der Compose-Stack liefert Entwicklungs-Standardwerte, legt seine Speicher-Volumes an, wendet Datenbankmigrationen an und wartet, bis alle Abhängigkeiten fehlerfrei laufen.

## Funktionen

### Profile

- Mehrere Profile mit Name, Zielrolle, Standardsprache und Markdown-freundlichem Inhalt anlegen und verwalten.
- Profilinhalte direkt eingeben oder aus TXT-, Markdown-, TeX-, DOC-, DOCX-, PDF-, PNG- und JPEG-Dateien importieren.
- Extrahierten Text vor dem Speichern im Profil prüfen und bearbeiten.
- Optional einen JPEG- oder PNG-Avatar für Lebenslauf-Layouts mit Foto hinzufügen.
- Profile über die Weboberfläche suchen, filtern, paginieren, bearbeiten und löschen.

### Stellenangebote

- Jobtitel, Unternehmen, Standort, Land, Stadt, Arbeitsmodus, Beschäftigungsart, Quell-URL, Beschreibung und Bewerbungsstatus erfassen.
- Angebote manuell anlegen oder bis zu 50 öffentliche Job-URLs auf einmal importieren.
- Schnelle deterministische Extraktion für unterstützte LinkedIn- und Indeed-Seiten nutzen.
- Den Job-Import-Agenten für KI-gestützte Extraktion von anderen öffentlichen HTTPS-Jobseiten aktivieren.
- Importierte Informationen bearbeiten und den Bewerbungsstatus direkt in der Angebotsliste aktualisieren.
- Angebote nach Herkunft filtern, damit manuelle, per URL importierte und vom Job Hunter gefundene Ergebnisse unterscheidbar bleiben.
- Mehrere Angebote auswählen und Lebenslauf- oder Anschreiben-Aufgaben im Batch erstellen.

### Job Hunter

- Wiederkehrende Suchen nach Beruf, Standort, Arbeitsmodus, Vertragsart, Erfahrung, Stichwörtern und einem zusätzlichen Prompt planen.
- Optional ein gespeichertes Profil als Abgleichskontext verwenden.
- Jeden Lauf auf höchstens 10 neue Angebote begrenzen.
- Einen Hunter über die Weboberfläche starten, pausieren, fortsetzen, bearbeiten oder löschen.
- Gefundene URLs deduplizieren, bevor Angebote erstellt werden.
- Blockierte oder nicht auswertbare Ergebnisse aus der Angebotsliste heraushalten und in einem Bestätigungs-Postfach zur manuellen Prüfung sammeln.

### Vorlagen

- Getrennte Vorlagen für Lebenslauf und Anschreiben verwalten.
- DOC-, DOCX-, einzelne TeX-Dateien oder mehrteilige LaTeX-ZIP-Projekte hochladen.
- Eine LaTeX-Einstiegsdatei angeben, wenn ein ZIP kein eindeutiges `main.tex` verwendet.
- Hochgeladene Quellen scannen, LLM-lesbaren Inhalt extrahieren und eine zwischengespeicherte PDF-Vorschau erzeugen.
- Eigene Vorlagen ansehen, aktualisieren, herunterladen und löschen.
- Die mitgelieferte, schreibgeschützte Rezume-LaTeX-Vorlage nutzen, mit Attribution zu ihrer [Overleaf-Quelle](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs).

### Erstellung von Lebenslauf und Anschreiben

- Einen maßgeschneiderten Lebenslauf oder ein Anschreiben aus einem gespeicherten Profil und einem Stellenangebot erzeugen.
- Eine optionale LaTeX-Vorlage verwenden oder die Vorlage leer lassen und den Document-Designer-Agenten ein druckfertiges HTML/CSS-Layout gestalten lassen.
- Für jeden Agenten ein eigenes Standard-LLM konfigurieren oder eine Generierung mit einem einzigen Modell für den gesamten Ablauf überschreiben.
- Writer-, Template-Applying- bzw. Document-Designer- und Visual-Reviewer-Rollen über begrenzte LangChainGo-Agent-Executoren ausführen.
- LaTeX- und HTML/CSS-Artefakte sowie Word-Vorlagenvorschauen über isolierte Rendering-Werkzeuge erzeugen.
- PDF-Seiten für die visuelle Prüfung rastern und bis zu zwei automatische Layout-Reparaturrunden durchführen.
- Ein gültiges PDF mit sichtbarem Hinweis erhalten, wenn das gewählte Modell keine visuelle Prüfung durchführen kann.
- Bei unzuverlässig renderbaren KI-generierten Dokumenten auf ein sicheres Basislayout zurückfallen.
- Den Generierungsfortschritt verfolgen und Entwürfe, Zwischen-PDFs, Prüf-Feedback, Warnungen und Nutzerhinweise in einer Zeitleiste einsehen.
- Eingaben einer abgeschlossenen oder fehlgeschlagenen Generierung bearbeiten und neu generieren, ohne frühere Zeitleisten-Einträge zu verwerfen.
- Eine Folgeanweisung senden, um ein fertiges Artefakt zu überarbeiten.

### LLM-Anbieter und Einstellungen

- OpenAI-, OpenAI-kompatible und lokale Ollama-Anbieter konfigurieren.
- API-Token verschlüsselt speichern und nie im Klartext an den Browser zurückgeben.
- Verfügbare Modelle automatisch über `/models` bzw. Ollamas `/api/tags`-Endpunkt ermitteln.
- Standardanbieter und -modelle unabhängig für Writer-, Template-Applying-, Document-Designer-, Visual-Reviewer-, Job-Import- und Job-Hunter-Agenten zuweisen.
- Das Standard-Routing für eine einzelne Generierung mit einem einzigen Modell überschreiben.
- Die Oberflächensprache sowie System-, helles oder dunkles Theme konfigurieren.
- Die Oberfläche in den Einstellungen zwischen Englisch, Deutsch, Französisch, Spanisch, Japanisch, vereinfachtem und traditionellem Chinesisch umschalten.

### Hintergrundverarbeitung und Betrieb

- Dokumentenextraktion, Vorlagenaufbereitung, Job-Import, Job-Hunting und Dokumentengenerierung über eine PostgreSQL-gestützte, dauerhafte Warteschlange ausführen.
- Verwaiste (geleaste) Jobs wiederherstellen, transiente Fehler erneut versuchen und Task-Lifecycle-Ereignisse persistieren.
- Task-Status, Versuche, bereinigte Eingaben, Fehler und Logs im Task Monitor einsehen.
- Quelldateien, Avatare, Vorschauen und erzeugte PDFs in S3-kompatiblem Objektspeicher ablegen.
- Transaktionale Outbox- und Audit-Datensätze für zentrale Mutationen persistieren.
- Strukturierte Logs und OpenTelemetry-Traces mit W3C-Trace-Context-Propagation ausgeben.
- Mit den mitgelieferten Skripten PostgreSQL-Backups erstellen und eine automatisierte Wiederherstellungsprüfung durchführen.

## Architektur

Siehe [Architektur](./architecture-design.md) für das implementierte Komponentenmodell, das Persistenzlayout, die Abläufe und die Sicherheitsgrenzen.

## Screenshots

<table>
<tr>
<td width="50%"><img src="./images/landing_page_overview.png" alt="Workspace overview"><br><sub>Arbeitsbereich-Übersicht — Profile, Stellenangebote, Vorlagen, Anbieter und erzeugte PDFs auf einen Blick.</sub></td>
<td width="50%"><img src="./images/profiles.png" alt="Profiles"><br><sub>Profile — ein fokussiertes Profil pro Rolle, mit gespeichertem Inhalt und Sprache.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/opportunity_list.png" alt="Job Opportunities list"><br><sub>Stellenangebote — Status und Quelle im Blick behalten und für jedes Angebot einen Lebenslauf oder ein Anschreiben anstoßen.</sub></td>
<td width="50%"><img src="./images/opportunity_detail.png" alt="Job opportunity detail"><br><sub>Detailansicht eines Stellenangebots — vollständige Beschreibung, Standort, Arbeitsmodus und Tracking-Status.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/job_hunter.png" alt="Job Hunter"><br><sub>Job Hunter — geplante Suchen, die automatisch neue Stellenangebote finden.</sub></td>
<td width="50%"><img src="./images/templates.png" alt="Templates"><br><sub>Vorlagen — die integrierte Rezume-LaTeX-Vorlage neben einer hochgeladenen eigenen Vorlage.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv.png" alt="Create CVs and Cover Letters"><br><sub>Lebensläufe und Anschreiben erstellen — jede Generierung mit Status, Modell und Vorlage.</sub></td>
<td width="50%"><img src="./images/opportunity_create_cv.png" alt="Quick CV generation dialog"><br><sub>Schnellgenerierung — direkt aus einem Stellenangebot einen Lebenslauf oder ein Anschreiben starten.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv_stages.png" alt="Generation progress and PDF preview"><br><sub>Live-Generierungsfortschritt mit gestufter PDF-Vorschau, hier für einen japanischen Lebenslauf.</sub></td>
<td width="50%"><img src="./images/setting-1.png" alt="Settings: LLM providers"><br><sub>Einstellungen — konfigurierte LLM-Anbieter, Cloud und lokal.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/setting-2.png" alt="Settings: default models"><br><sub>Einstellungen — Standard-Modell-Routing je Agentenrolle.</sub></td>
<td width="50%"><img src="./images/setting-3.png" alt="Settings: interface language"><br><sub>Einstellungen — Sprachumschalter der Oberfläche, hier in vereinfachtem Chinesisch.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/task_mng.png" alt="Task Monitor"><br><sub>Task Monitor — Verlauf der Hintergrundaufgaben mit Details je Task.</sub></td>
<td width="50%"></td>
</tr>
</table>

## Lokaler Stack

Das standardmäßige Compose-Deployment betreibt:

| Dienst | Zweck | Lokaler Zugriff |
|---|---|---|
| `web` | Vue-3-Anwendung, ausgeliefert über Nginx | `http://localhost:5173` |
| `api` | Go-HTTP-API | `http://localhost:8080` |
| `worker` | Dauerhafte Hintergrundprozessoren und Zeitpläne | Intern |
| `document-worker` | Malware-Scan, Extraktion, OCR, Vorschau und PDF-Rendering | Intern |
| `web-worker` | Isoliertes Playwright-Rendering für KI-gestützte Importe | Intern |
| `postgres` | Anwendungsdaten, Warteschlange, Ereignisse und Einstellungen | `localhost:5432` |
| `redis` | Ablaufender Cache für die Modellerkennung der Anbieter | `localhost:6379` |
| `minio` | S3-kompatibler Objektspeicher | API `localhost:9000`, Konsole `localhost:9001` |
| `migrate` | Einmaliger Datenbankmigrationsprozess | Intern |
| `pgadmin` | Web-UI zur Inspektion/Fehlersuche der Postgres-Datenbank | `http://localhost:5050` |

Status prüfen oder Logs verfolgen mit:

```bash
docker compose ps
docker compose logs -f
```

Den Stack stoppen, ohne Daten zu löschen:

```bash
docker compose down
```

PostgreSQL-, Redis- und MinIO-Daten bleiben in benannten Volumes erhalten. Nutze `docker compose down --volumes` nur, wenn du absichtlich lokale Anwendungsdaten löschen willst.

## Erste Nutzung

1. Öffne `http://localhost:5173/settings`.
2. Füge einen OpenAI-, OpenAI-kompatiblen oder Ollama-Anbieter hinzu und teste ihn.
3. Wähle Standardmodelle für die Agenten, die du nutzen willst.
4. Lege ein Profil an und speichere deinen Quellinhalt.
5. Füge ein Stellenangebot hinzu oder importiere eines.
6. Erstelle einen Lebenslauf oder ein Anschreiben, optional mit einer ausgewählten Vorlage.

Für Ollama auf dem Docker-Host verwende `http://host.docker.internal:11434` als Basis-URL.

## Konfiguration

Compose liefert Entwicklungs-Standardwerte, eine `.env`-Datei ist daher optional. Kopiere `.env.example`, wenn du Ports, Zugangsdaten, Speicher, Authentifizierung oder Tracing anpassen willst:

```bash
cp .env.example .env
```

Wichtige Einstellungen sind:

| Variable | Zweck |
|---|---|
| `WEB_PORT` | Browserseitiger Web-Port; Standard `5173` |
| `API_PORT` | Browserseitiger API-Port; Standard `8080` |
| `POSTGRES_*` | Lokale PostgreSQL-Datenbank und Zugangsdaten |
| `REDIS_PORT`, `MODEL_CACHE_TTL` | Redis-Port und Lebensdauer des LLM-Anbieter-Modell-Caches |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Lokale Zugangsdaten für den Objektspeicher |
| `SETTINGS_ENCRYPTION_KEY` | Verschlüsselt gespeicherte LLM-API-Token |
| `AUTH_MODE` | `development` oder `oidc` |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID` | Erforderlich, wenn `AUTH_MODE=oidc` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Optionaler OpenTelemetry-Collector-Endpunkt |

Der integrierte Verschlüsselungsschlüssel und die Speicher-Zugangsdaten sind nur für die Entwicklung gedacht. Lege eigene, private Werte fest, bevor du die Anwendung außerhalb eines lokalen Rechners einsetzt.

## Entwicklung

Für die Host-Entwicklung werden Go 1.26.8 oder neuer, Node.js 22 oder neuer, Python 3.12, `uv` und GNU Make benötigt.

Frontend- und Python-Abhängigkeiten installieren:

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

PostgreSQL und MinIO starten, Migrationen anwenden und die Hauptprozesse in getrennten Terminals ausführen:

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

Der vollständige Worker erwartet zudem den Document- und den Web-Worker. Für die durchgängige Entwicklung ist der komplette Compose-Stack die einfachste Option.

Validierung ausführen:

```bash
make test
make build
```

Nützliche Befehle:

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## Repository-Struktur

```text
apps/web/                   Vue 3 und TypeScript Frontend
cmd/api/                    Go API Einstiegspunkt
cmd/worker/                 Go Hintergrund-Worker Einstiegspunkt
cmd/migrate/                Eingebetteter Migrations-Runner
internal/                   Domänenmodule, Services, Ports und Adapter
migrations/                 Versionierte PostgreSQL-Migrationen
services/document-worker/   Isolierter Dokumentenverarbeitungs- und Rendering-Dienst
services/web-worker/        Isolierter Playwright-Browser-Dienst
scripts/                    Backup- und Restore-Check-Hilfsprogramme
docs/                       Aktuelle Architekturdokumentation
```

## Lizenz

ResumeGPT wird gemäß den Bedingungen in [LICENSE](../LICENSE) vertrieben. Die mitgelieferte Rezume-Vorlage behält ihre eigenen Attributions- und Lizenzmetadaten in der Anwendung.
