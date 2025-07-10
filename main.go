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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error while loading .env file")
	}

	var err error
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
}

var urlStore = make(map[string]string)

func generateShortURL() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		short := make([]byte, 8)
		for i:= range short {
			short[i] = letters[r.Intn(len(letters))]
		}
		code := string(short)

		if _, exists := urlStore[code]; !exists {
			return code
		}
	}
}


func main() {
	initDB()
	defer db.Close()

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
		err := db.QueryRow("SELECT short_code FROM urls WHERE original_url = $1", req.URL).Scan(&existingCode)

		if err == nil {
			c.JSON(http.StatusOK, gin.H{"short_url": "http://localhost:8080/" + existingCode})
			return
		}

		short := generateShortURL()
		_, err = db.Exec("INSERT INTO urls (original_url, short_code) VALUES ($1, $2)", req.URL, short)
		
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
		
		err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", short).Scan(&original)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found!"})
			return
		}

		c.Redirect(http.StatusFound, original)
	})

	r.Run(":8080")
}
