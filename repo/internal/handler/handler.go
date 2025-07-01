package handler

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/lumbrjx/codek7/repo/internal/service"
	"codek7/common/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
	user, err := h.userService.GetUser(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}
	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UploadVideo handles both original videos and generated files
func (h *RepoHandler) UploadVideo(stream pb.RepoService_UploadVideoServer) error {
	var metadata *pb.VideoMetadata
	buf := new(bytes.Buffer)

	// Collect all chunks
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

	// Determine if this is an original video or generated file
	isOriginalVideo := h.isOriginalVideo(metadata.FileName)

	if isOriginalVideo {
		// Handle original video upload (with DB metadata)
		video, err := h.videoService.UploadOriginalVideo(
			stream.Context(),
			metadata.UserId,
			metadata.Title,
			metadata.Description,
			metadata.FileName,
			buf.Bytes(),
		)
		if err != nil {
			return status.Errorf(codes.Internal, "original video upload failed: %v", err)
		}

		return stream.SendAndClose(&pb.VideoMetadataResponse{
			Id:          video.ID,
			UserId:      video.UserID,
			Title:       video.Title,
			Description: video.Description,
			CreatedAt:   video.CreatedAt.Format(time.RFC3339),
		})
	} else {
		// Handle generated file upload (no DB metadata)
		err := h.videoService.UploadGeneratedFile(
			stream.Context(),
			metadata.FileName,
			buf.Bytes(),
		)
		if err != nil {
			return status.Errorf(codes.Internal, "generated file upload failed: %v", err)
		}

		// Return a simple response for generated files
		return stream.SendAndClose(&pb.VideoMetadataResponse{
			Id:          "", // No ID for generated files
			UserId:      metadata.UserId,
			Title:       metadata.Title,
			Description: "Generated file uploaded successfully",
			CreatedAt:   time.Now().Format(time.RFC3339),
		})
	}
}

// isOriginalVideo determines if the uploaded file is an original video or generated content
func (h *RepoHandler) isOriginalVideo(fileName string) bool {
	// Original videos typically don't have resolution suffixes or are .mp4 without special naming
	// Generated files have patterns like: videoID_360p.mp4, videoID/360/index.m3u8, etc.
	
	// Check for resolution patterns
	resolutionPatterns := []string{"_144p.", "_240p.", "_360p.", "_480p.", "_720p.", "_1080p."}
	for _, pattern := range resolutionPatterns {
		if strings.Contains(fileName, pattern) {
			return false
		}
	}
	
	// Check for HLS patterns
	hlsPatterns := []string{"/index.m3u8", "seg_", "_master.m3u8"}
	for _, pattern := range hlsPatterns {
		if strings.Contains(fileName, pattern) {
			return false
		}
	}
	
	// Check for .ts segments
	if strings.HasSuffix(fileName, ".ts") {
		return false
	}
	
	// If none of the generated file patterns match, assume it's original
	return true
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
	content, filename, err := h.videoService.DownloadFile(stream.Context(), req.FileName)
	if err != nil {
		return status.Errorf(codes.Internal, "download failed: %v", err)
	}

	// Determine content type based on file extension
	contentType := h.getContentType(filename)

	// Send metadata first
	err = stream.Send(&pb.VideoFileResponse{
		Data: &pb.VideoFileResponse_Metadata{
			Metadata: &pb.VideoFileMetadata{
				FileName:    filename,
				FileSize:    int64(len(content)),
				ContentType: contentType,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send metadata: %v", err)
	}

	// Send chunks
	const chunkSize = 512 * 1024 // 512 KB
	n := len(content)
	chunkNumber := int32(0)
	
	for i := 0; i < n; i += chunkSize {
		end := i + chunkSize
		if end > n {
			end = n
		}
		chunk := content[i:end]
		isLast := end == n

		err = stream.Send(&pb.VideoFileResponse{
			Data: &pb.VideoFileResponse_Chunk{
				Chunk: &pb.VideoFileChunk{
					Data:        chunk,
					ChunkNumber: chunkNumber,
					IsLast:      isLast,
				},
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send chunk %d: %v", chunkNumber, err)
		}
		chunkNumber++
	}

	return nil
}

// getContentType determines the MIME type based on file extension
func (h *RepoHandler) getContentType(filename string) string {
	if strings.HasSuffix(filename, ".mp4") {
		return "video/mp4"
	} else if strings.HasSuffix(filename, ".m3u8") {
		return "application/x-mpegURL"
	} else if strings.HasSuffix(filename, ".ts") {
		return "video/MP2T"
	}
	return "application/octet-stream"
}

func (h *RepoHandler) RemoveVideo(ctx context.Context, req *pb.GetVideoRequest) (*emptypb.Empty, error) {
	if err := h.videoService.RemoveVideo(ctx, req.VideoId); err != nil {
		return nil, status.Errorf(codes.Internal, "remove video failed: %v", err)
	}
	return &emptypb.Empty{}, nil
}
