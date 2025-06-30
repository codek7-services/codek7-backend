import cv2
from transformers import AutoImageProcessor, AutoModelForImageClassification
import torch
from PIL import Image

print("Loading NSFW detection model...")
processor = AutoImageProcessor.from_pretrained("Falconsai/nsfw_image_detection")
model = AutoModelForImageClassification.from_pretrained("Falconsai/nsfw_image_detection")
model.eval()  # Set to evaluation mode
print("Model loaded successfully!")


def handleFrameByFrame(video_path, frame_rate=30):
    print("Starting frame extraction...")
    cap = cv2.VideoCapture(video_path)
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
