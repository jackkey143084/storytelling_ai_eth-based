
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// load env vars if provided
	_ = godotenvLoad()

	r := gin.Default()
	r.POST("/mint", MintHandler)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	port := os.Getenv("PORT")
	if port == "" { port = "8081" }
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
