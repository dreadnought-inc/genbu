# Genbu (玄武)

> [日本語ドキュメントは下部にあります / Japanese documentation is below](#genbu-玄武-1)

---

## English

**Genbu** is a single-binary CLI tool that manages environment variables for cloud and container environments. It reads a YAML config, fetches secrets from cloud providers, validates values, and execs your application — designed to be used as a Docker entrypoint.

### Features

- **YAML-based configuration** — Define env vars, sources, defaults, and validation rules in a single file
- **Cloud provider integration** — AWS SSM Parameter Store, AWS Secrets Manager (GCP/Azure planned)
- **Validation engine** — `required`, `pattern`, `enum`, `min_length`, `max_length`
- **Variable references** — `${VAR}` syntax with automatic dependency resolution and circular reference detection
- **Expression functions** — `${{ sha256(SECRET) }}`, `${{ random_hex(32) }}`, `${{ datetime("rfc3339") }}`, etc.
- **Docker-native** — Uses `syscall.Exec` to replace the process (correct PID 1 / signal handling)
- **Import existing configs** — Convert `.env`, `.ini`, `.toml` files into `genbu.yaml` templates
- **Multiple output formats** — Dump resolved values as dotenv, INI, TOML, or JSON

### Installation

#### From source

```bash
go install github.com/dreadnought-inc/genbu/cmd/genbu@latest
```

#### From release binary

Download from [Releases](https://github.com/dreadnought-inc/genbu/releases).

### Quick Start

#### 1. Create a config file

```yaml
# genbu.yaml
version: "1"

defaults:
  required: true

variables:
  - name: APP_ENV
    value: "production"
    validate:
      enum: ["development", "staging", "production"]

  - name: DB_HOST
    source:
      type: aws-ssm
      path: "/myapp/prod/db-host"

  - name: DB_PORT
    default: "5432"

  - name: DATABASE_URL
    value: "postgres://${DB_HOST}:${DB_PORT}/mydb"
    validate:
      pattern: "^postgres://"

  - name: SESSION_SECRET
    value: "${{ random_hex(32) }}"

  - name: BUILD_DATE
    value: "${{ date() }}"
```

See [genbu.yaml.sample](genbu.yaml.sample) for a comprehensive example.

#### 2. Run your application

```bash
genbu exec -c genbu.yaml -- /app/server
```

#### 3. Use as Docker entrypoint

```dockerfile
COPY genbu /usr/local/bin/genbu
COPY genbu.yaml /etc/genbu.yaml

ENTRYPOINT ["genbu", "exec", "-c", "/etc/genbu.yaml", "--"]
CMD ["/app/server"]
```

### Commands

| Command | Description |
|---------|-------------|
| `genbu exec -- CMD` | Resolve, validate, set env vars, then exec CMD (replaces process) |
| `genbu validate` | Resolve and validate only — exit 0 on success, 1 on failure |
| `genbu dump` | Resolve and print variables (formats: `dotenv`, `ini`, `toml`, `json`) |
| `genbu import FILE` | Generate `genbu.yaml` template from `.env`, `.ini`, or `.toml` |
| `genbu version` | Print version |

```bash
# Validate config in CI
genbu validate -c genbu.yaml

# Dump as TOML with masked values
genbu dump -c genbu.yaml --format toml --mask

# Import an existing .env file
genbu import .env > genbu.yaml
```

### Source Types

| `source.type` | Description |
|----------------|-------------|
| *(none with `value:`)* | Literal value |
| `aws-ssm` | AWS SSM Parameter Store |
| `aws-secretsmanager` | AWS Secrets Manager (`json_key` supported) |
| `env` | Read from existing environment variable |
| *(none, no `value:`)* | Read from current env for validation only |

### Variable References

Reference other variables with `${VAR_NAME}`. Dependencies are resolved automatically via topological sort. Circular references are detected and cause an error.

```yaml
- name: HOST
  value: "db.example.com"
- name: PORT
  default: "5432"
- name: DSN
  value: "postgres://${HOST}:${PORT}/mydb"
```

```
circular reference detected: A -> B -> A
```

### Expression Functions

Use `${{ function(args) }}` to evaluate built-in functions. Functions can be nested.

```yaml
- name: TOKEN
  value: "${{ tohex(bcrypt(random_string(16), 12)) }}"
```

#### Available Functions

| Category | Functions |
|----------|-----------|
| Encoding | `base64encode(s)`, `base64decode(s)`, `tohex(s)`, `fromhex(s)` |
| Hashing | `sha256(s)`, `sha384(s)`, `sha512(s)` |
| Password | `bcrypt(s [, cost])` |
| Random | `random_string(len)`, `random_hex(bytes)` |
| Date/Time | `date([fmt])`, `time([fmt])`, `datetime([fmt])`, `timestamp([fmt])` |
| String | `upper(s)`, `lower(s)`, `trim(s)`, `replace(s, old, new)`, `substr(s, start [, len])`, `concat(a, b, ...)` |

> MD5 and SHA-1 are intentionally excluded as insecure.

#### Date/Time format aliases

`rfc3339`, `iso8601`, `unix`, `unixmilli`, `rfc822`, `rfc850`, `rfc1123`, `kitchen`, `ansic`, `stamp`

Custom formats use [Go's time layout](https://pkg.go.dev/time#pkg-constants):

```yaml
- name: BUILD_DATE
  value: "${{ date('20060102') }}"
- name: DEPLOY_TIME
  value: "${{ datetime('rfc3339') }}"
- name: EPOCH
  value: "${{ timestamp() }}"
```

### Validation

```yaml
defaults:
  required: true

variables:
  - name: PORT
    value: "8080"
    validate:
      required: true
      pattern: "^[0-9]+$"
      enum: ["8080", "8443", "3000"]
      min_length: 1
      max_length: 5

  - name: OPTIONAL_VAR
    validate:
      required: false    # overrides default
```

### Config Reference

```yaml
version: "1"                    # Required. Only "1" is supported.
dump_format: dotenv             # Default dump output format (dotenv/ini/toml/json)

defaults:
  required: true                # Default required setting for all variables

variables:
  - name: VAR_NAME              # Required. Environment variable name.
    value: "literal"            # Literal value (supports ${REF} and ${{ expr }})
    default: "fallback"         # Fallback when resolved value is empty
    source:
      type: aws-ssm            # Provider type
      path: "/param/path"      # SSM parameter path
      secret_id: "secret-name" # Secrets Manager secret ID
      json_key: "key"          # Extract key from JSON secret
      region: "ap-northeast-1" # AWS region override
    validate:
      required: true
      pattern: "^prefix"
      enum: ["a", "b", "c"]
      min_length: 1
      max_length: 256

groups:
  - name: group-name
    source:                     # Shared source config (inherited by variables)
      type: aws-ssm
      region: "ap-northeast-1"
    variables:
      - name: GROUPED_VAR
        source:
          path: "/param/path"   # Merged with group source
```

### Development

```bash
make build       # Build binary to bin/genbu
make test        # Run all tests with race detector
make test-cover  # Run tests with coverage report
make lint        # Run golangci-lint
make fmt         # Format code
make vet         # Run go vet
make clean       # Remove build artifacts
```

### License

[MIT License](LICENSE) - Copyright (c) 2026 Dreadnought, Inc.

---

## Genbu (玄武)

**Genbu** は、クラウド・コンテナ環境向けの環境変数管理CLIツールです。YAML設定ファイルに基づいて、AWS Parameter Store や Secrets Manager からの値の取得、バリデーション、アプリケーションの起動を一括で行います。Docker の entrypoint として最適なシングルバイナリで提供されます。

### 主な機能

- **YAML設定ファイル** — 環境変数の定義、取得元、デフォルト値、バリデーションルールを1ファイルに集約
- **クラウドプロバイダ連携** — AWS SSM Parameter Store / Secrets Manager 対応（GCP/Azure は将来対応予定）
- **バリデーション** — `required`, `pattern`（正規表現）, `enum`, `min_length`, `max_length`
- **変数参照** — `${VAR}` で他の変数を参照。依存関係を自動解決し、循環参照を検出
- **関数式** — `${{ sha256(SECRET) }}`, `${{ random_hex(32) }}`, `${{ datetime("rfc3339") }}` など
- **Docker対応** — `syscall.Exec` によるプロセス置換で PID 1 のシグナル処理に対応
- **既存設定のインポート** — `.env`, `.ini`, `.toml` から `genbu.yaml` テンプレートを生成
- **多フォーマット出力** — dotenv, INI, TOML, JSON 形式で値を出力

### インストール

#### ソースから

```bash
go install github.com/dreadnought-inc/genbu/cmd/genbu@latest
```

#### リリースバイナリから

[Releases](https://github.com/dreadnought-inc/genbu/releases) からダウンロードしてください。

### クイックスタート

#### 1. 設定ファイルを作成

```yaml
# genbu.yaml
version: "1"

defaults:
  required: true

variables:
  - name: APP_ENV
    value: "production"
    validate:
      enum: ["development", "staging", "production"]

  - name: DB_HOST
    source:
      type: aws-ssm
      path: "/myapp/prod/db-host"

  - name: DB_PORT
    default: "5432"

  - name: DATABASE_URL
    value: "postgres://${DB_HOST}:${DB_PORT}/mydb"
    validate:
      pattern: "^postgres://"

  - name: SESSION_SECRET
    value: "${{ random_hex(32) }}"

  - name: BUILD_DATE
    value: "${{ date() }}"
```

すべてのオプションを網羅したサンプルは [genbu.yaml.sample](genbu.yaml.sample) を参照してください。

#### 2. アプリケーションを実行

```bash
genbu exec -c genbu.yaml -- /app/server
```

#### 3. Docker entrypoint として利用

```dockerfile
COPY genbu /usr/local/bin/genbu
COPY genbu.yaml /etc/genbu.yaml

ENTRYPOINT ["genbu", "exec", "-c", "/etc/genbu.yaml", "--"]
CMD ["/app/server"]
```

### コマンド一覧

| コマンド | 説明 |
|---------|------|
| `genbu exec -- CMD` | 変数を解決・バリデーション後、CMD をプロセス置換で実行 |
| `genbu validate` | 変数の解決とバリデーションのみ実行（成功: 0, 失敗: 1） |
| `genbu dump` | 変数を解決して出力（形式: `dotenv`, `ini`, `toml`, `json`） |
| `genbu import FILE` | `.env` / `.ini` / `.toml` から `genbu.yaml` テンプレートを生成 |
| `genbu version` | バージョン表示 |

```bash
# CI でバリデーション
genbu validate -c genbu.yaml

# TOML形式でマスク付き出力
genbu dump -c genbu.yaml --format toml --mask

# .env ファイルからインポート
genbu import .env > genbu.yaml
```

### ソースタイプ

| `source.type` | 説明 |
|----------------|------|
| *（`value:` 指定時は省略）* | リテラル値 |
| `aws-ssm` | AWS SSM Parameter Store |
| `aws-secretsmanager` | AWS Secrets Manager（`json_key` でJSONキー抽出対応） |
| `env` | 既存の環境変数を読み取り |
| *（`source` / `value` ともに省略）* | 現在の環境変数を読み取り（バリデーション専用） |

### 変数参照

`${VAR_NAME}` で他の変数を参照できます。依存関係はトポロジカルソートで自動解決されます。

```yaml
- name: HOST
  value: "db.example.com"
- name: PORT
  default: "5432"
- name: DSN
  value: "postgres://${HOST}:${PORT}/mydb"
```

循環参照が検出された場合はエラーで終了します:

```
circular reference detected: A -> B -> A
```

### 関数式

`${{ 関数(引数) }}` で組み込み関数を評価できます。入れ子も可能です。

```yaml
- name: TOKEN
  value: "${{ tohex(bcrypt(random_string(16), 12)) }}"
```

#### 利用可能な関数

| カテゴリ | 関数 |
|----------|------|
| エンコード | `base64encode(s)`, `base64decode(s)`, `tohex(s)`, `fromhex(s)` |
| ハッシュ | `sha256(s)`, `sha384(s)`, `sha512(s)` |
| パスワード | `bcrypt(s [, cost])` |
| ランダム | `random_string(len)`, `random_hex(bytes)` |
| 日時 | `date([fmt])`, `time([fmt])`, `datetime([fmt])`, `timestamp([fmt])` |
| 文字列 | `upper(s)`, `lower(s)`, `trim(s)`, `replace(s, old, new)`, `substr(s, start [, len])`, `concat(a, b, ...)` |

> MD5 と SHA-1 は脆弱なため意図的に除外しています。

#### 日時フォーマットエイリアス

`rfc3339`, `iso8601`, `unix`, `unixmilli`, `rfc822`, `rfc850`, `rfc1123`, `kitchen`, `ansic`, `stamp`

カスタムフォーマットは [Go の time レイアウト](https://pkg.go.dev/time#pkg-constants)で指定します:

```yaml
- name: BUILD_DATE
  value: "${{ date('20060102') }}"
- name: DEPLOY_TIME
  value: "${{ datetime('rfc3339') }}"
- name: EPOCH
  value: "${{ timestamp() }}"
```

### バリデーション

```yaml
defaults:
  required: true    # 全変数に適用

variables:
  - name: PORT
    value: "8080"
    validate:
      required: true       # 空でないこと
      pattern: "^[0-9]+$"  # 正規表現マッチ
      enum: ["8080", "8443", "3000"]
      min_length: 1
      max_length: 5

  # デフォルトを上書きしてオプショナルにする
  - name: OPTIONAL_VAR
    validate:
      required: false
```

### 設定リファレンス

```yaml
version: "1"                    # 必須。現在は "1" のみ対応。
dump_format: dotenv             # dump時のデフォルト出力形式 (dotenv/ini/toml/json)

defaults:
  required: true                # 全変数に適用されるデフォルトの required 設定

variables:
  - name: VAR_NAME              # 必須。環境変数名。
    value: "literal"            # リテラル値（${REF} や ${{ expr }} を使用可能）
    default: "fallback"         # 解決された値が空の場合のフォールバック値
    source:
      type: aws-ssm            # プロバイダタイプ
      path: "/param/path"      # SSM パラメータパス
      secret_id: "secret-name" # Secrets Manager シークレットID
      json_key: "key"          # JSONシークレットからのキー抽出
      region: "ap-northeast-1" # AWSリージョン指定（オプション）
    validate:
      required: true
      pattern: "^prefix"
      enum: ["a", "b", "c"]
      min_length: 1
      max_length: 256

groups:
  - name: group-name
    source:                     # 共通ソース設定（配下の変数に継承される）
      type: aws-ssm
      region: "ap-northeast-1"
    variables:
      - name: GROUPED_VAR
        source:
          path: "/param/path"   # グループのsource設定とマージされる
```

### 開発

```bash
make build       # bin/genbu にビルド
make test        # 全テスト実行（race detector付き）
make test-cover  # カバレッジレポート付きテスト
make lint        # golangci-lint 実行
make fmt         # コード整形
make vet         # go vet 実行
make clean       # ビルド成果物を削除
```

### ライセンス

[MIT License](LICENSE) - Copyright (c) 2026 Dreadnought, Inc.
