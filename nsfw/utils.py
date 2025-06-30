import cv2
from transformers import AutoImageProcessor, AutoModelForImageClassification
import torch
import os
from dotenv import load_dotenv
from PIL import Image
import os
import grpc
import repo_pb2
import repo_pb2_grpc

load_dotenv()
print("Loading NSFW detection model...")
processor = AutoImageProcessor.from_pretrained("Falconsai/nsfw_image_detection")
model = AutoModelForImageClassification.from_pretrained("Falconsai/nsfw_image_detection")
model.eval() 
print("Model loaded successfully!")


def handleFrameByFrame(video_path, frame_rate=30):
    print("Starting frame extraction...")
    # Open video file to loop into it
    cap = cv2.VideoCapture(video_path)
    # keep track of the frame count to check every 30th frame
    count = 0
    nsfw_detected = False
    
    while cap.isOpened():
        ret, frame = cap.read()
        if not ret:
            break        
        if count % frame_rate == 0:
            print(f"Processing frame {count}")
            result = checkNSFW(frame)
            # Handle potential errors from checkNSFW
            if isinstance(result, dict) and 'is_nsfw' in result:
                if result['is_nsfw']:
                    nsfw_detected = True
                    break
            elif isinstance(result, dict) and 'error' in result:
                print(f"Error processing frame {count}: {result['error']}")
        
        count += 1
        
    cap.release()
    print(f"Finished processing {count} frames")
    return nsfw_detected

def checkNSFW(image):
    try:
        image = Image.fromarray(cv2.cvtColor(image, cv2.COLOR_BGR2RGB))
        # Process image
        inputs = processor(image, return_tensors="pt")   
        # Run inference
        with torch.no_grad():
            outputs = model(**inputs)
            predictions = torch.nn.functional.softmax(outputs.logits, dim=-1) 
        # Get probabilities
        nsfw_prob = predictions[0][1].item()  # Assuming index 1 is NSFW
        sfw_prob = predictions[0][0].item()   # Assuming index 0 is SFW
        
        result = {
            'is_nsfw': nsfw_prob > 0.5,
            'nsfw_probability': round(nsfw_prob, 2),
            'sfw_probability': round(sfw_prob, 2),
            'confidence': round(max(nsfw_prob, sfw_prob), 2)
        }
        print(f"NSFW Check Result: {result}")
        return result
        
    except Exception as e:
        error_response = {'error': str(e)}
        return error_response


def getVideoById(video_id):
    assembledVideoChunks = []
    try:
        # Create gRPC channel
        grpc_url = os.getenv('gRPC_URL', 'localhost:50051')  # Default to localhost:50051 if not set
        print(f"Connecting to gRPC server at: {grpc_url}")
        
        with grpc.insecure_channel(grpc_url) as channel:
            stub = repo_pb2_grpc.RepoServiceStub(channel)
            # Create request message
            request = repo_pb2.DownloadVideoRequest(video_id=str(video_id))
            # Make gRPC call - DownloadVideo returns a stream
            response_stream = stub.DownloadVideo(request)
            
            file_metadata = None
            for response in response_stream:
                if response.HasField('metadata'):
                    file_metadata = response.metadata
                    print(f"Receiving file: {file_metadata.file_name}, size: {file_metadata.file_size}")
                elif response.HasField('chunk'):
                    chunk = response.chunk
                    print(f"Received chunk {chunk.chunk_number}, size: {len(chunk.data)}")
                    assembledVideoChunks.append(chunk.data)
                    
                    if chunk.is_last:
                        print("Received last chunk")
                        break
            
            print("Video fetched successfully via gRPC!")
            
            # Create uploads directory if it doesn't exist
            uploads_dir = './uploads'
            if not os.path.exists(uploads_dir):
                os.makedirs(uploads_dir)
            
            # Determine file extension from metadata or default to .mp4
            file_extension = '.mp4'
            if file_metadata and file_metadata.file_name:
                _, ext = os.path.splitext(file_metadata.file_name)
                if ext:
                    file_extension = ext
            
            # Save the video temporarily in the file system
            video_path = f'{uploads_dir}/{video_id}{file_extension}'
            with open(video_path, 'wb') as f:
                f.write(b''.join(assembledVideoChunks))
            
            return video_path  # Return the path to the saved video file
            
    except grpc.RpcError as e:
        print(f"gRPC error: {e.code()} - {e.details()}")
        return None
    except Exception as e:
        print(f"Error fetching video: {e}")
        return None
