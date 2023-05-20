package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func main() {
	router := gin.Default()

	// Configuration de CORS
	addCors(router)

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

func addCors(engine *gin.Engine) gin.IRoutes {
	return engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}
