package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"math/rand"
	"time"
)

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
	r := gin.Default()

	r.POST("/shorten", func(c *gin.Context) {
		var req struct {
			URL string `json:"url"`
		}

		if err := c.BindJSON(&req); err != nil || req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL"})
			return
		}

		short := generateShortURL()
		urlStore[short] = req.URL

		c.JSON(http.StatusOK, gin.H{
			"short_url": "http://localhost:8080/" + short,
		})
	})

	r.GET("/:short", func(c *gin.Context) {
		short := c.Param("short")
		original, ok := urlStore[short]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found!"})
			return
		}
		c.Redirect(http.StatusFound, original)
	})

	r.Run(":8080")
}
