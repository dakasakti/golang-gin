package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type (
	any      interface{}
	Response struct {
		Status int `json:"status"`
		Data   any `json:"data,omitempty"`
	}
)

func setupRouter() *gin.Engine {
	db, err := sql.Open("mysql", "root:@/dakasakti")
	if err != nil {
		log.Fatal(err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	defer func() {
		err := db.Close()
		if err != nil {
			log.Println("failed to close db:", err)
		}
	}()

	e := gin.Default()
	e.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, Response{
			Status: 1,
			Data:   "Mahmuda Karima",
		})
	})

	e.POST("students", func(ctx *gin.Context) {
		postHandler(ctx, db)
	})

	e.GET("students", func(ctx *gin.Context) {
		getsHandler(ctx, db)
	})

	e.GET("students/:id", func(ctx *gin.Context) {
		getHandler(ctx, db)
	})

	e.PATCH("students/:id", func(ctx *gin.Context) {
		updateHandler(ctx, db)
	})

	e.DELETE("students/:id", func(ctx *gin.Context) {
		deleteHandler(ctx, db)
	})

	return e
}

type Student struct {
	ID      uint   `json:"id"`
	NISN    string `json:"nisn"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type StudentCreate struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address,omitempty" binding:"required"`
}

type StudentUpdate struct {
	Name    string `json:"name" binding:"omitempty"`
	Address string `json:"address,omitempty" binding:"omitempty"`
}

func getsHandler(ctx *gin.Context, db *sql.DB) {
	var res []Student

	rows, err := db.Query("SELECT * FROM students")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			log.Println("failed to close rows:", err)
		}
	}()

	for rows.Next() {
		var req Student
		err := rows.Scan(&req.ID, &req.NISN, &req.Name, &req.Address)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})

			return
		}

		res = append(res, req)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success get students",
		"status":  1,
		"data":    res,
	})
}

func getHandler(ctx *gin.Context, db *sql.DB) {
	var res Student

	rows := db.QueryRow("SELECT * FROM students WHERE id = ?", ctx.Param("id"))
	err := rows.Scan(&res.ID, &res.NISN, &res.Name, &res.Address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "student not found",
			})

			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success get student",
		"status":  1,
		"data":    res,
	})
}

func generateNumber() string {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	rand.Seed(time.Now().UnixNano())

	return fmt.Sprintf("%02d%02d%02d%04d", year%100, month, day, rand.Intn(10000))
}

func postHandler(ctx *gin.Context, db *sql.DB) {
	var req StudentCreate

	err := ctx.Bind(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	res, err := db.Exec("INSERT INTO students (nisn, name, address) VALUES (?, ?, ?)", generateNumber(), req.Name, req.Address)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	code, err := res.RowsAffected()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "success create student",
		"status":  code,
	})

}

func updateHandler(ctx *gin.Context, db *sql.DB) {
	var req StudentUpdate

	err := ctx.Bind(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	if (req == StudentUpdate{}) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "request body is empty",
		})

		return
	}

	// optional update
	var query string
	args := []interface{}{}

	query = "UPDATE students SET"
	if req.Name != "" {
		query += " name=?,"
		args = append(args, req.Name)
	}

	if req.Address != "" {
		query += " address=?,"
		args = append(args, req.Address)
	}

	query = strings.TrimSuffix(query, ",")
	query += " WHERE id=?"
	args = append(args, ctx.Param("id"))

	stmt, err := db.Prepare(query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	defer func() {
		err := stmt.Close()
		if err != nil {
			log.Println("failed to close statement:", err)
		}
	}()

	res, err := stmt.Exec(args...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	code, err := res.RowsAffected()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success update student",
		"status":  code,
	})
}

func deleteHandler(ctx *gin.Context, db *sql.DB) {
	res, err := db.Exec("DELETE FROM students WHERE id = ?", ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	code, err := res.RowsAffected()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success delete student",
		"status":  code,
	})
}

func main() {
	e := setupRouter()
	e.Run()
}
