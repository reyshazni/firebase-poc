package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	cloudStorage "cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var client *storage.Client
var firestoreClient *firestore.Client

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
	client, err = app.Storage(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	firestoreClient, err = app.Firestore(ctx)
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

	r.POST("/data-firestore-sdk/:data", func(c *gin.Context) {
		data := c.Param("data")

		// Call the addDocWithoutID function to add data to Firestore
		err := addDocWithoutID(c, firestoreClient, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add data to Firestore"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Data added to Firestore with timestamp"})
	})

	r.GET("/data-firestore-sdk", func(c *gin.Context) {
		// Call the allDocs function to retrieve all documents from Firestore
		data, err := allDocs(c, firestoreClient)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data from Firestore"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	r.GET("/data-firestore-url-unsigned", func(c *gin.Context) {
		// Buat URL Firestore API yang sesuai dengan dokumen yang ingin Anda ambil
		firestoreURL := "https://firestore.googleapis.com/v1/projects/test-pharindo/databases/(default)/documents/tes"

		// Buat permintaan HTTP GET ke URL Firestore API
		response, err := http.Get(firestoreURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data from Firestore"})
			return
		}
		defer response.Body.Close()

		// Baca data dari respons Firestore API
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read data from Firestore response"})
			return
		}

		// Gunakan fungsi sanitizeData untuk mendapatkan list of object fields
		sanitizedData, err := sanitizeData(string(data))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sanitize data"})
			return
		}

		// Return data sebagai respons JSON
		c.JSON(http.StatusOK, gin.H{"data": sanitizedData})
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

func addDocWithoutID(ctx context.Context, client *firestore.Client, data string) error {
	_, _, err := client.Collection("tes").Add(ctx, map[string]interface{}{
		"timestamp": time.Now(),
		"queryData": data,
	})
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
	}

	return err
}

func allDocs(ctx context.Context, client *firestore.Client) ([]map[string]interface{}, error) {
	var data []map[string]interface{}

	iter := client.Collection("tes").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		data = append(data, doc.Data())
	}

	return data, nil
}

func sanitizeData(firestoreData string) ([]map[string]interface{}, error) {
	// Struktur data untuk mengurai respons Firestore
	var firestoreResponse map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(firestoreData), &firestoreResponse); err != nil {
		return nil, err
	}

	// Mengurai data Firestore dan mendapatkan daftar fields dari setiap array
	fieldLists := []map[string]interface{}{}
	documents := firestoreResponse["documents"]
	for _, doc := range documents {
		if fields, ok := doc["fields"].(map[string]interface{}); ok {
			fieldLists = append(fieldLists, fields)
		}
	}

	return fieldLists, nil
}
