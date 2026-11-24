# To-do API

A small Go/Gin API backed by PostgreSQL. It creates the `todos` table on startup; no Docker setup is included.

## Run

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
