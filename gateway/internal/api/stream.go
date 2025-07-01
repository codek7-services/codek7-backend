package api

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lumbrjx/codek7/gateway/internal/infra"
	"github.com/minio/minio-go/v7"
)

func (a *API) StreamFromMinIO(w http.ResponseWriter, r *http.Request) {
	objectKey := chi.URLParam(r, "*")

	log.Println("Requested object key:", objectKey)
	var contentType string
	switch {
	case strings.HasSuffix(objectKey, ".m3u8"):
		contentType = "application/vnd.apple.mpegurl"
	case strings.HasSuffix(objectKey, ".ts"):
		contentType = "video/MP2T"
	default:
		contentType = "application/octet-stream"
	}

	client := infra.GetMinio()
	obj, err := client.GetObject(r.Context(), "videos", objectKey, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("MinIO GetObject error: %v", err)
		http.Error(w, "Object fetch failed", http.StatusInternalServerError)
		return
	}

	_, err = obj.Stat()
	if err != nil {
		log.Printf("MinIO Stat error: %v", err)
		http.Error(w, "Object not found", http.StatusNotFound)
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, obj); err != nil {
		http.Error(w, "Failed to stream object", http.StatusInternalServerError)
	}
}
