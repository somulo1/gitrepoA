package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadLearningImage handles image uploads for learning content
func UploadLearningImage(c *gin.Context) {
	// Debug logging
	fmt.Printf("🔍 Upload request received - Method: %s, Content-Type: %s\n",
		c.Request.Method, c.Request.Header.Get("Content-Type"))
	fmt.Printf("🔍 Form data keys: %v\n", c.Request.Form)

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		fmt.Printf("❌ Error getting form file: %v\n", err)
		fmt.Printf("❌ Available form fields: %v\n", c.Request.Form)
		if c.Request.MultipartForm != nil {
			fmt.Printf("❌ Available file fields: %v\n", c.Request.MultipartForm.File)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No image file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	fmt.Printf("✅ File received - Name: %s, Size: %d, Type: %s\n",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Validate file type
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid file type. Only JPEG, PNG, WebP, and GIF images are allowed",
		})
		return
	}

	// Validate file size (10MB max for learning content)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Image too large. Maximum size is 10MB",
		})
		return
	}

	// Create uploads directory
	uploadDir := "./uploads/learning/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create upload directory",
		})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		// Determine extension from content type
		switch contentType {
		case "image/jpeg", "image/jpg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		}
	}

	filename := fmt.Sprintf("learning_img_%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save image: " + err.Error(),
		})
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("/uploads/learning/images/%s", filename)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Image uploaded successfully",
		"image_url": fileURL,
		"filename":  filename,
		"size":      header.Size,
	})
}

// UploadLearningVideo handles video uploads for learning content
func UploadLearningVideo(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No video file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file type
	allowedTypes := map[string]bool{
		"video/mp4":       true,
		"video/mpeg":      true,
		"video/quicktime": true,
		"video/x-msvideo": true, // .avi
		"video/webm":      true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid file type. Only MP4, MPEG, QuickTime, AVI, and WebM videos are allowed",
		})
		return
	}

	// Validate file size (100MB max for videos)
	if header.Size > 100*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Video too large. Maximum size is 100MB",
		})
		return
	}

	// Create uploads directory
	uploadDir := "./uploads/learning/videos"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create upload directory",
		})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		// Determine extension from content type
		switch contentType {
		case "video/mp4":
			ext = ".mp4"
		case "video/mpeg":
			ext = ".mpeg"
		case "video/quicktime":
			ext = ".mov"
		case "video/x-msvideo":
			ext = ".avi"
		case "video/webm":
			ext = ".webm"
		}
	}

	filename := fmt.Sprintf("learning_vid_%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save video: " + err.Error(),
		})
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("/uploads/learning/videos/%s", filename)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Video uploaded successfully",
		"video_url": fileURL,
		"filename":  filename,
		"size":      header.Size,
	})
}

// UploadLearningDocument handles document uploads for learning content
func UploadLearningDocument(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No document file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file type
	allowedTypes := map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.ms-powerpoint":                                             true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"text/plain": true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid file type. Only PDF, Word, Excel, PowerPoint, and text documents are allowed",
		})
		return
	}

	// Validate file size (20MB max for documents)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Document too large. Maximum size is 20MB",
		})
		return
	}

	// Create uploads directory
	uploadDir := "./uploads/learning/documents"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create upload directory",
		})
		return
	}

	// Generate unique filename preserving original extension
	ext := filepath.Ext(header.Filename)
	baseName := strings.TrimSuffix(header.Filename, ext)
	filename := fmt.Sprintf("learning_doc_%s_%s_%d%s", uuid.New().String(), baseName, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save document: " + err.Error(),
		})
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("/uploads/learning/documents/%s", filename)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Document uploaded successfully",
		"document_url":  fileURL,
		"filename":      filename,
		"original_name": header.Filename,
		"size":          header.Size,
		"type":          contentType,
	})
}

// ValidateVideoURL validates and processes video URLs
func ValidateVideoURL(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req struct {
		VideoURL string `json:"video_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	videoURL := strings.TrimSpace(req.VideoURL)
	if videoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Video URL is required",
		})
		return
	}

	// Validate and process different video URL types
	videoInfo := processVideoURL(videoURL)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Video URL validated successfully",
		"video_info": videoInfo,
	})
}

// processVideoURL processes different types of video URLs
func processVideoURL(videoURL string) map[string]interface{} {
	result := map[string]interface{}{
		"original_url": videoURL,
		"type":         "unknown",
		"playable":     false,
		"embed_url":    "",
		"thumbnail":    "",
	}

	// YouTube URL processing
	if strings.Contains(videoURL, "youtube.com") || strings.Contains(videoURL, "youtu.be") {
		videoID := extractYouTubeVideoID(videoURL)
		if videoID != "" {
			result["type"] = "youtube"
			result["video_id"] = videoID
			result["embed_url"] = fmt.Sprintf("https://www.youtube.com/embed/%s", videoID)
			result["thumbnail"] = fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)
			result["playable"] = true
		}
		return result
	}

	// Vimeo URL processing
	if strings.Contains(videoURL, "vimeo.com") {
		vimeoID := extractVimeoVideoID(videoURL)
		if vimeoID != "" {
			result["type"] = "vimeo"
			result["video_id"] = vimeoID
			result["embed_url"] = fmt.Sprintf("https://player.vimeo.com/video/%s", vimeoID)
			result["playable"] = true
		}
		return result
	}

	// Direct video file URLs
	if isDirectVideoURL(videoURL) {
		result["type"] = "direct"
		result["embed_url"] = videoURL
		result["playable"] = true
		return result
	}

	// Streaming URLs (HLS, DASH)
	if isStreamingURL(videoURL) {
		result["type"] = "streaming"
		result["embed_url"] = videoURL
		result["playable"] = true
		return result
	}

	// Default: try to play as direct URL
	result["type"] = "generic"
	result["embed_url"] = videoURL
	result["playable"] = true

	return result
}

// extractYouTubeVideoID extracts video ID from YouTube URLs
func extractYouTubeVideoID(url string) string {
	patterns := []string{
		`(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// extractVimeoVideoID extracts video ID from Vimeo URLs
func extractVimeoVideoID(url string) string {
	re := regexp.MustCompile(`vimeo\.com\/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// isDirectVideoURL checks if URL is a direct video file
func isDirectVideoURL(url string) bool {
	videoExtensions := []string{".mp4", ".webm", ".ogg", ".avi", ".mov", ".wmv", ".flv", ".mkv"}
	lowerURL := strings.ToLower(url)
	for _, ext := range videoExtensions {
		if strings.Contains(lowerURL, ext) {
			return true
		}
	}
	return false
}

// isStreamingURL checks if URL is a streaming video
func isStreamingURL(url string) bool {
	streamingExtensions := []string{".m3u8", ".mpd"}
	lowerURL := strings.ToLower(url)
	for _, ext := range streamingExtensions {
		if strings.Contains(lowerURL, ext) {
			return true
		}
	}
	return false
}
