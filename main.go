package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"github.com/gin-gonic/gin"
	"url-shortener/config"
	_ "github.com/lib/pq"
)

func initDB() {
	config.LoadEnvOrFail()

	var err error
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
}


func generateShortURL() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	short := make([]byte, 8)
	for i:= range short {
		short[i] = letters[r.Intn(len(letters))]
	}
	return string(short)
}


func main() {
	initDB()
	defer DB.Close()

	go startGRPCServer()

	r := gin.Default()

	r.POST("/shorten", func(c *gin.Context) {
		var req struct {
			URL string `json:"url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var existingCode string
		err := DB.QueryRow("SELECT short_code FROM urls WHERE original_url = $1", req.URL).Scan(&existingCode)

		if err == nil {
			c.JSON(http.StatusOK, gin.H{"short_url": "http://localhost:8080/" + existingCode})
			return
		}

		var short string
		for {
			short = generateShortURL()
			err = DB.QueryRow("SELECT short_code FROM urls WHERE short_code = $1", short).Scan(&existingCode)
			if err == sql.ErrNoRows {
				break // is unique, so break
			}
		}

		_, err = DB.Exec("INSERT INTO urls (original_url, short_code) VALUES ($1, $2)", req.URL, short)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save URL"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_url": "http://localhost:8080/" + short,
		})
	})

	r.GET("/:short", func(c *gin.Context) {
		short := c.Param("short")
		var original string
		
		err := DB.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", short).Scan(&original)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found!"})
			return
		}

		c.Redirect(http.StatusFound, original)
	})

	r.Run(":" + os.Getenv("HTTP_SERVER_PORT"))
}
