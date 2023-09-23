package utils

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"strings"
	"time"
)

var signatures = map[string]string{
	"JVBERi0":     "application/pdf",
	"iVBORw0KGgo": "image/png",
	"/9j/":        "image/jpeg",
}

type Base64File struct {
	Name     string
	Contents []byte
}

func NewBase64File(name string, contents []byte) *Base64File {
	return &Base64File{
		Name:     name,
		Contents: contents,
	}
}

func detectMimeType(b64 string) (string, error) {
	for s, mimeType := range signatures {
		if strings.HasPrefix(b64, s) {
			return mimeType, nil
		}
	}
	return "", errors.New("unsupported base64 format")
}

func DecodeBase64WithFormat(base64Data string) ([]byte, string, error) {
	mimeType, err := detectMimeType(base64Data)
	if err != nil {
		return nil, "", err
	}

	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", err
	}

	var ext string
	switch mimeType {
	case "application/pdf":
		ext = ".pdf"
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	default:
		return nil, "", errors.New("unsupported mimeType: " + mimeType)
	}

	return decodedData, ext, nil
}

func GenerateRandomName() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 128)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
