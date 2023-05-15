FROM golang:1.20-alpine as builder

# Copie le code Go dans le conteneur.
COPY . /app

# Se déplace dans le répertoire de travail de l'application.
WORKDIR /app

# Compile l'application Go.
RUN go build -o video-player

# Définit l'image de base.
FROM alpine:latest

# Installe les dépendances nécessaires pour GStreamer.
RUN apk update && \
        apk add --no-cache git build-base gstreamer gst-plugins-base gst-plugins-good gst-plugins-bad gst-plugins-ugly


# Copie l'application compilée dans l'image.
COPY --from=builder /app/video-player /app/video-player

# Expose le port utilisé par l'application.
EXPOSE 8080

# Se déplace dans le répertoire de travail de l'application.
WORKDIR /app

# Démarre l'application vidéo.
ENTRYPOINT ["./video-player"]
