# MySQL Schema MCP Server Requirements

## Overview
This project implements a server using the Model Context Protocol (MCP) to provide MySQL database schema information. LLM clients like Claude can connect to this server. The server offers tools to get information about the database schema.

## Environment Variables
- **DB_HOST**: Database host name
- **DB_PORT**: Database port number
- **DB_USER**: Database username
- **DB_PASSWORD**: Database password

## MCP Server Tools

1.  **Tool: `list_tables`**
    -   Description: Returns a list of all table names in the specified database.
    -   Arguments: `dbName` (string) - The name of the database to get information from.
    -   Return Value: A list of table names, table comments, and key information (in text format).
    -   Output Format:
        ```text
        Tables in database "DB_NAME" (Total: X)
        Format: Table Name - Table Comment [PK: Primary Key] [UK: Unique Key 1; Unique Key 2...] [FK: Foreign Key -> Referenced Table.Column; ...]
        * Composite keys (keys made of multiple columns) are grouped in parentheses: (col1, col2)
        * Multiple different key constraints are separated by semicolons: key1; key2

        - users - User information [PK: id] [UK: email; username] [FK: role_id -> roles.id; department_id -> departments.id]
        - posts - Post information [PK: id] [UK: slug] [FK: user_id -> users.id; category_id -> categories.id]
        - order_items - Order items [PK: (order_id, item_id)] [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id]
        ```

2.  **Tool: `describe_tables`**
    -   Description: Returns detailed information for the specified tables, such as column info, indexes, and foreign key constraints.
    -   Arguments:
        -   `dbName` (string) - The name of the database to get information from.
        -   `tableNames` (array of strings) - The names of the tables to get detailed information for (you can specify multiple names).
    -   Return Value: Formatted text with detailed information for each table.
    -   Output Format:
        ```text
        # Table: order_items - Order Items

        ## Columns
        - order_id: int(11) NOT NULL [Order ID]
        - item_id: int(11) NOT NULL [Item ID]
        - product_id: int(11) NOT NULL [Product ID]
        - quantity: int(11) NOT NULL [Quantity]
        - price: decimal(10,2) NOT NULL [Price]
        - user_id: int(11) NOT NULL [User ID]

        ## Key Information
        [PK: (order_id, item_id)]
        [UK: (user_id, product_id)]
        [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id; user_id -> users.id]
        [INDEX: price; quantity]

        ---

        # Table: users - User Information

        ## Columns
        - id: int(11) NOT NULL [User ID]
        - username: varchar(50) NOT NULL [Username]
        - email: varchar(100) NOT NULL [Email Address]
        - password: varchar(255) NOT NULL [Password]
        - created_at: timestamp NULL DEFAULT CURRENT_TIMESTAMP [Created At]

        ## Key Information
        [PK: id]
        [UK: email; username]
        [INDEX: created_at]
        ```

        If you specify multiple tables, a separator line (`---`) will be inserted between each table's information.

## Implementation Steps

1.  **Project Setup**
    -   Install the MCP library.
    -   Install necessary dependencies (like MySQL client library).

2.  **MCP Server Initialization**
    -   Create the server instance and set its name.

3.  **Load Environment Variables**
    -   Read environment variables when the server starts to set up database connection info.

4.  **Database Connection Helper**
    -   Implement a helper function to manage database connections.
    -   The database name will be received as an argument in tool calls.

5.  **Implement Tools**
    -   Implement each tool function.
    -   Run appropriate database queries within the tools and format the results.

6.  **Run the Server**
    -   Set up the server to communicate with the client using standard input/output (stdio).

## Progress

-   [x] Project Setup
-   [x] MCP Server Initialization
-   [x] Load Environment Variables
-   [x] Database Connection Helper Implementation
-   [x] Implement `list_tables` tool
-   [x] Implement `describe_tables` tool
-   [x] Receive DB_NAME as a tool call argument, not an environment variable
-   [ ] Adjust security settings
