# NSFW Content Detection Application

A Flask-based application that detects NSFW content in uploaded videos.

## Setup & Installation

```bash
cd nsfw

### 1. Create and activate virtual environment
```bash
python -m venv nsfw_env

# On Windows
nsfw_env\Scripts\activate

# On Linux/Mac
source nsfw_env/bin/activate
```

### 2. Install requirements
```bash
pip install -r requirements.txt
```

## Running the Application

```bash
python app.py
```

The application will start on `http://localhost:5000`

## Usage

1. Open your browser and go to `http://localhost:5000`
2. Upload a video file through an http client with multipart/form, key named
3. The system will analyze the video frame by frame
4. You'll receive a response indicating whether NSFW content was detected

## Notes

- The application uses the Hugging Face model "Falconsai/nsfw_image_detection" for content analysis
- Uploaded videos are automatically deleted after processing
- Make sure the "uploads" directory exists before starting the application