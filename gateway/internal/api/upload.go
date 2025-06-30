package api

import (
	"fmt"
	"io"
	"net/http"
)

func (a API) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form without a size limit
	err := r.ParseMultipartForm(0) // 0 means no explicit limit
	if err != nil {
		http.Error(w, `{"status":"error","message":"Failed to parse form"}`, http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"status":"error","message":"File not found in request"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Printf("Receiving file: %s\n", handler.Filename)

	// Use a reasonably large buffer (e.g., 32KB) — Go may optimize this internally
	buf := make([]byte, 32*1024)

	totalBytes := 0
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			http.Error(w, `{"status":"error","message":"Error reading file"}`, http.StatusInternalServerError)
			return
		}
		if n == 0 {
			break
		}

		processChunk(buf[:n])
		totalBytes += n
	}

	fmt.Printf("File fully received (%d bytes)\n", totalBytes)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"File uploaded and processed"}`))
}

func processChunk(chunk []byte) {
	fmt.Printf("Chunk of size: %d\n", len(chunk))
}

