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
        files = request.files.getlist("file")
        # userID= request.files.getlist("userId")
        for file in files:
            file.save(f'./uploads/{file.filename}')
            print(f"Processing video: {file.filename}")
            # 1 if NSFW frame detected, 0 if not
            isNSFW=handleFrameByFrame(f'./uploads/{file.filename}')
            if isNSFW:
                print(f"NSFW content detected in {file.filename}")
                os.remove(f'./uploads/{file.filename}')
                return jsonify({"isNSFW":True,"error": "NSFW content detected in the uploaded video."}), 200
            else:
                print(f"No NSFW content detected in {file.filename}")
                os.remove(f'./uploads/{file.filename}')
                return jsonify({"isNSFW":False,"message": "No NSFW content detected in the uploaded video."}), 200
        return jsonify({"message":"file wasnt uploaded correctly"}), 400,


if __name__ == '__main__':
    app.run(port=5000,debug=True)