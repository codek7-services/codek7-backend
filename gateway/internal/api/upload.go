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

	chunkBuf := make([]byte, 32*1024)
	videoID := uuid.New().String()
	sem := make(chan struct{}, 10) // limit to 10 goroutines
	index := 0

	for {
		n, err := file.Read(chunkBuf)
		if err != nil && err != io.EOF {
			log.Printf("❌ Error reading file: %v\n", err)
			http.Error(w, `{"status":"error","message":"Error reading file"}`, http.StatusInternalServerError)
			return
		}
		if n == 0 {
			break
		}

		// copy to avoid data race
		chunk := make([]byte, n)
		copy(chunk, chunkBuf[:n])

		sem <- struct{}{}
		go func(i int, chunk []byte) {
			defer func() { <-sem }()
			produceChunk(a.Producer, videoID, int32(i), chunk)
		}(index, chunk)

		index++
	}

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
