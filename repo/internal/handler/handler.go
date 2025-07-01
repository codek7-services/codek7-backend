package handler

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"codek7/common/pb"

	"github.com/lumbrjx/codek7/repo/internal/service"
	"github.com/lumbrjx/codek7/repo/pkg/logger"
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
	start := time.Now()

	logger.Logger.Info("Creating user",
		"username", req.Username,
		"email", req.Email,
	)

	user, err := h.userService.CreateUser(ctx, req.Password, req.Email, req.Username)

	logger.LogGRPCRequest(ctx, "CreateUser", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to create user",
			"username", req.Username,
			"email", req.Email,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	logger.Logger.Info("User created successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *RepoHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	start := time.Now()

	logger.Logger.Info("Fetching user",
		"username", req.Username,
	)

	user, err := h.userService.GetUser(ctx, req.Username)

	logger.LogGRPCRequest(ctx, "GetUser", time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("User not found",
			"username", req.Username,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	logger.Logger.Info("User fetched successfully",
		"user_id", user.ID,
		"username", user.Username,
	)

	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		Password:  user.Password,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UploadVideo handles both original videos and generated files
func (h *RepoHandler) UploadVideo(stream pb.RepoService_UploadVideoServer) error {
	start := time.Now()
	var metadata *pb.VideoMetadata
	buf := new(bytes.Buffer)

	logger.Logger.Info("Starting video upload stream")

	// Collect all chunks
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Logger.Error("Failed to receive chunk",
				"error", err.Error(),
			)
			return status.Errorf(codes.Unknown, "failed to receive chunk: %v", err)
		}

		switch data := req.Data.(type) {
		case *pb.UploadVideoRequest_Metadata:
			metadata = data.Metadata
			logger.Logger.Info("Received video metadata",
				"user_id", metadata.UserId,
				"title", metadata.Title,
				"filename", metadata.FileName,
			)
		case *pb.UploadVideoRequest_Chunk:
			buf.Write(data.Chunk.Data)
		}
	}

	if metadata == nil {
		logger.Logger.Error("Missing video metadata")
		return status.Error(codes.InvalidArgument, "missing video metadata")
	}

	fileSize := int64(buf.Len())
	logger.Logger.Info("Video upload chunks collected",
		"filename", metadata.FileName,
		"file_size_bytes", fileSize,
	)

	// Determine if this is an original video or generated file
	isOriginalVideo := h.isOriginalVideo(metadata.FileName)

	if isOriginalVideo {
		logger.Logger.Info("Processing original video upload",
			"filename", metadata.FileName,
			"user_id", metadata.UserId,
		)

		// Handle original video upload (with DB metadata)
		video, err := h.videoService.UploadOriginalVideo(
			stream.Context(),
			metadata.UserId,
			metadata.Title,
			metadata.Description,
			metadata.FileName,
			buf.Bytes(),
		)

		logger.LogGRPCRequest(stream.Context(), "UploadVideo-Original", time.Since(start), err)

		if err != nil {
			logger.Logger.Error("Original video upload failed",
				"filename", metadata.FileName,
				"user_id", metadata.UserId,
				"error", err.Error(),
			)
			return status.Errorf(codes.Internal, "original video upload failed: %v", err)
		}

		logger.Logger.Info("Original video uploaded successfully",
			"video_id", video.ID,
			"filename", metadata.FileName,
			"user_id", metadata.UserId,
			"file_size_bytes", fileSize,
		)

		return stream.SendAndClose(&pb.VideoMetadataResponse{
			Id:          video.ID,
			UserId:      video.UserID,
			Title:       video.Title,
			Description: video.Description,
			CreatedAt:   video.CreatedAt.Format(time.RFC3339),
		})
	} else {
		logger.Logger.Info("Processing generated file upload",
			"filename", metadata.FileName,
			"user_id", metadata.UserId,
		)

		// Handle generated file upload (no DB metadata)
		err := h.videoService.UploadGeneratedFile(
			stream.Context(),
			metadata.FileName,
			buf.Bytes(),
		)

		logger.LogGRPCRequest(stream.Context(), "UploadVideo-Generated", time.Since(start), err)

		if err != nil {
			logger.Logger.Error("Generated file upload failed",
				"filename", metadata.FileName,
				"user_id", metadata.UserId,
				"error", err.Error(),
			)
			return status.Errorf(codes.Internal, "generated file upload failed: %v", err)
		}

		logger.Logger.Info("Generated file uploaded successfully",
			"filename", metadata.FileName,
			"user_id", metadata.UserId,
			"file_size_bytes", fileSize,
		)

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
func (h *RepoHandler) GetLast3UserVideos(ctx context.Context, req *pb.GetLast3UserVideosRequest) (*pb.Video3ListResponse, error) {
	start := time.Now()

	logger.Logger.Info("Fetching last 3 videos for user",
		"user_id", req.UserId,
	)

	videos, err := h.videoService.GetLast3VideosByUser(ctx, req.UserId)

	logger.LogGRPCRequest(ctx, "GetLast3UserVideos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to fetch last 3 videos",
			"user_id", req.UserId,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.Internal, "failed to fetch videos: %v", err)
	}

	logger.Logger.Info("Successfully fetched last 3 videos",
		"user_id", req.UserId,
		"video_count", len(videos),
	)

	resp := &pb.Video3ListResponse{}
	for _, v := range videos {
		resp.Videos = append(resp.Videos, &pb.VideoMetadataResponse{
			Id:          v.ID,
			UserId:      v.UserID,
			Title:       v.Title,
			Description: v.Description,
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
			FileName:    v.FileName,
		})
	}
	return resp, nil
}
func (h *RepoHandler) GetUserVideos(ctx context.Context, req *pb.GetUserVideosRequest) (*pb.VideoListResponse, error) {
	start := time.Now()

	logger.Logger.Info("Fetching all videos for user",
		"user_id", req.UserId,
	)

	videos, err := h.videoService.GetVideosByUser(ctx, req.UserId)

	logger.LogGRPCRequest(ctx, "GetUserVideos", time.Since(start), err)

	if err != nil {
		logger.Logger.Error("Failed to fetch user videos",
			"user_id", req.UserId,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.Internal, "failed to fetch videos: %v", err)
	}

	logger.Logger.Info("Successfully fetched user videos",
		"user_id", req.UserId,
		"video_count", len(videos),
	)

	resp := &pb.VideoListResponse{}
	for _, v := range videos {
		resp.Videos = append(resp.Videos, &pb.VideoMetadataResponse{
			Id:          v.ID,
			UserId:      v.UserID,
			Title:       v.Title,
			Description: v.Description,
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
			FileName:    v.FileName,
		})
	}
	return resp, nil
}

func (h *RepoHandler) GetVideoByID(ctx context.Context, req *pb.GetVideoRequest) (*pb.VideoMetadataResponse, error) {
	start := time.Now()

	logger.Logger.Info("Fetching video by ID",
		"video_id", req.VideoId,
	)

	v, err := h.videoService.GetVideoByID(ctx, req.VideoId)

	logger.LogGRPCRequest(ctx, "GetVideoByID", time.Since(start), err)

	if err != nil {
		logger.Logger.Warn("Video not found",
			"video_id", req.VideoId,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.NotFound, "video not found: %v", err)
	}

	logger.Logger.Info("Successfully fetched video",
		"video_id", v.ID,
		"title", v.Title,
		"user_id", v.UserID,
	)

	return &pb.VideoMetadataResponse{
		Id:          v.ID,
		UserId:      v.UserID,
		Title:       v.Title,
		Description: v.Description,
		FileName:    v.FileName,
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (h *RepoHandler) DownloadVideo(req *pb.DownloadVideoRequest, stream pb.RepoService_DownloadVideoServer) error {
	start := time.Now()

	logger.Logger.Info("Starting video download",
		"filename", req.FileName,
	)

	content, filename, err := h.videoService.DownloadFile(stream.Context(), req.FileName)

	if err != nil {
		logger.Logger.Error("Video download failed",
			"filename", req.FileName,
			"error", err.Error(),
		)
		return status.Errorf(codes.Internal, "download failed: %v", err)
	}

	fileSize := int64(len(content))
	logger.Logger.Info("Video content retrieved",
		"filename", filename,
		"file_size_bytes", fileSize,
	)

	// Determine content type based on file extension
	contentType := h.getContentType(filename)

	// Send metadata first
	err = stream.Send(&pb.VideoFileResponse{
		Data: &pb.VideoFileResponse_Metadata{
			Metadata: &pb.VideoFileMetadata{
				FileName:    filename,
				FileSize:    fileSize,
				ContentType: contentType,
			},
		},
	})
	if err != nil {
		logger.Logger.Error("Failed to send download metadata",
			"filename", filename,
			"error", err.Error(),
		)
		return status.Errorf(codes.Internal, "failed to send metadata: %v", err)
	}

	// Send chunks
	const chunkSize = 512 * 1024 // 512 KB
	n := len(content)
	chunkNumber := int32(0)
	totalChunks := (n + chunkSize - 1) / chunkSize

	logger.Logger.Info("Starting to send file chunks",
		"filename", filename,
		"total_chunks", totalChunks,
		"chunk_size_kb", chunkSize/1024,
	)

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
			logger.Logger.Error("Failed to send chunk",
				"filename", filename,
				"chunk_number", chunkNumber,
				"error", err.Error(),
			)
			return status.Errorf(codes.Internal, "failed to send chunk %d: %v", chunkNumber, err)
		}
		chunkNumber++
	}

	logger.LogGRPCRequest(stream.Context(), "DownloadVideo", time.Since(start), nil)
	logger.Logger.Info("Video download completed successfully",
		"filename", filename,
		"file_size_bytes", fileSize,
		"chunks_sent", chunkNumber,
	)

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
	start := time.Now()

	logger.Logger.Info("Removing video",
		"video_id", req.VideoId,
	)

	if err := h.videoService.RemoveVideo(ctx, req.VideoId); err != nil {
		logger.Logger.Error("Failed to remove video",
			"video_id", req.VideoId,
			"error", err.Error(),
		)
		return nil, status.Errorf(codes.Internal, "remove video failed: %v", err)
	}

	logger.LogGRPCRequest(ctx, "RemoveVideo", time.Since(start), nil)
	logger.Logger.Info("Video removed successfully",
		"video_id", req.VideoId,
	)

	return &emptypb.Empty{}, nil
}
