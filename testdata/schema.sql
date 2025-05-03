-- users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT 'User system ID',
    email VARCHAR(255) NOT NULL UNIQUE COMMENT 'Email address',
    username VARCHAR(255) NOT NULL UNIQUE COMMENT 'Username',
    tenant_id INT NOT NULL COMMENT 'Tenant ID',
    employee_id INT NOT NULL COMMENT 'Employee ID',
    UNIQUE KEY uk_tenant_employee (tenant_id, employee_id) -- Composite unique key
) COMMENT='User information';

-- orders table (for reference from order_items)
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT 'Order ID',
    user_id INT NOT NULL COMMENT 'User ID (FK)',
    order_date DATETIME COMMENT 'Order date',
    INDEX(id), -- Index needed for referencing in composite foreign key
    FOREIGN KEY fk_user (user_id) REFERENCES users(id)
) COMMENT='Order header';

-- products table
CREATE TABLE products (
    product_code VARCHAR(50) PRIMARY KEY COMMENT 'Product code (Primary Key)',
    maker_code VARCHAR(50) NOT NULL COMMENT 'Maker code',
    internal_code VARCHAR(50) NOT NULL COMMENT 'Internal product code',
    product_name VARCHAR(255) COMMENT 'Product name',
    UNIQUE KEY uk_maker_internal (maker_code, internal_code), -- Composite unique key
    INDEX idx_product_name (product_name),
    INDEX idx_maker_product_name (maker_code, product_name)
) COMMENT='Product master';

-- order_items table
CREATE TABLE order_items (
    order_id INT NOT NULL COMMENT 'Order ID (FK)',
    item_seq INT NOT NULL COMMENT 'Order item sequence number',
    product_maker VARCHAR(50) NOT NULL COMMENT 'Product maker code (FK)',
    product_internal_code VARCHAR(50) NOT NULL COMMENT 'Product internal code (FK)',
    quantity INT NOT NULL COMMENT 'Quantity',
    PRIMARY KEY (order_id, item_seq), -- Composite primary key
    UNIQUE KEY uk_order_product (order_id, product_maker, product_internal_code), -- Composite unique key (prevent duplicate products in one order)
    FOREIGN KEY fk_order (order_id) REFERENCES orders(id) ON DELETE CASCADE, -- Single foreign key (delete items if order is deleted)
    FOREIGN KEY fk_product (product_maker, product_internal_code) REFERENCES products(maker_code, internal_code) -- Composite foreign key
) COMMENT='Order details';
