package main

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type Todo struct {
	bun.BaseModel `bun:"table:todos"`

	ID          int64  `json:"id" bun:"column:pk,autoincrement"`
	Title       string `json:"title" validate:"required,min=3" bun:"title,notnull"`
	Description string `json:"description" validate:"required,min=5" bun:"description"`
}

func main() {
	config, err := pgx.ParseConfig("postgres://postgres:postgres@localhost:5432/todos?sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}

	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	_, err = db.NewCreateTable().Model((*Todo)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	server := fiber.New()
	server.Use(cors.New())

	todos := server.Group("/todos")

	// TODO: get all todos
	// http://localhost:3000/todos
	todos.Get("/:page<int>?:title?", func(c *fiber.Ctx) error {
		todos := new([]Todo)
		pageStr := c.Query("page", "0")
		title := c.Query("title", "")

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]string{"err": err.Error()})
		}

		query := db.NewSelect().Model(todos).Offset(page * 5).Limit(5)
		if title != "" {
			query = query.Where("title ILIKE ?", "%"+title+"%")
		}

		err = query.Scan(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{"err": err.Error()})
		} else if len(*todos) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(map[string]string{"err": "No todos found"})
		}

		return c.JSON(todos)
	})

	// TODO: get todo by id
	// http://localhost:3000/todos/:id
	todos.Get("/:id<int64>", func(c *fiber.Ctx) error {
		todo := new(Todo)
		id := c.Params("id")

		err := db.NewSelect().Model(todo).Where("id =?", id).Scan(c.Context())
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(map[string]string{"err": err.Error()})
		}

		return c.JSON(todo)
	})

	// TODO: create todo
	// http://localhost:3000/todos/
	todos.Post("/", func(c *fiber.Ctx) error {
		todo := new(Todo)

		if err := c.BodyParser(todo); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]string{"err": err.Error()})
		}

		_, err = db.NewInsert().Model(todo).Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{"err": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(todo)
	})

	// TODO: update todo
	// http://localhost:3000/todos/:id
	todos.Put("/:id<int64>", func(c *fiber.Ctx) error {
		todo := new(Todo)
		id := c.Params("id")

		if err := c.BodyParser(todo); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to parse request body",
			})
		}

		idInt, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]string{
				"error": "Invalid ID",
			})
		}
		todo.ID = idInt

		_, err = db.NewUpdate().Model(todo).OmitZero().Where("id = ?", id).Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{"err": err.Error()})
		}

		return c.JSON(todo)
	})

	// TODO: delete todo
	// http://localhost:3000/todos/:id
	todos.Delete("/:id<int64>", func(c *fiber.Ctx) error {
		id := c.Params("id")

		_, err = db.NewDelete().Model((*Todo)(nil)).Where("id =?", id).Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{"err": err.Error()})
		}

		return c.JSON(map[string]string{
			"message": "Todo deleted successfully",
		})
	})

	log.Fatalln(server.Listen(":3000"))
}
