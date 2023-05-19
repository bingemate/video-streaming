package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func main() {
	router := gin.Default()

	// Configuration de CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	// Récupérer le dossier source des vidéos depuis la variable d'environnement
	rootDir := os.Getenv("VIDEO_ROOT")

	// Vérifier si la variable d'environnement a été définie
	if rootDir == "" {
		panic("La variable d'environnement VIDEO_ROOT n'est pas définie.")
	}

	// Route pour servir les fichiers statiques
	group := router.Group("/streaming-service")

	group.StaticFS("/", http.Dir(rootDir))

	router.Run() // Lance le serveur
}
