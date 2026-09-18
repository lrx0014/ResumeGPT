# ResumeGPT

<p align="center">
  <img src="./assets/resumegpt-logo.png" alt="ResumeGPT Ré logo" width="128" height="128">
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.zh-CN.md">简体中文</a> ·
  <b>繁體中文</b>
</p>

ResumeGPT 是一個自行架設的工作平台,用於針對特定職缺量身打造履歷與求職信。它把個人檔案管理、職缺追蹤、定時職缺搜尋、可重複使用的範本、可設定的 LLM 代理程式(Agent)、PDF 產生、視覺審查,以及背景任務監控整合在同一個應用程式中。

## 快速開始

安裝好 Docker Compose 後,一行指令就能啟動完整應用程式:

```bash
docker compose up -d
```

這會直接拉取 CI 建置並發布好的 `latest` 映像檔(參見 [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml)),而不是在本機重新建置,因此整套服務幾秒鐘就能就緒。如果你正在修改程式碼,想讓 Compose 用本機程式碼建置 `api`、`worker`、`document-worker`、`web-worker`、`web` 這幾個映像檔,加上 `--build`:

```bash
docker compose up -d --build
```

不論用哪種方式,等服務就緒後開啟 [http://localhost:5173](http://localhost:5173) 即可。Compose 會提供開發環境的預設設定、建立儲存磁碟區、套用資料庫遷移,並等待各依賴服務轉為健康狀態。

## 功能

### 個人檔案(Profiles)

- 建立並管理多個 Profile,各自擁有名稱、目標職位、預設語言以及支援 Markdown 的內容。
- 直接輸入 Profile 內容,或從 TXT、Markdown、TeX、DOC、DOCX、PDF、PNG、JPEG 檔案匯入。
- 在儲存到 Profile 之前,先檢視並編輯擷取出的文字。
- 可為支援照片的履歷版型選擇性加入 JPEG 或 PNG 大頭貼。
- 在網頁介面搜尋、篩選、分頁、編輯與刪除 Profile。

### 職缺機會(Job Opportunities)

- 記錄職稱、公司、地點、國家、城市、工作模式、雇用類型、來源網址、說明與應徵狀態。
- 手動建立職缺,或一次匯入最多 50 個公開職缺網址。
- 對支援的 LinkedIn 與 Indeed 頁面使用快速的決定性解析。
- 對其他公開 HTTPS 職缺頁面,啟用職缺匯入代理程式進行 AI 輔助解析。
- 直接在職缺清單中編輯匯入資訊、更新應徵狀態。
- 依來源篩選職缺,區分手動新增、以網址匯入,以及 Job Hunter 找到的結果。
- 批次選取多個職缺,一次建立履歷或求職信的產生任務。

### Job Hunter

- 依職稱、地點、工作模式、合約類型、經驗年資、關鍵字與額外提示,排程定期搜尋。
- 可選擇使用已儲存的 Profile 作為比對情境。
- 每次執行最多發現 10 個新職缺。
- 在網頁介面執行、暫停、恢復、編輯或刪除某個 Hunter。
- 在建立職缺前,先對找到的網址去除重複。
- 將被封鎖或無法解析的結果排除在職缺清單之外,統一收進確認信箱供人工複核。

### 範本(Templates)

- 分別管理履歷與求職信範本。
- 上傳 DOC、DOCX、單一檔案的 TeX,或多檔案的 LaTeX ZIP 專案。
- 當 ZIP 內沒有唯一的 `main.tex` 時,可指定 LaTeX 進入檔案。
- 掃描上傳的來源、擷取可供 LLM 讀取的內容,並產生具快取的 PDF 預覽。
- 檢視、更新、下載與刪除自訂範本。
- 使用內建的唯讀 Rezume LaTeX 範本,應用程式內保留了其原始 [Overleaf 來源](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs)的歸屬資訊。

### 履歷與求職信產生

- 依據已儲存的 Profile 與職缺機會,產生量身打造的履歷或求職信。
- 可選擇使用某個 LaTeX 範本,也可以不指定範本,讓 Document Designer 代理程式直接設計出適合列印的 HTML/CSS 版型。
- 為每個代理程式分別設定預設 LLM,也可以在單次產生時用一個模型整體覆寫預設路由。
- 透過具邊界限制的 LangChainGo 代理程式執行器,執行 Writer、Template Applying(或 Document Designer)、Visual Reviewer 這幾個角色。
- 透過隔離的算繪工具產生 LaTeX 與 HTML/CSS 成品,以及 Word 範本預覽。
- 將 PDF 頁面點陣化以進行視覺審查,並最多自動執行兩輪版面修復。
- 當所選模型不支援視覺檢查時,仍保留一份有效 PDF,並附上明顯提示。
- 當 AI 產生的文件無法可靠算繪時,回退到安全的基本版型。
- 在時間軸中追蹤產生進度,檢視草稿、中間產出的 PDF、審查回饋、警告以及使用者提示。
- 編輯一次已完成或失敗之產生工作的輸入內容並重新產生,不會遺失先前的時間軸記錄。
- 傳送後續指示,以修訂已完成的成品。

### LLM 供應商與設定

- 設定 OpenAI、OpenAI 相容介面,以及本機 Ollama 供應商。
- API 權杖會加密儲存,且不會以明文形式回傳到瀏覽器。
- 透過 `/models` 或 Ollama 的 `/api/tags` 端點自動探索可用模型。
- 為 Writer、Template Applying、Document Designer、Visual Reviewer、Job Import、Job Hunter 這幾個代理程式分別指定預設供應商與模型。
- 針對單次產生工作,用一個模型覆寫預設路由。
- 設定介面語言,以及跟隨系統 / 淺色 / 深色主題。
- 在設定頁面於英文、德文、法文、西班牙文、日文、簡體中文、繁體中文之間切換介面語言。

### 背景處理與維運

- 透過以 PostgreSQL 為基礎的持久化工作佇列,執行文件擷取、範本準備、職缺匯入、Job Hunting 與文件產生。
- 復原租約到期的工作、自動重試暫時性失敗,並持久化任務生命週期事件。
- 在 Task Monitor 中檢視任務狀態、重試次數、經過清理的輸入內容、錯誤與記錄檔。
- 將來源檔案、大頭貼、預覽圖與產生的 PDF 儲存在相容於 S3 的物件儲存中。
- 為核心的異動操作持久化交易性 Outbox 與稽核紀錄。
- 輸出結構化記錄,以及符合 W3C trace-context 傳播規範的 OpenTelemetry 追蹤。
- 使用隨附的指令碼建立 PostgreSQL 備份,並執行自動化的還原驗證。

## 架構

已實作的元件模型、持久化配置、工作流程與安全邊界,詳見[架構文件](./architecture-design.md)。

## 畫面截圖

<table>
<tr>
<td width="50%"><img src="./images/landing_page_overview.png" alt="Workspace overview"><br><sub>工作區總覽 — 一眼掌握 Profile、職缺機會、範本、供應商與已產生的 PDF。</sub></td>
<td width="50%"><img src="./images/profiles.png" alt="Profiles"><br><sub>Profiles — 每個職位方向對應一份聚焦的 Profile,附已儲存內容與語言標記。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/opportunity_list.png" alt="Job Opportunities list"><br><sub>職缺機會清單 — 追蹤狀態與來源,可對任一職缺發起履歷或求職信產生工作。</sub></td>
<td width="50%"><img src="./images/opportunity_detail.png" alt="Job opportunity detail"><br><sub>職缺機會詳情 — 完整說明、地點、工作模式與追蹤狀態。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/job_hunter.png" alt="Job Hunter"><br><sub>Job Hunter — 定期自動搜尋並發掘新的職缺機會。</sub></td>
<td width="50%"><img src="./images/templates.png" alt="Templates"><br><sub>範本 — 內建的 Rezume LaTeX 範本與上傳的自訂範本並列呈現。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv.png" alt="Create CVs and Cover Letters"><br><sub>建立履歷與求職信 — 每個產生工作的狀態、使用模型與範本總覽。</sub></td>
<td width="50%"><img src="./images/opportunity_create_cv.png" alt="Quick CV generation dialog"><br><sub>快速產生 — 直接從某個職缺機會發起履歷或求職信產生。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv_stages.png" alt="Generation progress and PDF preview"><br><sub>即時追蹤產生進度,並附上分階段的 PDF 預覽,此處示範的是一份日文履歷。</sub></td>
<td width="50%"><img src="./images/setting-1.png" alt="Settings: LLM providers"><br><sub>設定 — 已設定的 LLM 供應商,涵蓋雲端與本機。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/setting-2.png" alt="Settings: default models"><br><sub>設定 — 依代理程式角色分別設定的預設模型路由。</sub></td>
<td width="50%"><img src="./images/setting-3.png" alt="Settings: interface language"><br><sub>設定 — 介面語言切換,此處示範簡體中文介面。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/task_mng.png" alt="Task Monitor"><br><sub>Task Monitor — 背景任務歷史紀錄與各任務詳細資訊。</sub></td>
<td width="50%"></td>
</tr>
</table>

## 本機服務堆疊

預設的 Compose 部署包含以下服務:

| 服務 | 用途 | 本機存取方式 |
|---|---|---|
| `web` | 由 Nginx 提供服務的 Vue 3 應用程式 | `http://localhost:5173` |
| `api` | Go 撰寫的 HTTP API | `http://localhost:8080` |
| `worker` | 持久化的背景處理程序與排程器 | 僅供內部使用 |
| `document-worker` | 惡意軟體掃描、內容擷取、OCR、預覽與 PDF 算繪 | 僅供內部使用 |
| `web-worker` | 供 AI 輔助匯入使用的隔離 Playwright 頁面算繪 | 僅供內部使用 |
| `postgres` | 應用程式資料、佇列、事件與設定 | `localhost:5432` |
| `redis` | 供應商模型探索結果的具期限快取 | `localhost:6379` |
| `minio` | 相容於 S3 的物件儲存 | API `localhost:9000`,主控台 `localhost:9001` |
| `migrate` | 一次性的資料庫遷移程序 | 僅供內部使用 |
| `pgadmin` | 用於檢視/除錯 Postgres 資料庫的網頁介面 | `http://localhost:5050` |

檢視狀態或追蹤記錄檔:

```bash
docker compose ps
docker compose logs -f
```

停止服務堆疊但不刪除資料:

```bash
docker compose down
```

PostgreSQL、Redis 與 MinIO 的資料都保留在具名磁碟區中。只有在你確實想清除本機應用程式資料時,才使用 `docker compose down --volumes`。

## 初次使用

1. 開啟 `http://localhost:5173/settings`。
2. 新增一個 OpenAI、OpenAI 相容介面或 Ollama 供應商並測試。
3. 為你預計使用的代理程式選擇預設模型。
4. 建立一個 Profile 並儲存你的來源內容。
5. 新增或匯入一個職缺機會。
6. 建立履歷或求職信,可選擇性指定範本。

若 Ollama 執行在 Docker 主機上,Base URL 請使用 `http://host.docker.internal:11434`。

## 設定

Compose 已提供開發環境的預設值,因此 `.env` 檔案並非必要。若想自訂連接埠、憑證、儲存、驗證或追蹤功能,複製一份 `.env.example`:

```bash
cp .env.example .env
```

主要的設定項目包括:

| 變數 | 用途 |
|---|---|
| `WEB_PORT` | 面向瀏覽器的 Web 連接埠,預設 `5173` |
| `API_PORT` | 面向瀏覽器的 API 連接埠,預設 `8080` |
| `POSTGRES_*` | 本機 PostgreSQL 資料庫及其憑證 |
| `REDIS_PORT`、`MODEL_CACHE_TTL` | Redis 連接埠,以及 LLM 供應商模型快取的有效期 |
| `MINIO_ROOT_USER`、`MINIO_ROOT_PASSWORD` | 本機物件儲存的憑證 |
| `SETTINGS_ENCRYPTION_KEY` | 用於加密已儲存的 LLM API 權杖 |
| `AUTH_MODE` | `development` 或 `oidc` |
| `OIDC_ISSUER`、`OIDC_CLIENT_ID` | 當 `AUTH_MODE=oidc` 時為必填 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 選用的 OpenTelemetry 收集端點 |

內建的加密金鑰與儲存憑證僅供開發環境使用。在本機以外的環境使用本應用程式前,請設定你自己的私有值。

## 開發

主機端開發需要 Go 1.26.8(含)以上、Node.js 22(含)以上、Python 3.12、`uv` 與 GNU Make。

安裝前端與 Python 相依套件:

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

啟動 PostgreSQL 與 MinIO、套用遷移,並在不同終端機視窗分別執行各主要程序:

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

完整的 worker 還需要 document-worker 與 web-worker 搭配運作。若要進行端到端開發,直接使用完整的 Compose 服務堆疊是最簡單的作法。

執行驗證:

```bash
make test
make build
```

常用指令:

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## 儲存庫結構

```text
apps/web/                   Vue 3 + TypeScript 前端
cmd/api/                    Go API 進入點
cmd/worker/                 Go 背景 worker 進入點
cmd/migrate/                內嵌的遷移執行器
internal/                   領域模組、服務、埠(port)與轉接器
migrations/                 版本化的 PostgreSQL 遷移指令碼
services/document-worker/   獨立的文件處理與算繪服務
services/web-worker/        獨立的 Playwright 瀏覽器服務
scripts/                    備份與還原驗證相關指令碼
docs/                       目前的架構文件
```

## 授權

ResumeGPT 依據 [LICENSE](../LICENSE) 中的條款發布。隨附的 Rezume 範本在應用程式內保留其自身的歸屬與授權資訊。
