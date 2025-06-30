from flask import *
from utils import *
import os
app = Flask(__name__)


@app.route('/')
def main():
    return render_template("index.html")


@app.route('/upload/', methods=['POST'])
def upload():
    print("Received a file upload request")
    if request.method == 'POST':
        # getting the video id from the post request
        videoId = request.form.getlist("id")[0]
        print(f"Video ID received: {videoId}")
        filePath = getVideoById(videoId)
        # userID= request.files.getlist("userId")
        print(f"Processing video: {videoId}")
        # 1 if NSFW frame detected, 0 if not
        isNSFW=handleFrameByFrame(filePath)
        if isNSFW:
            print(f"NSFW content detected in {filePath}")
            os.remove(filePath)
            return jsonify({"isNSFW":True,"error": "NSFW content detected in the uploaded video."}), 200
        else:
            print(f"No NSFW content detected in {filePath}")
            # os.remove(filePath)
            return jsonify({"isNSFW":False,"message": "No NSFW content detected in the uploaded video."}), 200
    return jsonify({"message":"file wasnt uploaded correctly"}), 400,


if __name__ == '__main__':
    app.run(port=5000,debug=True)