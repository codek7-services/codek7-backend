package handler

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/lumbrjx/codek7/repo/internal/service"
	"github.com/lumbrjx/codek7/repo/pkg/pb/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RepoHandler struct {
	pb.UnimplementedRepoServiceServer
	userService  service.UserService
	videoService service.VideoService
}

func NewRepoHandler(userSvc service.UserService, videoSvc service.VideoService) *RepoHandler {
	return &RepoHandler{
		userService:  userSvc,
		videoService: videoSvc,
	}
}

func (h *RepoHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	user, err := h.userService.CreateUser(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}
	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *RepoHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := h.userService.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}
	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *RepoHandler) UploadVideo(stream pb.RepoService_UploadVideoServer) error {
	var metadata *pb.VideoMetadata
	buf := new(bytes.Buffer)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "failed to receive chunk: %v", err)
		}

		switch data := req.Data.(type) {
		case *pb.UploadVideoRequest_Metadata:
			metadata = data.Metadata
		case *pb.UploadVideoRequest_Chunk:
			buf.Write(data.Chunk.Data)
		}
	}

	if metadata == nil {
		return status.Error(codes.InvalidArgument, "missing video metadata")
	}

	video, err := h.videoService.UploadVideo(
		stream.Context(),
		metadata.UserId,
		metadata.Title,
		metadata.Description,
		metadata.FileName,
		buf.Bytes(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "upload failed: %v", err)
	}

	return stream.SendAndClose(&pb.VideoMetadataResponse{
		Id:          video.ID,
		UserId:      video.UserID,
		Title:       video.Title,
		Description: video.Description,
		CreatedAt:   video.CreatedAt.Format(time.RFC3339),
	})
}

func (h *RepoHandler) GetUserVideos(ctx context.Context, req *pb.GetUserVideosRequest) (*pb.VideoListResponse, error) {
	videos, err := h.videoService.GetVideosByUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch videos: %v", err)
	}

	resp := &pb.VideoListResponse{}
	for _, v := range videos {
		resp.Videos = append(resp.Videos, &pb.VideoMetadataResponse{
			Id:          v.ID,
			UserId:      v.UserID,
			Title:       v.Title,
			Description: v.Description,
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (h *RepoHandler) GetVideoByID(ctx context.Context, req *pb.GetVideoRequest) (*pb.VideoMetadataResponse, error) {
	v, err := h.videoService.GetVideoByID(ctx, req.VideoId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "video not found: %v", err)
	}
	return &pb.VideoMetadataResponse{
		Id:          v.ID,
		UserId:      v.UserID,
		Title:       v.Title,
		Description: v.Description,
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *RepoHandler) DownloadVideo(req *pb.DownloadVideoRequest, stream pb.RepoService_DownloadVideoServer) error {
	content, filename, err := h.videoService.DownloadVideo(stream.Context(), req.VideoId)
	if err != nil {
		return status.Errorf(codes.Internal, "download failed: %v", err)
	}

	// Send metadata
	_ = stream.Send(&pb.VideoFileResponse{
		Data: &pb.VideoFileResponse_Metadata{
			Metadata: &pb.VideoFileMetadata{
				FileName:    filename,
				FileSize:    int64(len(content)),
				ContentType: "video/mp4",
			},
		},
	})

	// Send chunks
	const chunkSize = 512 * 1024 // 512 KB
	n := len(content)
	for i := 0; i < n; i += chunkSize {
		end := i + chunkSize
		if end > n {
			end = n
		}
		chunk := content[i:end]
		isLast := end == n

		_ = stream.Send(&pb.VideoFileResponse{
			Data: &pb.VideoFileResponse_Chunk{
				Chunk: &pb.VideoFileChunk{
					Data:        chunk,
					ChunkNumber: int32(i / chunkSize),
					IsLast:      isLast,
				},
			},
		})
	}

	return nil
}
