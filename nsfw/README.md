# NSFW Video Processing Service Documentation

## Overview

This service implements an automated NSFW (Not Safe For Work) content detection system for video files using a microservices architecture. The system combines RabbitMQ message queuing, gRPC for video operations, and machine learning for content analysis. When NSFW content is detected, the service automatically removes the video from the repository.

## Architecture Components

### Message Queue Consumer (RabbitMQ)
The service acts as a consumer for RabbitMQ messages on the `verify_nsfw` queue. When a video filename is received:

1. Connects to RabbitMQ using URL parameters from environment variables
2. Declares and listens on the `verify_nsfw` queue
3. Processes incoming video filenames for NSFW analysis
4. Triggers video deletion via gRPC if NSFW content is detected

### Video Operations (gRPC)
The service performs two main gRPC operations:

**Video Retrieval**: `getVideoById(filename)`
- Uses `DownloadVideoRequest` with `file_name` parameter
- Connects to gRPC server using `gRPC_URL` environment variable
- Streams video chunks from the repository service using `DownloadVideo` RPC
- Assembles `VideoFileChunk` data into complete video file
- Saves temporarily to `./uploads/` directory with original filename
- Returns local file path for processing

**Video Deletion**: When NSFW content is detected
- Makes `RemoveVideo` gRPC call with the video ID
- Permanently removes the video from the repository
- Returns `google.protobuf.Empty` response on successful deletion

### AI Model: Falconsai NSFW Image Detection

The service uses a specialized computer vision model from Hugging Face for NSFW content detection:

**Model Architecture**: `Falconsai/nsfw_image_detection`
- Based on a fine-tuned image classification transformer
- Binary classification model trained specifically for NSFW vs SFW content
- Uses Vision Transformer (ViT) or similar architecture for robust image understanding
- Pre-trained on large datasets of labeled safe and unsafe content

**Model Components**:
- **AutoImageProcessor**: Handles image preprocessing including resizing, normalization, and tensor conversion
- **AutoModelForImageClassification**: The core neural network that outputs classification logits
- **Softmax Layer**: Converts raw logits to probability distributions

**Model Performance**:
- Optimized for high precision to minimize false positives
- Capable of detecting various types of NSFW content including nudity, sexual content, and graphic material
- Uses attention mechanisms to focus on relevant image regions
- Inference runs efficiently on CPU with torch.no_grad() for production use

**Classification Process**:
1. Input image converted from BGR (OpenCV) to RGB (PIL) format
2. Image preprocessed to model's expected input format (typically 224x224 pixels)
3. Forward pass through transformer layers with attention mechanisms
4. Output logits converted to probabilities via softmax
5. Decision threshold set at 0.5 probability for NSFW classification

### Frame Processing Pipeline

**Function**: `handleFrameByFrame(video_path, frame_rate=30)`
1. Opens video file using OpenCV's VideoCapture
2. Extracts every 30th frame to balance accuracy with performance
3. Converts frames from BGR to RGB color space for model compatibility
4. Runs NSFW detection on each sampled frame using the AI model
5. Returns `True` immediately if any frame contains NSFW content (early termination)

**Function**: `checkNSFW(image)`
1. Converts OpenCV image array to PIL Image format
2. Applies model-specific preprocessing transformations
3. Runs inference with gradient computation disabled for efficiency
4. Applies softmax activation to get normalized probabilities
5. Returns detailed classification results with confidence metrics

## Processing Flow

1. **Message Reception**: RabbitMQ callback receives video filename
2. **Video Retrieval**: gRPC `DownloadVideo` streams video file chunks by filename
3. **File Assembly**: `VideoFileChunk` data assembled and saved locally
4. **AI Analysis**: Video processed frame-by-frame using Falconsai NSFW detection model
5. **Content Decision**: 
   - If NSFW detected: gRPC `RemoveVideo` call deletes video from repository
   - If safe: Video remains in repository
6. **Cleanup**: Local temporary files removed
7. **Response**: JSON response with detection results

## Environment Configuration

Required environment variables:
- `RABBITMQ_HOST`: RabbitMQ connection URL
- `gRPC_URL`: gRPC server endpoint (defaults to localhost:50051)



## AI Model Response Format

**Detailed Detection Results**:
```json
{
  "is_nsfw": boolean,
  "nsfw_probability": float,
  "sfw_probability": float,
  "confidence": float
}
```

**Service Response Format**:

**NSFW Detected (Video Deleted)**:
```json
{
  "isNSFW": true,
  "error": "NSFW content detected in the uploaded video."
}
```

**Safe Content**:
```json
{
  "isNSFW": false,
  "message": "No NSFW content detected in the uploaded video."
}
```

## Error Handling

- gRPC connection failures are caught and logged with specific error codes
- AI model inference errors return detailed error dictionaries
- File system operations include directory creation and cleanup checks
- Frame processing continues gracefully on individual frame errors
- Video deletion failures are logged but don't prevent response

## Performance Considerations

- **Model Efficiency**: Falconsai model loaded once at startup to avoid repeated initialization overhead
- **Frame Sampling**: Processing every 30th frame significantly reduces computation time while maintaining detection accuracy
- **Memory Management**: Torch inference runs with gradient computation disabled to reduce memory usage
- **Early Termination**: Analysis stops immediately when NSFW content is detected
- **Streaming**: gRPC streaming handles large video files without loading entire content into memory
- **Auto-acknowledgment**: RabbitMQ messages acknowledged automatically for faster throughput
message.txt
7 KB
