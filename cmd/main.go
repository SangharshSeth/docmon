package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sangharshseth/docmon/internal/container"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/image"
	"github.com/sangharshseth/docmon/internal/middleware"
	"github.com/sangharshseth/docmon/internal/types"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket connections,
	},
}

func main() {

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	err := r.SetTrustedProxies(nil)
	if err != nil {
		return
	}
	opt, _ := redis.ParseURL(os.Getenv("REDIS"))
	redisClient := redis.NewClient(opt)

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Add your React dev server URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gzip.Gzip(gzip.BestCompression))
	dockerManager, err := docker.NewDockerManager()
	imageServiceImplManager := image.NewImageServiceImpl(dockerManager, redisClient)
	containerServiceImplManager := container.NewContainerServiceImpl(dockerManager, redisClient)

	if err != nil {
		log.Fatal(err.Error())
	}

	// API routes first
	api := r.Group("/api")
	{
		api.GET("/images", func(c *gin.Context) {
			ctx := context.Background()

			// Check Redis cache
			cachedData, err := redisClient.Get(ctx, "image-list").Bytes()
			if err == nil {
				var images []types.DockerImageDetails
				if err := json.Unmarshal(cachedData, &images); err == nil {
					slog.Info("Cache hit: Returning images from Redis")
					c.JSON(http.StatusOK, images)
					return
				}
				slog.Error("Failed to unmarshal cached image data", "error", err)
			}

			slog.Info("Cache miss: Fetching images from Docker")
			images, err := imageServiceImplManager.ListImages(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, images)
		})

		api.GET("/containers-snapshot", func(c *gin.Context) {
			ctx := context.Background()
			containerSnapshots, err := containerServiceImplManager.GetAllContainers(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, containerSnapshots)
		})

		// Update the existing WebSocket endpoint in your main.go
		api.GET("/ws", func(c *gin.Context) {
			conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				slog.Error("Failed to upgrade connection", "error", err)
				return
			}
			defer conn.Close()

			// Create a done channel to handle graceful shutdown
			done := make(chan bool)
			defer close(done)

			// Handle incoming messages in a separate goroutine
			go func() {
				for {
					_, msg, err := conn.ReadMessage()
					if err != nil {
						slog.Error("Failed to read message", "error", err)
						done <- true
						return
					}
					slog.Info("Received message from client", "message", string(msg))
				}
			}()

			// Create ticker for periodic updates
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			// Send initial data immediately
			ctx := context.Background()
			initialStats, err := containerServiceImplManager.GetTotalResourceUsageByAllContainers(ctx)
			if err == nil {
				if err := conn.WriteJSON(initialStats); err != nil {
					slog.Error("Failed to send initial stats", "error", err)
					return
				}
			}

			// Main event loop
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					ctx := context.Background()
					stats, err := containerServiceImplManager.GetTotalResourceUsageByAllContainers(ctx)
					if err != nil {
						slog.Error("Failed to get resource usage", "error", err)
						continue
					}

					if err := conn.WriteJSON(stats); err != nil {
						slog.Error("Failed to send stats", "error", err)
						return
					}
				}
			}
		})
	}
	staticDir := "./static"
	r.Use(middleware.ServeStaticWithIndex("/", staticDir))
	slog.Info("open the ui dashboard at http://localhost:3000")
	// Run the server on port 8082
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
