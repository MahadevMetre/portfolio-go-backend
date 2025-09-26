package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gomail.v2"
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
	router.SetTrustedProxies(nil) // trust proxy on Railway - mahadev

	config := cors.Config{
		AllowOrigins: []string{"*", "https://*.github.io"},
		// AllowOrigins:     []string{"https://portfolio.mahadev.gt.tc"},
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	// mahadev
	// Optional: handle preflight OPTIONS requests
	router.OPTIONS("/submit", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Routes
	router.POST("/submit", handleSubmit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4100" // default fallback
	}
	log.Printf("🚀 Server running on port:%s", port)
	router.Run(":" + port)
}

func handleSubmit(c *gin.Context) {
	// <-- ADD THIS LOGGING FIRST
	log.Println("Received request from:", c.Request.RemoteAddr)

	var input portfolioData
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println("BindJSON error:", err) // optional extra logging
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Parsed input: %+v\n", input) // logs what the backend actually received

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

	go sendEmailNotification(input)

	c.JSON(http.StatusOK, gin.H{"status": "Message saved successfully"})
}

// new implementation on sep 26

const emailHTML = `
<html>
  <body style="font-family: Arial, sans-serif; background:#fafbfc; margin:0; padding:30px;">
    <div style="max-width:480px; margin:auto; background:#fff; border-radius:8px; box-shadow:0 3px 12px #eee; padding:32px 24px;">
      <h2 style="color:#2552d0; margin-top:0;">New Contact Form Submission</h2>
      <table style="width:100%%; margin:16px 0 24px 0; border-collapse:collapse;">
        <tr>
          <td style="font-weight:600; padding:8px 0; width:100px;">Name:</td>
          <td style="padding:8px 0;">%s</td>
        </tr>
        <tr>
          <td style="font-weight:600; padding:8px 0;">Email:</td>
          <td style="padding:8px 0;">%s</td>
        </tr>
        <tr>
          <td style="font-weight:600; padding:8px 0; vertical-align:top;">Message:</td>
          <td style="padding:8px 0; white-space:pre-wrap;">%s</td>
        </tr>
      </table>
      <div style="color:#9da3ae; font-size:13px; border-top:1px solid #ededed; padding-top:24px; margin-top:24px;">
        <em>This message was sent from your website contact form.</em>
      </div>
    </div>
  </body>
</html>
`

func sendEmailNotification(inq portfolioData) {

	host := os.Getenv("SMTP_HOST")
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")

	m := gomail.NewMessage()
	m.SetHeader("From", user)
	m.SetHeader("To", user)
	m.SetHeader("Subject", "New Contact Form Submission")

	// HTML escape the message content to prevent formatting issues
	escapedMessage := html.EscapeString(inq.Message)
	fmt.Println(escapedMessage)

	// In your function, change the % to %% in the CSS properties
	body := fmt.Sprintf(emailHTML, inq.Name, inq.Email, html.EscapeString(inq.Message))
	m.SetBody("text/html", body)

	d := gomail.NewDialer(host, port, user, pass)

	if err := d.DialAndSend(m); err != nil {
		fmt.Println("Failed to send email:", err)
	}
}
