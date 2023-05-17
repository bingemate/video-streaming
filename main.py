import os
import subprocess

input_file = "/home/nospy/Téléchargements/media/file.mkv"
output_folder = "/home/nospy/Téléchargements/media/hls/" + input_file.split("/")[-1]
chunk_duration = 10  # durée des segments en secondes

# Création du dossier de sortie
os.makedirs(output_folder, exist_ok=True)

# Suppression des fichiers existants
for file in os.listdir(output_folder):
    os.remove(os.path.join(output_folder, file))

print("Récupération des informations sur les pistes audio et sous-titres...")
ffprobe_command = [
    "ffprobe",
    "-v", "error",
    "-show_entries", "stream=index,codec_type,codec_name",
    "-of", "csv=p=0",
    input_file,
]
ffprobe_output = subprocess.check_output(ffprobe_command).decode().strip().split("\n")

audio_streams = []
subtitle_streams = []
video_codec = None

# Analyse de la sortie de ffprobe pour récupérer les informations sur les pistes audio, sous-titres et codec vidéo
for line in ffprobe_output:
    stream_index, codec_name, codec_type = line.split(",")
    if codec_type == "audio":
        audio_streams.append(stream_index)
    elif codec_type == "subtitle":
        subtitle_streams.append(stream_index)
    elif codec_type == "video":
        video_codec = codec_name

print("Pistes audio trouvées :", audio_streams)
print("Pistes de sous-titres trouvées :", subtitle_streams)
print("Codec vidéo :", video_codec)

# Transcodage de la vidéo sans audio ni sous-titres
print("Début du transcodage en HLS...")
ffmpeg_video_command = [
    "ffmpeg",
    "-i", input_file,
    "-map", "0:0",  # Sélectionnez seulement la première piste vidéo
    "-map", "0:1",  # Ajouter la première piste audio à la liste des pistes à ignorer
    "-c:a", "aac",
    "-b:a", "128k",
    "-c:v", "libx264",
    "-preset", "ultrafast",
    "-hls_time", f"{chunk_duration}",
    "-hls_playlist_type", "vod",
    "-hls_segment_filename", f"{output_folder}/segment_%03d.ts",
    "-hls_flags", "delete_segments",
    "-f", "hls", f"{output_folder}/output.m3u8"
]
subprocess.run(ffmpeg_video_command)

# Extraction de chaque piste audio
# for audio_stream in audio_streams:
#     audio_output_file = f"{output_folder}/audio_{audio_stream}.m3u8"
#     ffmpeg_audio_command = [
#         "ffmpeg",
#         "-i", input_file,
#         "-map", f"0:{audio_stream}",
#         "-c:a", "aac",
#         "-b:a", "128k",
#         "-hls_time", f"{chunk_duration}",
#         "-hls_playlist_type", "vod",
#         "-hls_segment_filename", f"{output_folder}/audio_{audio_stream}_%03d.ts",
#         audio_output_file
#     ]
#     subprocess.run(ffmpeg_audio_command)
#     print("Piste audio extraite :", audio_output_file)

# Extraction de chaque piste de sous-titres
for subtitle_stream in subtitle_streams:
    subtitle_output_file = f"{output_folder}/subtitle_{subtitle_stream}.vtt"
    ffmpeg_subtitle_command = [
        "ffmpeg",
        "-i", input_file,
        "-map", f"0:{subtitle_stream}",
        # "-scodec", "mov_text",
        subtitle_output_file
    ]
    subprocess.run(ffmpeg_subtitle_command)
    print("Piste de sous-titres extraite :", subtitle_output_file)

print("Transcodage terminé. Fichiers HLS générés dans :", output_folder)
