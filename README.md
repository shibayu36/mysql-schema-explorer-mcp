# MySQL Schema MCP Server

This is a Model Context Protocol (MCP) server that provides compressed schema information for MySQL databases.
This MCP server is useful when the schema dump file does not fit in the context size because you are using a large database schema.

https://github.com/user-attachments/assets/f81b2513-31bd-4a60-9b54-45f76323d112

## Provided Tools

- List Tables (`list_tables`)
  - Lists all table information in the specified database. Includes table name, comment, primary key, unique key, and foreign key information.
  - Parameters
    - `dbName`: The name of the database to retrieve information from.
- Describe Tables (`describe_tables`)
  - Displays detailed information for specific tables in the specified database. Provides formatted information such as column definitions, key constraints, and indexes.
  - Parameters
    - `dbName`: The name of the database to retrieve information from.
    - `tableNames`: An array of table names to retrieve detailed information for.

## Quick Start
1. Install the command

    ```
    go install github.com/shibayu36/mysql-schema-explorer-mcp@latest
    ```

2. Configure mcp.json

    ```json
    {
      "mcpServers": {
        "mysql-schema-explorer-mcp": {
          "command": "/path/to/mysql-schema-explorer-mcp",
          "env": {
            "DB_HOST": "127.0.0.1",
            "DB_PORT": "3306",
            "DB_USER": "root",
            "DB_PASSWORD": "root"
          },
        }
      }
    }
    ```

3. Execute SQL generation using the agent

    Example: Using the structure of the ecshop database, list the names of the 3 most recently ordered products by the user shibayu36.
