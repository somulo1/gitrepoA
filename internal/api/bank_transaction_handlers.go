package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitiateBankTransfer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bank transfer endpoint - coming soon",
	})
}
