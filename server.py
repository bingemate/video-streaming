import os
from flask import Flask, send_from_directory
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

@app.route('/<path:filename>')
def serve_file(filename):
    root_dir = '/home/nospy/Téléchargements/media/hls'
    return send_from_directory(root_dir, filename)

if __name__ == '__main__':
    app.run()
