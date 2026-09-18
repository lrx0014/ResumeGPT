# ResumeGPT

<p align="center">
  <img src="./assets/resumegpt-logo.png" alt="ResumeGPT Ré logo" width="128" height="128">
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.fr.md">Français</a> ·
  <b>Español</b> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.zh-CN.md">简体中文</a> ·
  <a href="./README.zh-TW.md">繁體中文</a>
</p>

ResumeGPT es un espacio de trabajo autoalojado para adaptar CV y cartas de presentación a ofertas de empleo concretas. Combina en una sola aplicación la gestión de perfiles, el seguimiento de ofertas, la búsqueda programada de empleo, plantillas reutilizables, agentes LLM configurables, generación de PDF, revisión visual y monitorización de tareas en segundo plano.

## Resumen rápido

Con Docker Compose instalado, inicia la aplicación completa con un solo comando:

```bash
docker compose up -d
```

Esto descarga las imágenes `latest` preconstruidas publicadas por la CI (consulta [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml)) en lugar de construirlas localmente, por lo que la pila arranca en segundos. Si estás trabajando en el código y quieres que Compose construya las imágenes `api`, `worker`, `document-worker`, `web-worker` y `web` a partir de tu copia local, añade `--build`:

```bash
docker compose up -d --build
```

En cualquier caso, abre [http://localhost:5173](http://localhost:5173) cuando la pila esté lista. La pila de Compose proporciona valores predeterminados de desarrollo, crea sus volúmenes de almacenamiento, aplica las migraciones de base de datos y espera a que las dependencias estén saludables.

## Funcionalidades

### Perfiles

- Crea y gestiona varios perfiles con nombre, rol objetivo, idioma predeterminado y contenido compatible con Markdown.
- Introduce el contenido del perfil directamente o impórtalo desde archivos TXT, Markdown, TeX, DOC, DOCX, PDF, PNG y JPEG.
- Revisa y edita el texto extraído antes de guardarlo en un perfil.
- Adjunta opcionalmente un avatar JPEG o PNG para diseños de CV que admitan foto.
- Busca, filtra, pagina, edita y elimina perfiles desde la interfaz web.

### Ofertas de empleo

- Registra el puesto, la empresa, la ubicación, el país, la ciudad, la modalidad de trabajo, el tipo de contrato, la URL de origen, la descripción y el estado de la candidatura.
- Crea ofertas manualmente o importa hasta 50 URL públicas de empleo a la vez.
- Usa extracción determinista rápida para las páginas compatibles de LinkedIn e Indeed.
- Activa el agente de importación de ofertas para extracción asistida por IA desde otras páginas HTTPS públicas de empleo.
- Edita la información importada y actualiza el estado de la candidatura directamente desde la lista de ofertas.
- Filtra las ofertas por origen para distinguir los resultados manuales, importados por URL y encontrados por el Job Hunter.
- Selecciona varias ofertas y crea tareas de CV o carta de presentación por lotes.

### Job Hunter

- Programa búsquedas recurrentes por ocupación, ubicación, modalidad de trabajo, tipo de contrato, experiencia, palabras clave y una instrucción adicional.
- Usa opcionalmente un perfil guardado como contexto de coincidencia.
- Limita cada ejecución a un máximo de 10 nuevas ofertas.
- Ejecuta, pausa, reanuda, edita o elimina un Hunter desde la interfaz web.
- Deduplica las URL encontradas antes de crear ofertas.
- Mantén los resultados bloqueados o no interpretables fuera de la lista de ofertas y recógelos en una bandeja de confirmación para revisión manual.

### Plantillas

- Gestiona plantillas separadas para CV y carta de presentación.
- Sube archivos DOC, DOCX, TeX de un solo archivo o proyectos LaTeX en ZIP de varios archivos.
- Especifica un archivo de entrada LaTeX cuando un ZIP no use un `main.tex` inequívoco.
- Analiza las fuentes subidas, extrae contenido legible por un LLM y genera una vista previa en PDF almacenada en caché.
- Consulta, actualiza, descarga y elimina plantillas personalizadas.
- Usa la plantilla LaTeX Rezume incluida de solo lectura, con atribución a su [fuente original en Overleaf](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs).

### Generación de CV y carta de presentación

- Genera un CV o una carta de presentación a medida a partir de un perfil guardado y una oferta de empleo.
- Usa una plantilla LaTeX opcional, o deja la plantilla vacía y deja que el agente Document Designer cree un diseño HTML/CSS listo para imprimir.
- Configura un LLM predeterminado distinto para cada agente, o sustituye el enrutamiento predeterminado por un único modelo para todo el flujo de una generación.
- Ejecuta los roles Writer, Template Applying o Document Designer, y Visual Reviewer mediante ejecutores de agentes LangChainGo acotados.
- Genera artefactos LaTeX y HTML/CSS, además de vistas previas de plantillas Word, mediante herramientas de renderizado aisladas.
- Rasteriza las páginas del PDF para la revisión visual y realiza hasta dos rondas automáticas de reparación de diseño.
- Conserva un PDF válido con una advertencia visible cuando el modelo seleccionado no pueda realizar inspección visual.
- Recurre a un diseño básico seguro si un documento generado por IA no puede renderizarse de forma fiable.
- Sigue el progreso de la generación e inspecciona borradores, PDF intermedios, comentarios de revisión, advertencias e instrucciones del usuario en una línea de tiempo.
- Edita las entradas de una generación completada o fallida y vuelve a generarla sin descartar los registros anteriores de la línea de tiempo.
- Envía una instrucción de seguimiento para revisar un artefacto completado.

### Proveedores LLM y configuración

- Configura proveedores OpenAI, compatibles con OpenAI y Ollama local.
- Almacena los tokens de API cifrados en reposo y nunca devuelve tokens en texto plano al navegador.
- Descubre automáticamente los modelos disponibles mediante `/models` o el endpoint `/api/tags` de Ollama.
- Asigna proveedores y modelos predeterminados de forma independiente a los agentes Writer, Template Applying, Document Designer, Visual Reviewer, Job Import y Job Hunter.
- Sustituye el enrutamiento predeterminado por un único modelo para una generación individual.
- Configura el idioma de la interfaz y el tema Sistema, Claro u Oscuro.
- Cambia la interfaz entre inglés, alemán, francés, español, japonés, chino simplificado y chino tradicional desde Ajustes.

### Procesamiento en segundo plano y operaciones

- Ejecuta la extracción de documentos, la preparación de plantillas, la importación de ofertas, la búsqueda de empleo y la generación de documentos mediante una cola de trabajo duradera respaldada por PostgreSQL.
- Recupera trabajos con arrendamiento caducado, reintenta fallos transitorios y persiste eventos del ciclo de vida de las tareas.
- Inspecciona el estado de las tareas, los intentos, las entradas depuradas, los errores y los registros en el Task Monitor.
- Almacena archivos de origen, avatares, vistas previas y PDF generados en almacenamiento de objetos compatible con S3.
- Persiste registros transaccionales de outbox y auditoría para las mutaciones principales.
- Emite registros estructurados y trazas de OpenTelemetry con propagación de contexto de traza W3C.
- Crea copias de seguridad de PostgreSQL y ejecuta una comprobación de restauración automatizada con los scripts proporcionados.

## Arquitectura

Consulta [Arquitectura](./architecture-design.md) para conocer el modelo de componentes implementado, la disposición de la persistencia, los flujos de trabajo y los límites de seguridad.

## Capturas de pantalla

<table>
<tr>
<td width="50%"><img src="./images/landing_page_overview.png" alt="Workspace overview"><br><sub>Resumen del espacio de trabajo — perfiles, ofertas de empleo, plantillas, proveedores y PDF generados de un vistazo.</sub></td>
<td width="50%"><img src="./images/profiles.png" alt="Profiles"><br><sub>Perfiles — un perfil enfocado por rol, con contenido guardado e idioma.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/opportunity_list.png" alt="Job Opportunities list"><br><sub>Ofertas de empleo — controla el estado y el origen, y pon en cola un CV o carta de presentación para cualquier oferta.</sub></td>
<td width="50%"><img src="./images/opportunity_detail.png" alt="Job opportunity detail"><br><sub>Detalle de la oferta de empleo — descripción completa, ubicación, modalidad de trabajo y estado de seguimiento.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/job_hunter.png" alt="Job Hunter"><br><sub>Job Hunter — búsquedas programadas que encuentran nuevas ofertas automáticamente.</sub></td>
<td width="50%"><img src="./images/templates.png" alt="Templates"><br><sub>Plantillas — la plantilla LaTeX Rezume integrada junto a una plantilla personalizada subida.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv.png" alt="Create CVs and Cover Letters"><br><sub>Crear CV y cartas de presentación — cada generación con su estado, modelo y plantilla.</sub></td>
<td width="50%"><img src="./images/opportunity_create_cv.png" alt="Quick CV generation dialog"><br><sub>Generación rápida — inicia un CV o una carta de presentación directamente desde una oferta de empleo.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv_stages.png" alt="Generation progress and PDF preview"><br><sub>Progreso de generación en vivo con vista previa del PDF por etapas, aquí para un currículum en japonés.</sub></td>
<td width="50%"><img src="./images/setting-1.png" alt="Settings: LLM providers"><br><sub>Ajustes — proveedores LLM configurados, en la nube y locales.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/setting-2.png" alt="Settings: default models"><br><sub>Ajustes — enrutamiento de modelos predeterminado por rol de agente.</sub></td>
<td width="50%"><img src="./images/setting-3.png" alt="Settings: interface language"><br><sub>Ajustes — selector de idioma de la interfaz, aquí en chino simplificado.</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/task_mng.png" alt="Task Monitor"><br><sub>Task Monitor — historial de tareas en segundo plano y detalles por tarea.</sub></td>
<td width="50%"></td>
</tr>
</table>

## Pila local

El despliegue de Compose predeterminado ejecuta:

| Servicio | Propósito | Acceso local |
|---|---|---|
| `web` | Aplicación Vue 3 servida por Nginx | `http://localhost:5173` |
| `api` | API HTTP en Go | `http://localhost:8080` |
| `worker` | Procesadores y planificadores duraderos en segundo plano | Interno |
| `document-worker` | Escaneo de malware, extracción, OCR, vista previa y renderizado de PDF | Interno |
| `web-worker` | Renderizado de páginas Playwright aislado para importaciones asistidas por IA | Interno |
| `postgres` | Datos de la aplicación, cola, eventos y ajustes | `localhost:5432` |
| `redis` | Caché con expiración para el descubrimiento de modelos de proveedores | `localhost:6379` |
| `minio` | Almacenamiento de objetos compatible con S3 | API `localhost:9000`, consola `localhost:9001` |
| `migrate` | Proceso de migración de base de datos de una sola vez | Interno |
| `pgadmin` | Interfaz web para inspeccionar/depurar la base de datos Postgres | `http://localhost:5050` |

Comprueba el estado o sigue los registros con:

```bash
docker compose ps
docker compose logs -f
```

Detén la pila sin eliminar datos:

```bash
docker compose down
```

Los datos de PostgreSQL, Redis y MinIO permanecen en volúmenes con nombre. Usa `docker compose down --volumes` solo cuando quieras borrar intencionadamente los datos locales de la aplicación.

## Primer uso

1. Abre `http://localhost:5173/settings`.
2. Añade un proveedor OpenAI, compatible con OpenAI u Ollama, y pruébalo.
3. Elige los modelos predeterminados para los agentes que planeas usar.
4. Crea un perfil y guarda tu contenido fuente.
5. Añade o importa una oferta de empleo.
6. Crea un CV o una carta de presentación, seleccionando opcionalmente una plantilla.

Para Ollama ejecutándose en el host de Docker, usa `http://host.docker.internal:11434` como URL base.

## Configuración

Compose proporciona valores predeterminados de desarrollo, por lo que un archivo `.env` es opcional. Copia `.env.example` cuando quieras personalizar puertos, credenciales, almacenamiento, autenticación o trazabilidad:

```bash
cp .env.example .env
```

Los ajustes importantes incluyen:

| Variable | Propósito |
|---|---|
| `WEB_PORT` | Puerto web orientado al navegador; predeterminado `5173` |
| `API_PORT` | Puerto de API orientado al navegador; predeterminado `8080` |
| `POSTGRES_*` | Base de datos PostgreSQL local y credenciales |
| `REDIS_PORT`, `MODEL_CACHE_TTL` | Puerto de Redis y vida útil de la caché de modelos de proveedores LLM |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Credenciales locales de almacenamiento de objetos |
| `SETTINGS_ENCRYPTION_KEY` | Cifra los tokens de API de LLM almacenados |
| `AUTH_MODE` | `development` u `oidc` |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID` | Necesarios cuando `AUTH_MODE=oidc` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Endpoint opcional del recolector de OpenTelemetry |

La clave de cifrado integrada y las credenciales de almacenamiento son valores predeterminados solo para desarrollo. Establece valores privados antes de usar la aplicación fuera de una máquina local.

## Desarrollo

El desarrollo en el host requiere Go 1.26.8 o posterior, Node.js 22 o posterior, Python 3.12, `uv` y GNU Make.

Instala las dependencias de frontend y Python:

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

Inicia PostgreSQL y MinIO, aplica las migraciones y ejecuta los procesos principales en terminales separadas:

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

El worker completo también espera los workers de documentos y web. Para el desarrollo de extremo a extremo, la pila Compose completa es la opción más sencilla.

Ejecuta la validación:

```bash
make test
make build
```

Comandos útiles:

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## Estructura del repositorio

```text
apps/web/                   Frontend en Vue 3 y TypeScript
cmd/api/                    Punto de entrada de la API en Go
cmd/worker/                 Punto de entrada del worker en segundo plano en Go
cmd/migrate/                Ejecutor de migraciones embebido
internal/                   Módulos de dominio, servicios, puertos y adaptadores
migrations/                 Migraciones de PostgreSQL versionadas
services/document-worker/   Servicio aislado de procesamiento y renderizado de documentos
services/web-worker/        Servicio aislado de navegador Playwright
scripts/                    Utilidades de copia de seguridad y comprobación de restauración
docs/                       Documentación de arquitectura actual
```

## Licencia

ResumeGPT se distribuye según los términos de [LICENSE](../LICENSE). La plantilla Rezume incluida conserva sus propios metadatos de atribución y licencia en la aplicación.
