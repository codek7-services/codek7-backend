package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
    "github.com/lumbrjx/codek7/gateway/pb"
)

func (a API) GetVideoByID(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "video_id")
	if videoID == "" {
		http.Error(w, "missing video_id", http.StatusBadRequest)
		return
	}

	req := &pb.GetVideoRequest{
		VideoId: videoID,
	}

	// Call gRPC
	res, err := a.RepoClient.GetVideoByID(context.Background(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}
func (a API) GetRecentUserVideos(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "user_id")
    if userID == "" {
        http.Error(w, "missing user_id", http.StatusBadRequest)
        return
    }

    res, err := a.RepoClient.GetLast3UserVideos(context.Background(), &pb.GetLast3UserVideosRequest{UserId: userID})
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

   json.NewEncoder(w).Encode(res)
}
func (a API) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	res, err := a.RepoClient.GetUserVideos(context.Background(), &pb.GetUserVideosRequest{UserId: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (a API) DownloadVideo(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "video_id")
	if videoID == "" {
		http.Error(w, "missing video_id", http.StatusBadRequest)
		return
	}

	stream, err := a.RepoClient.DownloadVideo(context.Background(), &pb.DownloadVideoRequest{FileName: videoID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	first, err := stream.Recv()
	if err != nil {
		http.Error(w, "failed to receive metadata", http.StatusInternalServerError)
		return
	}

	metadata := first.GetMetadata()
	if metadata == nil {
		http.Error(w, "expected metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", metadata.FileName))
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.FileSize))

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if chunk := res.GetChunk(); chunk != nil {
			_, _ = w.Write(chunk.Data)
		}
	}
}
