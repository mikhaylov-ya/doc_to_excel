package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create temp directory for uploads
	if err := os.MkdirAll("./temp", 0755); err != nil {
		log.Fatal("Failed to create temp directory:", err)
	}

	// Set Gin to release mode for production
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production

	router := gin.Default()

	// CORS middleware for cross-origin requests
	router.Use(corsMiddleware())

	// Increase max upload size to 50MB
	router.MaxMultipartMemory = 50 << 20 // 50 MB

	// API endpoints
	router.GET("/health", healthCheck)
	router.POST("/api/convert", handleConvert)

	// Serve frontend static files
	router.StaticFile("/", "./frontend/index.html")
	router.Static("/static", "./frontend/static")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on http://0.0.0.0:%s\n", port)
	log.Printf("📁 Upload limit: 50MB\n")
	log.Printf("🌐 Frontend: http://localhost:%s\n", port)
	log.Printf("🔧 Health check: http://localhost:%s/health\n", port)

	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// healthCheck returns server status
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "doc-to-excel",
		"version": "1.0.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// handleConvert processes uploaded DOC/DOCX file and returns Excel
func handleConvert(c *gin.Context) {
	// 1. Get uploaded file
	file, err := c.FormFile("document")
	if err != nil {
		log.Printf("❌ Upload error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file uploaded. Please select a .doc or .docx file",
		})
		return
	}

	// 2. Validate file extension
	ext := filepath.Ext(file.Filename)
	if ext != ".doc" && ext != ".docx" {
		log.Printf("❌ Invalid file type: %s\n", file.Filename)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid file type '%s'. Only .doc and .docx files are allowed", ext),
		})
		return
	}

	// 3. Validate file size (50MB max)
	if file.Size > 50<<20 {
		log.Printf("❌ File too large: %.2f MB\n", float64(file.Size)/(1024*1024))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File too large (%.2f MB). Maximum allowed: 50 MB", float64(file.Size)/(1024*1024)),
		})
		return
	}

	// 4. Create unique filenames with timestamp
	timestamp := time.Now().UnixNano()
	inputFilename := fmt.Sprintf("%d_%s", timestamp, file.Filename)
	outputFilename := fmt.Sprintf("%d_output.xlsx", timestamp)

	inputPath := filepath.Join("./temp", inputFilename)
	outputPath := filepath.Join("./temp", outputFilename)

	// 5. Save uploaded file
	if err := c.SaveUploadedFile(file, inputPath); err != nil {
		log.Printf("❌ Failed to save file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save uploaded file",
		})
		return
	}

	log.Printf("📥 Uploaded: %s (%.2f MB)\n", file.Filename, float64(file.Size)/(1024*1024))

	// 6. Process document (convert to Excel)
	log.Printf("⚙️  Processing: %s\n", file.Filename)
	err = processDocument(inputPath, outputPath)
	if err != nil {
		log.Printf("❌ Processing failed: %v\n", err)
		// Clean up input file immediately on error
		if removeErr := os.Remove(inputPath); removeErr != nil {
			log.Printf("⚠️  Failed to delete input file after error: %v\n", removeErr)
		}
		// Clean up output file if it was partially created
		if removeErr := os.Remove(outputPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("⚠️  Failed to delete output file after error: %v\n", removeErr)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process document: %v", err),
		})
		return
	}

	log.Printf("✅ Processed successfully: %s → %s\n", file.Filename, outputFilename)

	baseFilename := file.Filename[:len(file.Filename)-len(ext)]
	downloadFilename := fmt.Sprintf("%s.xlsx", baseFilename)

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadFilename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.File(outputPath)

	go cleanupFiles(inputPath, outputPath, file.Filename)
}

// cleanupFiles removes temporary files after a delay
func cleanupFiles(inputPath, outputPath, filename string) {
	time.Sleep(30 * time.Second)

	if err := os.Remove(inputPath); err != nil {
		log.Printf("⚠️  Failed to delete input file: %v\n", err)
	}
	if err := os.Remove(outputPath); err != nil {
		log.Printf("⚠️  Failed to delete output file: %v\n", err)
	}

	log.Printf("🗑️  Cleaned up: %s\n", filename)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
