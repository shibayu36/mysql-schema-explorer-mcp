# Model Context Protocol (MCP) Server Implementation Guide

## What is MCP?

The Model Context Protocol (MCP) is an open protocol. It connects AI language models (LLMs) with external data sources and tools in a standard way. It acts like a plugin system between LLM applications and external tools, giving seamless access to data sources.

## Basic Architecture

MCP uses a client-server model:

-   **MCP Server**: A lightweight program that provides access to data sources or tools.
-   **MCP Host/Client**: An LLM application like Claude Desktop. It connects to MCP servers to use their features.

## Main Features of an MCP Server

MCP servers can offer three main types of features:

1.  **Tools**: Functions that the LLM can call (with user approval).
2.  **Resources**: Data in a file-like format that the client can read (like API responses or file content).
3.  **Prompts**: Standard templates that help with specific tasks.

## Communication Protocols

MCP servers support these communication methods:

-   **Standard Input/Output (stdio)**: A simple method suitable for local development.
-   **Server-Sent Events (SSE)**: A more flexible method for distributed teams.
-   **WebSockets**: For real-time, two-way communication.

## What You Need to Build

### 1. Basic Structure

To implement an MCP server, you need these elements:

-   Server initialization and transport setup.
-   Definitions for tools, resources, or prompts.
-   Implementation of request handlers.
-   The main function to run the server.

### 2. Server Configuration

The server configuration includes this information:

```json
{
  "mcpServers": {
    "myserver": {
      "command": "command_to_execute",
      "args": ["arg1", "arg2"],
      "env": {
        "ENV_VAR_NAME": "value"
      }
    }
  }
}
```

## How to Implement in Different Languages

You can implement MCP servers in various programming languages:

### Python

```python
# Install necessary libraries
# pip install mcp[cli]

import asyncio
import mcp
from mcp.server import NotificationOptions, InitializationOptions

# Example tool definition
@mcp.server.tool("tool_name", "Description of the tool")
async def some_tool(param1: str, param2: int) -> str:
    # Implement the tool's logic
    return "Result"

# Initialize the server
server = mcp.server.Server()

# Main function
async def main():
    # Run the server with stdin/stdout streams
    async with mcp.server.stdio.stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream, write_stream,
            InitializationOptions(
                server_name="Server Name",
                server_version="Version",
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )

if __name__ == "__main__":
    asyncio.run(main())
```

### TypeScript/JavaScript

```typescript
// Install necessary libraries
// npm install @modelcontextprotocol/sdk

import { Server, StdioServerTransport } from "@modelcontextprotocol/sdk";

// Create a server instance
const server = new Server();

// Define a tool
const myTool = {
  name: "tool_name",
  description: "Description of the tool",
  parameters: {
    // Define parameters
  },
  execute: async (params) => {
    // Implement the tool's logic
    return { result: "Result" };
  }
};

// Register the tool
server.tools.registerTool(myTool);

// Main function
async function main() {
  // Set up transport
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("Server running");
}

main().catch((error) => {
  console.error("Error:", error);
  process.exit(1);
});
```

## Security Points to Consider

When implementing an MCP server, consider these security aspects:

-   **Access Control**: Limit access to the data and tools the server exposes.
-   **Authentication**: Mechanisms to verify the client.
-   **Data Protection**: Handle sensitive data properly.
-   **Resource Limiting**: Limit resource usage to prevent Denial of Service (DoS) attacks.

## Best Practices

1.  **Clear Documentation**: Provide detailed descriptions for each tool, resource, and prompt.
2.  **Error Handling**: Return appropriate error messages and status codes.
3.  **Versioning**: Manage API versions for compatibility.
4.  **Testing**: Perform unit tests and integration tests.
5.  **Logging**: Implement logging for debugging and auditing.

## Examples of Existing MCP Servers

-   **File System**: A secure server for file operations.
-   **PostgreSQL**: A server for database access.
-   **GitHub**: Provides features for repository and issue management.
-   **Brave Search**: Offers web search functionality.

## Connecting with Claude Desktop

To connect your MCP server with Claude Desktop:

1.  Install Claude Desktop.
2.  Edit `~/Library/Application Support/Claude/claude_desktop_config.json`.
3.  Add your custom server to the `mcpServers` section.

```json
{
  "mcpServers": {
    "myserver": {
      "command": "command_to_execute",
      "args": ["arg1", "arg2"]
    }
  }
}
```

## Debugging and Troubleshooting

1.  **Log Output**: Record detailed debug information.
2.  **Step-by-Step Testing**: Test from basic features to complex ones.
3.  **Error Codes**: Implement clear error codes and messages.
4.  **MCP Inspector**: Use debugging tools to check behavior.

## Summary

Implementing an MCP server allows you to connect various data sources and tools to LLMs. This extends the capabilities of AI assistants and can provide a richer user experience.
