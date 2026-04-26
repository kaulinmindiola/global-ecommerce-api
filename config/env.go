package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv busca un archivo .env desde la raíz del proyecto y lo carga
// en las variables de entorno del sistema operativo.
// No sobreescribe variables que ya existan (útil para Docker/Producción).
func LoadEnv() {
	// Intentamos cargar el .env en el directorio actual
	err := godotenv.Load()
	if err == nil {
		log.Println("✓ Loaded environment variables from .env file")
		return
	}

	// Fallback para tests: busca en directorios superiores
	// Esto es útil cuando corres tests dentro de internal/domain/
	dir, _ := os.Getwd()
	for i := 0; i < 4; i++ { // Busca hasta 4 niveles arriba
		envPath := filepath.Join(dir, ".env")
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("✓ Loaded environment variables from %s", envPath)
			return
		}
		dir = filepath.Dir(dir)
	}

	// Si no encuentra el archivo .env, asumimos que las variables ya fueron
	// inyectadas por el sistema (ej. Docker, Kubernetes, GitHub Actions).
	log.Println("ℹ No .env file found, relying on system environment variables")
}
