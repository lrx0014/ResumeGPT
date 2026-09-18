# ResumeGPT

<p align="center">
  <img src="./assets/resumegpt-logo.png" alt="ResumeGPT Ré logo" width="128" height="128">
</p>

<p align="center">
  <a href="../README.md">English</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.es.md">Español</a> ·
  <b>日本語</b> ·
  <a href="./README.zh-CN.md">简体中文</a> ·
  <a href="./README.zh-TW.md">繁體中文</a>
</p>

ResumeGPT は、履歴書と職務経歴書・カバーレターを個々の求人に合わせて作成するための、セルフホスト型ワークスペースです。プロフィール管理、求人トラッキング、定期的な求人検索、再利用可能なテンプレート、設定可能な LLM エージェント、PDF 生成、ビジュアルレビュー、バックグラウンドタスクの監視を1つのアプリケーションにまとめています。

## TL;DR

Docker Compose をインストール済みであれば、次の1つのコマンドでアプリケーション全体を起動できます。

```bash
docker compose up -d
```

これは CI が公開しているビルド済みの `latest` イメージを取得するもので([`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml) を参照)、ローカルでビルドしないため数秒でスタックが起動します。コードを編集していて、`api`・`worker`・`document-worker`・`web-worker`・`web` の各イメージをローカルのチェックアウトから Compose にビルドさせたい場合は、`--build` を付けてください。

```bash
docker compose up -d --build
```

いずれの場合も、スタックの準備が整ったら [http://localhost:5173](http://localhost:5173) を開いてください。Compose スタックは開発用のデフォルト値を提供し、ストレージ用ボリュームを作成し、データベースマイグレーションを適用し、依存サービスが健全になるまで待機します。

## 機能

### プロフィール

- 名前、目標とする職種、既定言語、Markdown 対応のコンテンツを持つ複数のプロフィールを作成・管理します。
- プロフィールの内容を直接入力するか、TXT、Markdown、TeX、DOC、DOCX、PDF、PNG、JPEG ファイルからインポートします。
- プロフィールに保存する前に、抽出されたテキストを確認・編集します。
- 写真対応のレイアウト向けに、JPEG または PNG のアバターを任意で添付します。
- Web インターフェースからプロフィールを検索・フィルタ・ページ送り・編集・削除します。

### 求人機会

- 職種、会社名、勤務地、国、市区町村、勤務形態、雇用形態、ソース URL、説明、応募状況を記録します。
- 求人を手動で作成するか、一度に最大 50 件の公開求人 URL をインポートします。
- 対応済みの LinkedIn および Indeed のページには、高速な決定的抽出を使用します。
- その他の公開 HTTPS 求人ページからの AI 支援抽出には、求人インポートエージェントを有効化します。
- インポートされた情報を編集し、求人一覧から直接応募状況を更新します。
- 手動追加・URL インポート・Job Hunter による結果を区別できるよう、由来で求人をフィルタします。
- 複数の求人を選択し、履歴書またはカバーレターの作成タスクを一括で作成します。

### Job Hunter

- 職種、勤務地、勤務形態、契約形態、経験年数、キーワード、追加のプロンプトを指定して、定期検索をスケジュールします。
- 任意で保存済みのプロフィールをマッチングのコンテキストとして使用します。
- 各実行で新規に見つかる求人を最大 10 件までに制限します。
- Web インターフェースから Hunter を実行・一時停止・再開・編集・削除します。
- 求人を作成する前に、発見した URL を重複排除します。
- ブロックされた結果や解析できなかった結果は求人一覧に載せず、手動レビュー用の確認用受信箱にまとめます。

### テンプレート

- 履歴書用とカバーレター用のテンプレートを別々に管理します。
- DOC、DOCX、単一ファイルの TeX、または複数ファイル構成の LaTeX ZIP プロジェクトをアップロードします。
- ZIP に一意な `main.tex` が含まれない場合は、LaTeX のエントリファイルを指定します。
- アップロードされたソースをスキャンし、LLM が読み取れる形式のコンテンツを抽出し、キャッシュされた PDF プレビューを生成します。
- カスタムテンプレートの表示・更新・ダウンロード・削除を行います。
- 同梱の読み取り専用 Rezume LaTeX テンプレートを使用できます。元の [Overleaf ソース](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs) へのクレジット表記付きです。

### 履歴書・カバーレターの生成

- 保存済みのプロフィールと求人情報から、その求人に合わせた履歴書またはカバーレターを生成します。
- 任意で LaTeX テンプレートを使用するか、テンプレートを指定せずに Document Designer エージェントに印刷対応の HTML/CSS レイアウトを作成させます。
- エージェントごとに異なる既定の LLM を設定するか、1つのモデルでワークフロー全体の生成を上書きします。
- Writer、Template Applying または Document Designer、Visual Reviewer の各ロールを、制限付きの LangChainGo エージェント実行器で実行します。
- 分離されたレンダリングツールを通じて LaTeX と HTML/CSS の成果物、および Word テンプレートのプレビューを生成します。
- ビジュアルレビューのために PDF ページをラスタライズし、最大 2 回まで自動レイアウト修正を行います。
- 選択したモデルがビジュアル検査に対応していない場合でも、有効な PDF を目立つ警告とともに保持します。
- AI 生成のドキュメントが確実にレンダリングできない場合は、安全な基本レイアウトにフォールバックします。
- 生成の進行状況を追跡し、下書き、途中経過の PDF、レビューのフィードバック、警告、ユーザーからの指示をタイムラインで確認します。
- 完了または失敗した生成の入力内容を編集し、それまでのタイムライン記録を失うことなく再生成します。
- 完了した成果物を修正するため、追加の指示を送信します。

### LLM プロバイダーと設定

- OpenAI、OpenAI 互換、ローカル Ollama の各プロバイダーを設定します。
- API トークンは暗号化して保存し、平文のトークンをブラウザに返すことはありません。
- `/models` または Ollama の `/api/tags` エンドポイントから利用可能なモデルを自動検出します。
- Writer、Template Applying、Document Designer、Visual Reviewer、Job Import、Job Hunter の各エージェントに、既定のプロバイダーとモデルを個別に割り当てます。
- 個々の生成に対して、既定のルーティングを単一モデルで上書きします。
- インターフェースの言語、およびシステム・ライト・ダークのテーマを設定します。
- 設定画面から、英語、ドイツ語、フランス語、スペイン語、日本語、簡体字中国語、繁体字中国語の間でインターフェースを切り替えます。

### バックグラウンド処理と運用

- ドキュメント抽出、テンプレート準備、求人インポート、Job Hunting、ドキュメント生成を、PostgreSQL を基盤とする永続的なワークキューを通じて実行します。
- リース切れのジョブを回復し、一時的な失敗を再試行し、タスクのライフサイクルイベントを永続化します。
- Task Monitor でタスクの状態、試行回数、サニタイズ済みの入力、エラー、ログを確認します。
- ソースファイル、アバター、プレビュー、生成された PDF を S3 互換のオブジェクトストレージに保存します。
- 主要な変更に対して、トランザクション整合の Outbox と監査レコードを永続化します。
- W3C トレースコンテキスト伝播に対応した構造化ログと OpenTelemetry トレースを出力します。
- 付属のスクリプトを使って PostgreSQL のバックアップを作成し、自動化されたリストア確認を実行します。

## アーキテクチャ

実装済みのコンポーネントモデル、永続化レイアウト、ワークフロー、セキュリティ境界については [アーキテクチャ](./architecture-design.md) を参照してください。

## スクリーンショット

<table>
<tr>
<td width="50%"><img src="./images/landing_page_overview.png" alt="Workspace overview"><br><sub>ワークスペース概要 — プロフィール、求人、テンプレート、プロバイダー、生成済み PDF を一目で確認できます。</sub></td>
<td width="50%"><img src="./images/profiles.png" alt="Profiles"><br><sub>プロフィール — 職種ごとに絞り込んだプロフィールと、保存済みコンテンツ・言語。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/opportunity_list.png" alt="Job Opportunities list"><br><sub>求人機会 — 状況とソースを管理し、任意の求人から履歴書やカバーレターの作成をキューに追加できます。</sub></td>
<td width="50%"><img src="./images/opportunity_detail.png" alt="Job opportunity detail"><br><sub>求人詳細 — 完全な説明文、勤務地、勤務形態、トラッキング状況。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/job_hunter.png" alt="Job Hunter"><br><sub>Job Hunter — 新しい求人を自動的に見つけるスケジュール検索。</sub></td>
<td width="50%"><img src="./images/templates.png" alt="Templates"><br><sub>テンプレート — 同梱の Rezume LaTeX テンプレートと、アップロードしたカスタムテンプレート。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv.png" alt="Create CVs and Cover Letters"><br><sub>履歴書・カバーレター作成 — 各生成の状況、モデル、テンプレートの一覧。</sub></td>
<td width="50%"><img src="./images/opportunity_create_cv.png" alt="Quick CV generation dialog"><br><sub>クイック生成 — 求人情報から直接、履歴書またはカバーレターの作成を開始します。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/create_cv_stages.png" alt="Generation progress and PDF preview"><br><sub>段階的な PDF プレビュー付きの生成進行状況(ここでは日本語の履歴書を表示)。</sub></td>
<td width="50%"><img src="./images/setting-1.png" alt="Settings: LLM providers"><br><sub>設定 — クラウドおよびローカルで設定済みの LLM プロバイダー。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/setting-2.png" alt="Settings: default models"><br><sub>設定 — エージェントロールごとの既定モデルのルーティング。</sub></td>
<td width="50%"><img src="./images/setting-3.png" alt="Settings: interface language"><br><sub>設定 — インターフェース言語の切り替え(簡体字中国語表示の例)。</sub></td>
</tr>
<tr>
<td width="50%"><img src="./images/task_mng.png" alt="Task Monitor"><br><sub>Task Monitor — バックグラウンドタスクの履歴とタスクごとの詳細。</sub></td>
<td width="50%"></td>
</tr>
</table>

## ローカルスタック

デフォルトの Compose デプロイでは、次のサービスが実行されます。

| サービス | 役割 | ローカルアクセス |
|---|---|---|
| `web` | Nginx で配信される Vue 3 アプリケーション | `http://localhost:5173` |
| `api` | Go 製の HTTP API | `http://localhost:8080` |
| `worker` | 永続的なバックグラウンド処理とスケジューラー | 内部専用 |
| `document-worker` | マルウェアスキャン、抽出、OCR、プレビュー、PDF レンダリング | 内部専用 |
| `web-worker` | AI 支援インポート用の分離された Playwright ページレンダリング | 内部専用 |
| `postgres` | アプリケーションデータ、キュー、イベント、設定 | `localhost:5432` |
| `redis` | プロバイダーのモデル検出用の有効期限付きキャッシュ | `localhost:6379` |
| `minio` | S3 互換のオブジェクトストレージ | API `localhost:9000`、コンソール `localhost:9001` |
| `migrate` | 一度きりのデータベースマイグレーション処理 | 内部専用 |
| `pgadmin` | Postgres データベースを確認・デバッグする Web UI | `http://localhost:5050` |

状態確認やログの追跡には次を使用します。

```bash
docker compose ps
docker compose logs -f
```

データを削除せずにスタックを停止するには:

```bash
docker compose down
```

PostgreSQL、Redis、MinIO のデータは名前付きボリュームに保持されます。ローカルのアプリケーションデータを意図的に消去したい場合にのみ `docker compose down --volumes` を使用してください。

## はじめての利用

1. `http://localhost:5173/settings` を開きます。
2. OpenAI、OpenAI 互換、または Ollama のプロバイダーを追加してテストします。
3. 利用予定のエージェントに既定モデルを選択します。
4. プロフィールを作成し、元となるコンテンツを保存します。
5. 求人情報を追加またはインポートします。
6. 任意でテンプレートを選択して、履歴書またはカバーレターを作成します。

Docker ホスト上で動作している Ollama を使う場合は、ベース URL として `http://host.docker.internal:11434` を使用してください。

## 設定

Compose は開発用のデフォルト値を提供するため、`.env` ファイルは任意です。ポート、認証情報、ストレージ、認証、トレースをカスタマイズしたい場合は `.env.example` をコピーしてください。

```bash
cp .env.example .env
```

主な設定項目は次のとおりです。

| 変数 | 用途 |
|---|---|
| `WEB_PORT` | ブラウザ向け Web ポート。デフォルトは `5173` |
| `API_PORT` | ブラウザ向け API ポート。デフォルトは `8080` |
| `POSTGRES_*` | ローカル PostgreSQL データベースと認証情報 |
| `REDIS_PORT`、`MODEL_CACHE_TTL` | Redis のポートと LLM プロバイダーのモデルキャッシュの有効期間 |
| `MINIO_ROOT_USER`、`MINIO_ROOT_PASSWORD` | ローカルオブジェクトストレージの認証情報 |
| `SETTINGS_ENCRYPTION_KEY` | 保存された LLM API トークンを暗号化 |
| `AUTH_MODE` | `development` または `oidc` |
| `OIDC_ISSUER`、`OIDC_CLIENT_ID` | `AUTH_MODE=oidc` の場合に必須 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 任意の OpenTelemetry コレクターのエンドポイント |

組み込みの暗号化キーとストレージ認証情報は、開発用のみのデフォルト値です。ローカルマシン以外でアプリケーションを使用する前に、非公開の値を設定してください。

## 開発

ホスト環境での開発には、Go 1.26.8 以降、Node.js 22 以降、Python 3.12、`uv`、GNU Make が必要です。

フロントエンドと Python の依存関係をインストールします。

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

PostgreSQL と MinIO を起動し、マイグレーションを適用し、主要なプロセスを別々のターミナルで実行します。

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

フルワーカーは document-worker と web-worker も必要とします。エンドツーエンドの開発には、完全な Compose スタックを使用するのが最も簡単です。

検証を実行します。

```bash
make test
make build
```

便利なコマンド:

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## リポジトリ構成

```text
apps/web/                   Vue 3 と TypeScript によるフロントエンド
cmd/api/                    Go 製 API のエントリポイント
cmd/worker/                 Go 製バックグラウンドワーカーのエントリポイント
cmd/migrate/                組み込みのマイグレーションランナー
internal/                   ドメインモジュール、サービス、ポート、アダプター
migrations/                 バージョン管理された PostgreSQL マイグレーション
services/document-worker/   分離されたドキュメント処理・レンダリングサービス
services/web-worker/        分離された Playwright ブラウザサービス
scripts/                    バックアップおよびリストア確認用ユーティリティ
docs/                       最新のアーキテクチャドキュメント
```

## ライセンス

ResumeGPT は [LICENSE](../LICENSE) に記載された条件のもとで配布されています。同梱の Rezume テンプレートは、アプリケーション内に独自のクレジット表記とライセンスメタデータを保持しています。
