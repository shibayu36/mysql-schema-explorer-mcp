# テスト方針

## 概要
このドキュメントでは、`mysql-schema-mcp` のハンドラー機能に対するテスト方針を記述します。
まずは `ListTables` メソッドの正常系テストから開始します。

## テスト環境

### 新しいアプローチ
- ローカル開発環境とCI環境で `docker-compose up -d` コマンド等で起動したMySQLコンテナを使用する
  - ローカル開発環境: 開発者が `docker-compose.yml` を元に起動したMySQLコンテナ
  - CI環境: GitHub Actions のワークフロー内で `docker-compose up -d` を実行してMySQLコンテナを起動
- 各テストケースごとに一意のデータベースを作成し、テスト終了後に削除する
- テスト用DB接続情報は本番環境と同じ環境変数を使用
  - `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`
  - `docker-compose.yml` で設定された値に対応する環境変数を設定してテストを実行する。（例: `DB_HOST=localhost`, `DB_PORT=13306`, `DB_USER=root`, `DB_PASSWORD=rootpass`）

### テスト向けユーティリティ
- `setupTestDB`: `docker-compose` で起動したMySQLインスタンスに接続し、テスト用のデータベースを作成して接続を返す。テスト終了時にデータベースを削除するクリーンアップ処理も登録する。
- `applySchema`: SQLファイルからスキーマを適用

## 1. `ListTables` メソッド テスト

### 1.1. テスト目的
- `docs/requirements.md` に記載された出力フォーマットに従って、テーブル情報が正しく整形されて返却されることを確認する。
- 単一キー、複合キー（PK, UK, FK）、複数のキー制約（とくにUK）を含むさまざまなテーブル構成に対応できることを確認する。

### 1.2. テスト環境
- `docker-compose` で起動したMySQLサーバーに対してテストを実行する
- テストの検証には以下のライブラリを使用する
  - `stretchr/testify`: アサーション（期待値との比較）を容易にするため
- テストの準備（テストDB作成、スキーマ適用）と後片付け（DB削除）を行うヘルパー関数 (`setupTestDB`) を使用する

### 1.3. テストケース: 複合キーおよび複数UKを含むテーブル構成

#### 1.3.1. テスト用テーブルスキーマ
以下のCREATE TABLE文で定義される4つのテーブルを持つデータベースを各テストごとに用意する。

```sql
-- orders テーブル（order_items からの参照用）
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '注文ID',
    order_date DATETIME COMMENT '注文日時',
    INDEX(id) -- 複合外部キーの参照先としてインデックスが必要
) COMMENT='注文ヘッダー';

-- users テーブル
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT 'ユーザーシステムID',
    email VARCHAR(255) NOT NULL UNIQUE COMMENT 'メールアドレス',
    username VARCHAR(255) NOT NULL UNIQUE COMMENT 'ユーザー名',
    tenant_id INT NOT NULL COMMENT 'テナントID',
    employee_id INT NOT NULL COMMENT '従業員ID',
    UNIQUE KEY uk_tenant_employee (tenant_id, employee_id) -- 複合一意キー
) COMMENT='ユーザー情報';

-- products テーブル
CREATE TABLE products (
    product_code VARCHAR(50) PRIMARY KEY COMMENT '商品コード（主キー）',
    maker_code VARCHAR(50) NOT NULL COMMENT 'メーカーコード',
    internal_code VARCHAR(50) NOT NULL COMMENT '社内商品コード',
    product_name VARCHAR(255) COMMENT '商品名', -- 例としてカラム追加
    UNIQUE KEY uk_maker_internal (maker_code, internal_code), -- 複合一意キー
    INDEX idx_product_name (product_name),                 -- 単一カラムインデックス
    INDEX idx_maker_product_name (maker_code, product_name) -- 複合インデックスを追加
) COMMENT='商品マスター';

-- order_items テーブル
CREATE TABLE order_items (
    order_id INT NOT NULL COMMENT '注文ID (FK)',
    item_seq INT NOT NULL COMMENT '注文明細連番',
    product_maker VARCHAR(50) NOT NULL COMMENT '商品メーカーコード (FK)',
    product_internal_code VARCHAR(50) NOT NULL COMMENT '商品社内コード (FK)',
    quantity INT NOT NULL COMMENT '数量', -- 例としてカラム追加
    PRIMARY KEY (order_id, item_seq), -- 複合主キー
    UNIQUE KEY uk_order_product (order_id, product_maker, product_internal_code), -- 複合一意キー （注文内での同一商品は許さない）
    FOREIGN KEY fk_order (order_id) REFERENCES orders(id) ON DELETE CASCADE, -- 単一外部キー （注文が消えたら明細も消す例）
    FOREIGN KEY fk_product (product_maker, product_internal_code) REFERENCES products(maker_code, internal_code) -- 複合外部キー
) COMMENT='注文明細';
```

#### 1.3.2. 実行手順
1. `docker-compose up -d` コマンド等でテスト用のMySQLコンテナを起動する
2. テストコード内で `setupTestDB` を呼び出し、テスト用DBを作成し、スキーマを適用する
3. `ListTables` ツールを `dbName: "{テスト用DB名}"` で呼び出す
4. 返却されたテキストが期待される出力と一致することを確認する
5. テスト完了後、`setupTestDB` 内で登録されたクリーンアップ処理によりテスト用DBが自動的に削除される
6. (必要に応じて `docker-compose down` でコンテナを停止する)

#### 1.3.3. 期待される出力

```
データベース「{テスト用DB名}」のテーブル一覧 (全4件)
フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
※ 複数の異なるキー制約はセミコロンで区切り: key1; key2

- order_items - 注文明細 [PK: (order_id, item_seq)] [UK: (order_id, product_maker, product_internal_code)] [FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
- orders - 注文ヘッダー [PK: id]
- products - 商品マスター [PK: product_code] [UK: (maker_code, internal_code)]
- users - ユーザー情報 [PK: id] [UK: email; (tenant_id, employee_id); username]
```

## 2. `DescribeTables` メソッド テスト

### 2.1. テスト目的
- `docs/requirements.md` に記載された出力フォーマットに従って、テーブルの詳細情報（カラム、キー、インデックス、コメント）が正しく整形されて返却されることを確認する。
- 複数のテーブルを同時に指定した場合に、各テーブルの情報が `---` で区切られて正しく出力されることを確認する。
- 単一および複合キー（PK, UK, FK）、単一および複合インデックス、複数のキー/インデックスが存在するケースに対応できることを確認する。

### 2.2. テスト環境
- `ListTables` メソッドのテストと同様に、`docker-compose` で起動したMySQLサーバーと `setupTestDB` ヘルパーを使用する。

### 2.3. テスト用スキーマ
- `ListTables` メソッドのテストで使用した `testdata/schema.sql` をベースとし、以下のインデックスを追加する。
  - `products` テーブルに単一カラムインデックス `INDEX idx_product_name (product_name)`
  - `products` テーブルに複合カラムインデックス `INDEX idx_maker_product_name (maker_code, product_name)`

```sql
-- products テーブルへの追加箇所
CREATE TABLE products (
    product_code VARCHAR(50) PRIMARY KEY COMMENT '商品コード（主キー）',
    maker_code VARCHAR(50) NOT NULL COMMENT 'メーカーコード',
    internal_code VARCHAR(50) NOT NULL COMMENT '社内商品コード',
    product_name VARCHAR(255) COMMENT '商品名',
    UNIQUE KEY uk_maker_internal (maker_code, internal_code), -- 複合一意キー
    INDEX idx_product_name (product_name),                 -- 追加: 単一カラムインデックス
    INDEX idx_maker_product_name (maker_code, product_name) -- 追加: 複合インデックス
) COMMENT='商品マスター';
```

### 2.4. テストケース

#### 2.4.1. 主要ケース: 複数テーブル指定
- **実行手順**: `DescribeTables` ツールを `dbName: "{テスト用DB名}"`, `tableNames: ["users", "products", "order_items"]` で呼び出す。
- **確認項目**: 各テーブルのカラム情報、キー情報（PK, UK, FK）、インデックス情報、テーブルコメントが正しく出力されること。`products` テーブルでは複数のインデックス（単一、複合）が、`users` テーブルでは複数のUK（単一、複合）が表示されること。各テーブル情報が `---` で区切られていること。
- **期待される出力の骨子**:

```
# テーブル: users - ユーザー情報

## カラム
- id: int(11) NOT NULL [ユーザーシステムID]
- email: varchar(255) NOT NULL [メールアドレス]
- username: varchar(255) NOT NULL [ユーザー名]
- tenant_id: int(11) NOT NULL [テナントID]
- employee_id: int(11) NOT NULL [従業員ID]

## キー情報
[PK: id]
[UK: email; (tenant_id, employee_id); username]

---

# テーブル: products - 商品マスター

## カラム
- product_code: varchar(50) NOT NULL [商品コード（主キー）]
- maker_code: varchar(50) NOT NULL [メーカーコード]
- internal_code: varchar(50) NOT NULL [社内商品コード]
- product_name: varchar(255) NULL [商品名]

## キー情報
[PK: product_code]
[UK: (maker_code, internal_code)]
[INDEX: product_name; (maker_code, product_name)]

---

# テーブル: order_items - 注文明細

## カラム
- order_id: int(11) NOT NULL [注文ID (FK)]
- item_seq: int(11) NOT NULL [注文明細連番]
- product_maker: varchar(50) NOT NULL [商品メーカーコード (FK)]
- product_internal_code: varchar(50) NOT NULL [商品社内コード (FK)]
- quantity: int(11) NOT NULL [数量]

## キー情報
[PK: (order_id, item_seq)]
[UK: (order_id, product_maker, product_internal_code)]
[FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
```
*(注: カラムのNULL制約、デフォルト値、インデックスの順序などは実行環境によって若干異なる可能性があるため、テスト実装時に実環境での出力を確認し、適切に調整すること)*

#### 2.4.2. 追加ケース
- **存在しないテーブル**: `tableNames` に存在しないテーブル名を含めて呼び出し、「テーブルが見つかりません」というメッセージが正しく出力されることを確認する。
- **キー情報がないテーブル**: (必要に応じてスキーマにキーを持たないテーブルを追加し) キー情報セクションが出力されないことを確認する。
- **引数エラー**: `dbName` や `tableNames` が指定されない、またはフォーマットが不正な場合に `mcp.NewToolResultError` が返されることを確認する。

## 3. 今後のテスト項目（検討中）
- エラーケースのテスト（DB接続エラーなど）
- 特殊文字を含むテーブル/カラム名/コメントのテスト

## 4. 実装 TODO リスト

1. [x] テストヘルパーファイル (`testhelper_test.go`) を作成し、`setupTestDB` 関数を実装する (スキーマ適用も含む)
2. [x] テストデータのSQLファイル (`testdata/schema.sql`) を作成する
3. [x] `handler_test.go` を作成し、`TestListTables` 関数を実装する
4. [x] ローカル環境でテストを実行し確認する (`docker-compose` でMySQLを起動しておく必要がある)
5. [ ] `testdata/schema.sql` を更新し、`products` テーブルにインデックスを追加する。
6. [ ] `handler_test.go` に `TestDescribeTables` 関数を追加し、主要ケース（複数テーブル指定）を実装する。
7. [ ] GitHub Actions用の設定ファイルを作成して、CI環境でもテストが実行できることを確認する (`docker-compose` を利用する設定を追加)
