# Étape de construction
FROM golang:1.20 AS builder

ENV GO111MODULE=on


WORKDIR /app

# Copier les fichiers de l'application
COPY . .

# Compilation de l'application
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -x -ldflags "-s -w" -o app .

# Étape de production
FROM alpine:latest

# Installation des dépendances nécessaires
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copier l'exécutable construit précédemment
COPY --from=builder /app/app .

ENV VIDEO_ROOT=/mnt/media

# Définir le dossier /mnt/media en tant que volume
VOLUME ["/mnt/media"]

# Exposition du port de l'application
EXPOSE 8080

USER 1000:100

# Commande pour démarrer l'application
CMD ["./app"]
