package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func (a API) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
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

	// Read the entire file into memory
	buf := make([][]byte, 0)
	chunk := make([]byte, 32*1024)

	for {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			log.Printf("❌ Error reading file: %v\n", err)
			http.Error(w, `{"status":"error","message":"Error reading file"}`, http.StatusInternalServerError)
			return
		}
		if n == 0 {
			break
		}

		// Copy the chunk because `chunk` is reused
		cpy := make([]byte, n)
		copy(cpy, chunk[:n])
		buf = append(buf, cpy)
	}

	videoID := uuid.New().String()

	// Fire and forget processing
	go func(chunks [][]byte, videoID string) {
		for i, chunk := range chunks {
			produceChunk(a.Producer, videoID, int32(i), chunk)
		}
	}(buf, videoID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"File is being processed"}`))
}


func produceChunk(producer *kafka.Writer, videoID string, index int32, chunk []byte) {
	msg := kafka.Message{
		Key:   []byte(videoID),
		Value: chunk,
		Headers: []kafka.Header{
			{Key: "chunk_index", Value: []byte(strconv.Itoa(int(index)))},
		},
	}

	err := producer.WriteMessages(context.Background(), msg)
	if err != nil {
		fmt.Printf("❌ Failed to send chunk %d: %v\n", index, err)
	} else {
		fmt.Printf("✅ Sent chunk %d to Kafka\n", index)
	}
}

