package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type portfolioData struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	Message string `json:"message" binding:"required"`
}

var collection *mongo.Collection

func main() {

	// mongoURI := "mongodb+srv://juicekuditiya4_db_user:t6eK8HLqoqwo6oER@cluster0.ucu1skg.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	
	// Read Mongo URI from environment variable
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable not set")
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}

	// Ping MongoDB to ensure connection works
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}
	log.Println("✅ MongoDB connected successfully")

	collection = client.Database("portfolio").Collection("messages")

	// Setup Gin router
	router := gin.Default()

	config := cors.Config{
		AllowOrigins:     []string{"*"},
		// AllowOrigins:     []string{"https://portfolio.mahadev.gt.tc"},
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	// Routes
	router.POST("/submit", handleSubmit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4100" // default fallback
	}
	log.Println("🚀 Server running on port:", port)
	router.Run(":" + port)
}

func handleSubmit(c *gin.Context) {
	var input portfolioData
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	doc := bson.M{
		"name":      input.Name,
		"email":     input.Email,
		"message":   input.Message,
		"createdAt": time.Now(),
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Mongo insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Message saved successfully"})
}

