package types

type UploadRequest struct {
	FileName   string `json:"file_name"`
	Base64Data string `json:"base64_data"`
}

type UrlFile struct {
	Url string `json:"url"`
}
