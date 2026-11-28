# Deploying an Application Using Brimble MCP in VS Code

This repository contains the Todo API used in the DevRel tutorial **Deploying an Application on Brimble Using MCP in VS Code**. The tutorial uses the Brimble MCP server from VS Code to deploy this Go application.

The API is built with Go and Gin and uses PostgreSQL.

## Configure Brimble MCP in VS Code

Add the following server configuration to your MCP configuration.

```json
{
  "mcpServers": {
    "brimble": {
      "tools": [
        "*"
      ],
      "type": "http",
      "url": "https://mcp.brimble.io",
      "headers": {
        "Authorization": "Bearer ${BRIMBLE_API_KEY}"
      }
    }
  }
}
```

## Run the Todo API locally

Set `DATABASE_URL` to a PostgreSQL connection string and optionally set `PORT` (defaults to `8080`), then run:

```powershell
$env:DATABASE_URL = "postgres://user:password@localhost:5432/todos?sslmode=disable"
go run .
```

The database user needs permission to create the `todos` table if it does not already exist.

## Endpoints

- `GET /health` — checks the database connection.
- `GET /todos` — returns all todos.
- `POST /todos` — creates a todo with `{"title":"Buy milk"}`.
- `GET /todos/:id` — returns one todo.
- `PATCH /todos/:id` — updates `title` and/or `completed`.
- `DELETE /todos/:id` — deletes a todo.
