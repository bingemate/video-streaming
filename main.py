import os
import subprocess

# Réglages par défaut
INPUT_FILE = "/home/nospy/Téléchargements/file.mkv"
OUTPUT_FOLDER = "/home/nospy/Téléchargements/streaming/" + INPUT_FILE.split("/")[-1]
CHUNK_DURATION = 10  # durée des segments en secondes


def prepare_output_folder(output_folder):
    """
    Prépare le dossier de sortie en le créant si nécessaire et en supprimant tous les fichiers existants.
    """
    os.makedirs(output_folder, exist_ok=True)
    for file in os.listdir(output_folder):
        os.remove(os.path.join(output_folder, file))


def extract_streams_info(input_file):
    """
    Utilise ffprobe pour extraire des informations sur les pistes audio, sous-titres et codec vidéo
    """
    print("Récupération des informations sur les pistes audio et sous-titres...")
    ffprobe_command = [
        "ffprobe",
        "-v", "error",
        "-show_entries", "stream=index,codec_name,codec_type",
        "-of", "csv=p=0",
        input_file,
    ]
    ffprobe_output = subprocess.check_output(ffprobe_command).decode().strip().split("\n")

    audio_streams = []
    subtitle_streams = []
    video_codec = None

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

    return audio_streams, subtitle_streams, video_codec


def transcode_video(input_file, output_folder, chunk_duration):
    """
    Transcode la vidéo sans audio ni sous-titres
    """
    print("Début du transcodage en HLS...")
    ffmpeg_video_command = [
        "ffmpeg",
        "-i", input_file,
        "-map", "0:0",  # Sélectionnez seulement la première piste vidéo
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-vf", "scale=1280:720",  # Rescale en 720p    ,
        "-pix_fmt", "yuv420p",
        "-hls_time", f"{chunk_duration}",
        "-hls_playlist_type", "vod",
        "-hls_segment_filename", f"{output_folder}/segment_%03d.ts",
        "-hls_flags", "delete_segments",
        "-f", "hls", f"{output_folder}/index.m3u8"
    ]
    subprocess.run(ffmpeg_video_command)


def extract_audio_streams(input_file, audio_streams, output_folder, chunk_duration):
    """
    Extrait chaque piste audio
    """
    for audio_stream in audio_streams:
        audio_output_file = f"{output_folder}/audio_{audio_stream}.m3u8"
        ffmpeg_audio_command = [
            "ffmpeg",
            "-i", input_file,
            "-map", f"0:{audio_stream}",
            "-c:a", "aac",
            "-b:a", "160k",
            "-ac", "2",
            "-hls_time", f"{chunk_duration}",
            "-hls_playlist_type", "vod",
            "-hls_segment_filename", f"{output_folder}/audio_{audio_stream}_%03d.ts",
            audio_output_file
        ]
        subprocess.run(ffmpeg_audio_command)
        print("Piste audio extraite :", audio_output_file)


def extract_subtitle_streams(input_file, subtitle_streams, output_folder):
    """
    Extrait chaque piste de sous-titres
    """
    for subtitle_stream in subtitle_streams:
        subtitle_output_file = f"{output_folder}/subtitle_{subtitle_stream}.vtt"
        ffmpeg_subtitle_command = [
            "ffmpeg",
            "-i", input_file,
            "-map", f"0:{subtitle_stream}",
            subtitle_output_file
        ]
        subprocess.run(ffmpeg_subtitle_command)
        print("Piste de sous-titres extraite :", subtitle_output_file)


def main(input_file, output_folder, chunk_duration):
    prepare_output_folder(output_folder)
    audio_streams, subtitle_streams, video_codec = extract_streams_info(input_file)
    transcode_video(input_file, output_folder, chunk_duration)
    extract_audio_streams(input_file, audio_streams, output_folder, chunk_duration)
    extract_subtitle_streams(input_file, subtitle_streams, output_folder)
    print("Transcodage terminé. Fichiers HLS générés dans :", output_folder)


if __name__ == "__main__":
    main(INPUT_FILE, OUTPUT_FOLDER, CHUNK_DURATION)
