package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	cloudStorage "cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var client *storage.Client

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	// Gunakan variabel lingkungan yang telah di-set
	bucketName := os.Getenv("BUCKET_NAME")

	// Define map to firebase config
	firebaseConfig := map[string]string{
		"type":                        os.Getenv("FIREBASE_TYPE"),
		"project_id":                  os.Getenv("FIREBASE_PROJECT_ID"),
		"private_key_id":              os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		"private_key":                 strings.Replace(string(os.Getenv("FIREBASE_PRIVATE_KEY")), "\\n", "\n", -1),
		"client_email":                os.Getenv("FIREBASE_CLIENT_EMAIL"),
		"client_id":                   os.Getenv("FIREBASE_CLIENT_ID"),
		"auth_uri":                    os.Getenv("FIREBASE_AUTH_URL"),
		"token_uri":                   os.Getenv("FIREBASE_TOKEN_URL"),
		"auth_provider_x509_cert_url": os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
		"client_x509_cert_url":        os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
		"universe_domain":             os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
	}

	// Marshal firebase ke JSON
	firebaseConfigJSON, err := json.Marshal(firebaseConfig)
	if err != nil {
		log.Fatalf("Failed to marshal Firebase configuration to JSON: %v", err)
	}

	config := &firebase.Config{
		StorageBucket: bucketName,
	}

	// Inject config
	opt := option.WithCredentialsJSON(firebaseConfigJSON)

	ctx := context.Background()

	// Init Firebase App
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatalln(err)
	}

	// Init Firebase Storage client
	client, err := app.Storage(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = client.DefaultBucket()
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	r := gin.Default()

	// Endpoint untuk get raw url and signed url
	r.GET("/url/:filename", func(c *gin.Context) {
		// Dapatkan nama file dari parameter URL
		filename := c.Param("filename")

		signedURL, rawURL, err := GenerateURL(filename, 30, client) // 30 second ttl biar bisa liat2 dulu
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the signed and raw URLs as a response
		c.JSON(http.StatusOK, gin.H{
			"signed_url": signedURL,
			"raw_url":    rawURL,
		})
	})

	// Endpoint untuk download signed url
	r.GET("/download-signed/:filename", func(c *gin.Context) {
		// Get the filename from the URL parameter
		filename := c.Param("filename")

		// Generate the signed URL for downloading the file
		url, _, err := GenerateURL(filename, 5, client) // 5 second ttl karena langsung download
		if err != nil {
			if strings.Contains(err.Error(), "token expired") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else if strings.Contains(err.Error(), "no keys") {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "No keys available"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		// Set appropriate headers for the file download
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "application/octet-stream")

		// Trigger the file download by redirecting the client to the signed URL
		c.Redirect(http.StatusFound, url)
	})

	// Endpoint untuk download unsigned url
	r.GET("/download-unsigned/:filename", func(c *gin.Context) {
		// Get the filename from the URL parameter
		filename := c.Param("filename")

		// Generate the signed URL for downloading the file
		_, url, err := GenerateURL(filename, 5, client) // 5 second ttl karena langsung download
		if err != nil {
			if strings.Contains(err.Error(), "token expired") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else if strings.Contains(err.Error(), "no keys") {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "No keys available"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		// Set appropriate headers for the file download
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "application/octet-stream")

		// Trigger the file download by redirecting the client to the signed URL
		c.Redirect(http.StatusFound, url)
	})

	r.Run()
}

// Generate Signed URL menggunakan metode yang sama seperti di ethica-be
func GenerateURL(filename string, ttlSecond int, client *storage.Client) (string, string, error) {
	// Konversi ttlSecond ke time.Duration
	expirationDuration := time.Duration(ttlSecond) * time.Second

	// Set the expiration time to just a few seconds in the future
	expirationTime := time.Now().Add(expirationDuration)

	bucketName := os.Getenv("BUCKET_NAME")

	signedUrl, err := cloudStorage.SignedURL(bucketName, filename, &cloudStorage.SignedURLOptions{
		GoogleAccessID: os.Getenv("FIREBASE_CLIENT_EMAIL"),
		PrivateKey:     []byte(strings.Replace(string(os.Getenv("FIREBASE_PRIVATE_KEY")), "\\n", "\n", -1)),
		Method:         "GET",
		Expires:        expirationTime,
	})

	if err != nil {
		return "", "", err
	}

	rawURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, filename)

	return signedUrl, rawURL, err
}
