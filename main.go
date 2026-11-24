package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createTodoRequest struct {
	Title string `json:"title" binding:"required"`
}

type updateTodoRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := createSchema(ctx, db); err != nil {
		log.Fatalf("initialize database schema: %v", err)
	}

	router := newRouter(db)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

func createSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todos (
			id BIGSERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func newRouter(db *sql.DB) *gin.Engine {
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/todos", listTodos(db))
	router.POST("/todos", createTodo(db))
	router.GET("/todos/:id", getTodo(db))
	router.PATCH("/todos/:id", updateTodo(db))
	router.DELETE("/todos/:id", deleteTodo(db))
	return router
}

func listTodos(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT id, title, completed, created_at, updated_at
			FROM todos ORDER BY id
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list todos"})
			return
		}
		defer rows.Close()

		todos := make([]todo, 0)
		for rows.Next() {
			var item todo
			if err := rows.Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read todos"})
				return
			}
			todos = append(todos, item)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read todos"})
			return
		}
		c.JSON(http.StatusOK, todos)
	}
}

func createTodo(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request createTodoRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}

		var item todo
		err := db.QueryRowContext(c.Request.Context(), `
			INSERT INTO todos (title) VALUES ($1)
			RETURNING id, title, completed, created_at, updated_at
		`, request.Title).Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create todo"})
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

func getTodo(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseTodoID(c)
		if !ok {
			return
		}
		item, err := findTodo(c.Request.Context(), db, id)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get todo"})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func updateTodo(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseTodoID(c)
		if !ok {
			return
		}
		var request updateTodoRequest
		if err := c.ShouldBindJSON(&request); err != nil || (request.Title == nil && request.Completed == nil) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provide a title or completed value"})
			return
		}
		if request.Title != nil && *request.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
			return
		}

		var item todo
		err := db.QueryRowContext(c.Request.Context(), `
			UPDATE todos
			SET title = COALESCE($2, title),
				completed = COALESCE($3, completed),
				updated_at = NOW()
			WHERE id = $1
			RETURNING id, title, completed, created_at, updated_at
		`, id, request.Title, request.Completed).Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update todo"})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func deleteTodo(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseTodoID(c)
		if !ok {
			return
		}
		result, err := db.ExecContext(c.Request.Context(), `DELETE FROM todos WHERE id = $1`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete todo"})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete todo"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func findTodo(ctx context.Context, db *sql.DB, id int64) (todo, error) {
	var item todo
	err := db.QueryRowContext(ctx, `
		SELECT id, title, completed, created_at, updated_at
		FROM todos WHERE id = $1
	`, id).Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func parseTodoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return 0, false
	}
	return id, true
}
