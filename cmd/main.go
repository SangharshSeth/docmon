package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/image"
)

func main() {

	r := gin.Default()
	//opt, _ := redis.ParseURL("redis_url")
	//redisClient := redis.NewClient(opt)

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080"}, // Add your React dev server URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gzip.Gzip(gzip.BestCompression))
	dockerManager, err := docker.NewDockerManager()
	imageServiceImplManager := image.NewImageServiceImpl(dockerManager)

	if err != nil {
		log.Fatal(err.Error())
	}

	// API routes first
	api := r.Group("/api")
	{
		api.GET("/images", func(c *gin.Context) {
			//Check in cache
			//redisClient.Set(context.Background(), "request", c.Request.URL.String(), 0)

			ctx := context.Background()
			images, err := imageServiceImplManager.ListImages(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, images)
		})
	}

	// Run the server on port 8082
	if err := r.Run(":8082"); err != nil {
		log.Fatal(err)
	}
}
