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

	// Define a map to store Firebase configuration
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
		"universe_domain":             "googleapis.com",
	}

	// Marshal the Firebase configuration map to JSON
	firebaseConfigJSON, err := json.Marshal(firebaseConfig)
	if err != nil {
		log.Fatalf("Failed to marshal Firebase configuration to JSON: %v", err)
	}

	config := &firebase.Config{
		StorageBucket: bucketName,
	}

	opt := option.WithCredentialsJSON(firebaseConfigJSON)

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, config, opt) // Initialize Firebase App
	if err != nil {
		log.Fatalln(err)
	}

	client, err = app.Storage(ctx) // Initialize Firebase Storage client
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
	// Endpoint untuk upload file dengan service account
	r.POST("/upload-signed", func(c *gin.Context) {
		// Pastikan Anda memiliki autentikasi menggunakan serviceAccount.json
		// Sebelum melakukan upload, Anda perlu memeriksa kredensial.

		// Dalam endpoint ini, Anda dapat menerima file menggunakan multipart/form-data.
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Baca file yang diupload
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		defer src.Close()

		// Anda dapat menggunakan utils.DecodeBase64WithFormat jika diperlukan
		// atau langsung mengambil data biner dari src.

		// Lakukan proses upload menggunakan service account di sini.
		// uploader.UploadFile(decodedData, object)

		// Setelah upload selesai, kirimkan respons yang sesuai.
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"file":    file.Filename,
		})
	})

	// Endpoint untuk upload file tanpa autentikasi
	r.POST("/upload-unsigned", func(c *gin.Context) {
		// Dalam endpoint ini, Anda dapat menerima file menggunakan multipart/form-data.
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Baca file yang diupload
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		defer src.Close()

		// Anda dapat menggunakan utils.DecodeBase64WithFormat jika diperlukan
		// atau langsung mengambil data biner dari src.

		// Lakukan proses upload tanpa autentikasi di sini.

		// Setelah upload selesai, kirimkan respons yang sesuai.
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"file":    file.Filename,
		})
	})

	// Endpoint untuk mengunduh file dengan service account
	r.GET("/download-signed/:filename", func(c *gin.Context) {
		// Pastikan Anda memiliki autentikasi menggunakan serviceAccount.json
		// Sebelum mengunduh file, Anda perlu memeriksa kredensial.

		// Dapatkan nama file dari parameter URL
		filename := c.Param("filename")

		// Lakukan proses pengambilan file menggunakan service account di sini.
		// Misalnya, membaca file dari penyimpanan GCS.

		// Set header untuk mengindikasikan tipe konten dan nama file yang akan diunduh
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "application/octet-stream")

		// Baca file dan kirimkannya sebagai respons
		// Gantilah ini dengan kode yang sesuai dengan penyimpanan Anda.
		fileData := []byte{} // Isi dengan data file yang ingin Anda kirimkan
		c.Data(http.StatusOK, "application/octet-stream", fileData)
	})

	// Endpoint untuk mengunduh file tanpa autentikasi
	r.GET("/download-unsigned/:filename", func(c *gin.Context) {
		// Dapatkan nama file dari parameter URL
		filename := c.Param("filename")

		// Lakukan proses pengambilan file tanpa autentikasi di sini.
		// Misalnya, membaca file dari penyimpanan GCS.

		// Set header untuk mengindikasikan tipe konten dan nama file yang akan diunduh
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "application/octet-stream")

		// Baca file dan kirimkannya sebagai respons
		// Gantilah ini dengan kode yang sesuai dengan penyimpanan Anda.
		fileData := []byte{} // Isi dengan data file yang ingin Anda kirimkan
		c.Data(http.StatusOK, "application/octet-stream", fileData)
	})

	r.GET("/download-url/:filename", func(c *gin.Context) {
		// Get the filename from the URL parameter
		filename := c.Param("filename")

		// Call the method to generate a signed URL for downloading the file
		url, err := GenerateDownloadURL(filename, client) // Pass the initialized client
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the signed URL as a response
		c.JSON(http.StatusOK, gin.H{"url": url})
	})

	r.Run()
}

// UploadFile uploads an object
// func (c *ClientUploader) UploadFile(decodedData []byte, object string) error {
// 	ctx := context.Background()

// 	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
// 	defer cancel()

// 	// Upload an object with storage.Writer.
// 	wc := c.cl.Bucket(c.bucketName).Object(object).NewWriter(ctx)
// 	if _, err := wc.Write(decodedData); err != nil {
// 		return fmt.Errorf("write: %v", err)
// 	}
// 	if err := wc.Close(); err != nil {
// 		return fmt.Errorf("Writer.Close: %v", err)
// 	}

// 	return nil
// }

func GenerateDownloadURL(filename string, client *storage.Client) (string, error) {
	// Set the expiration time to 5 minutes after the current time
	expirationTime := time.Now().Add(5 * time.Minute)

	bucketName := os.Getenv("BUCKET_NAME")

	url, err := cloudStorage.SignedURL(bucketName, filename, &cloudStorage.SignedURLOptions{
		GoogleAccessID: os.Getenv("FIREBASE_CLIENT_EMAIL"),
		PrivateKey:     []byte(strings.Replace(string(os.Getenv("FIREBASE_PRIVATE_KEY")), "\\n", "\n", -1)),
		Method:         "GET",
		Expires:        expirationTime,
	})

	return url, err
}
