FROM golang:1.20-bullseye as builder

# Copie le code Go dans le conteneur.
COPY . /app

# Se déplace dans le répertoire de travail de l'application.
WORKDIR /app

RUN apt update && \
    apt install -y libgstreamer1.0-dev libgstreamer-plugins-base1.0-dev libgstreamer-plugins-bad1.0-dev gstreamer1.0-libav

# Compile l'application Go.
RUN go build -o video-player .

# Définit l'image de base.
FROM debian:bullseye-slim

# Installe les dépendances nécessaires pour GStreamer.
RUN apt update && \
    apt install -y libgstreamer1.0-0 libgstreamer-plugins-base1.0-0 libgstreamer-plugins-bad1.0-0 gstreamer1.0-libav && \
    rm -rf /var/lib/apt/lists/* && \
    rm -rf /var/cache/apt/*

# Copie l'application compilée dans l'image.
COPY --from=builder /app/video-player /app/video-player

# Expose le port utilisé par l'application.
EXPOSE 8080

# Se déplace dans le répertoire de travail de l'application.
WORKDIR /app

# Démarre l'application vidéo.
ENTRYPOINT ["./video-player"]
