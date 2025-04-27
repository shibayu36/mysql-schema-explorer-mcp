-- users テーブル
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT 'ユーザーシステムID',
    email VARCHAR(255) NOT NULL UNIQUE COMMENT 'メールアドレス',
    username VARCHAR(255) NOT NULL UNIQUE COMMENT 'ユーザー名',
    tenant_id INT NOT NULL COMMENT 'テナントID',
    employee_id INT NOT NULL COMMENT '従業員ID',
    UNIQUE KEY uk_tenant_employee (tenant_id, employee_id) -- 複合一意キー
) COMMENT='ユーザー情報';

-- orders テーブル（order_items からの参照用）
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '注文ID',
    user_id INT NOT NULL COMMENT 'ユーザーID (FK)',
    order_date DATETIME COMMENT '注文日時',
    INDEX(id), -- 複合外部キーの参照先としてインデックスが必要
    FOREIGN KEY fk_user (user_id) REFERENCES users(id)
) COMMENT='注文ヘッダー';

-- products テーブル
CREATE TABLE products (
    product_code VARCHAR(50) PRIMARY KEY COMMENT '商品コード（主キー）',
    maker_code VARCHAR(50) NOT NULL COMMENT 'メーカーコード',
    internal_code VARCHAR(50) NOT NULL COMMENT '社内商品コード',
    product_name VARCHAR(255) COMMENT '商品名',
    UNIQUE KEY uk_maker_internal (maker_code, internal_code), -- 複合一意キー
    INDEX idx_product_name (product_name),
    INDEX idx_maker_product_name (maker_code, product_name)
) COMMENT='商品マスター';

-- order_items テーブル
CREATE TABLE order_items (
    order_id INT NOT NULL COMMENT '注文ID (FK)',
    item_seq INT NOT NULL COMMENT '注文明細連番',
    product_maker VARCHAR(50) NOT NULL COMMENT '商品メーカーコード (FK)',
    product_internal_code VARCHAR(50) NOT NULL COMMENT '商品社内コード (FK)',
    quantity INT NOT NULL COMMENT '数量',
    PRIMARY KEY (order_id, item_seq), -- 複合主キー
    UNIQUE KEY uk_order_product (order_id, product_maker, product_internal_code), -- 複合一意キー （注文内での同一商品は許さない）
    FOREIGN KEY fk_order (order_id) REFERENCES orders(id) ON DELETE CASCADE, -- 単一外部キー （注文が消えたら明細も消す例）
    FOREIGN KEY fk_product (product_maker, product_internal_code) REFERENCES products(maker_code, internal_code) -- 複合外部キー
) COMMENT='注文明細'; 
