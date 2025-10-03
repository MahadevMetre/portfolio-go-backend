package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
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

	mongoURI := "mongodb+srv://juicekuditiya4_db_user:naS6pVsdM6h3gQwD@cluster0.ucu1skg.mongodb.net/portfolio?retryWrites=true&w=majority"

	// Read Mongo URI from environment variable
	//mongoURI := os.Getenv("MONGO_URI")
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
		port = "4200" // default fallback
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


	apiKey := os.Getenv("SENDGRID_API_KEY")
	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")
	toEmail := os.Getenv("SENDGRID_TO_EMAIL")

	if fromEmail == "" || toEmail == "" || apiKey == "" {
		log.Println("SendGrid environment variables not set")
		return
	}

	from := mail.NewEmail("Portfolio Website", fromEmail)
	to := mail.NewEmail("Portfolio Owner", toEmail)
	subject := "New Contact Form Submission Via Portfolio Website"

	escapedMessage := html.EscapeString(inq.Message)
	body := fmt.Sprintf(emailHTML, inq.Name, inq.Email, escapedMessage)
	plainText := fmt.Sprintf("Name: %s\nEmail: %s\nMessage: %s", inq.Name, inq.Email, inq.Message)

	message := mail.NewSingleEmail(from, subject, to, plainText, body)
	client := sendgrid.NewSendClient(apiKey)

	response, err := client.Send(message)
	if err != nil {
		log.Println("SendGrid error:", err)
	} else {
		log.Println("SendGrid response code:", response.StatusCode)
	}
}
