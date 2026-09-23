package controllers

import (
	"errors"
	"net/http"
	"strings"

	"doc-manager/apihelpers"
	_ "doc-manager/models"
	"doc-manager/services"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

// AcceptDocument handles POST /api/documents
// @Summary      Upload and accept a document
// @Description  Accepts a file (CSV, PDF, etc.) with metadata, stores record in DB, prevents duplicate uploads via content_hash, and starts async processing
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Document file to upload"
// @Param        typeOfFile formData string false "MIME type or file format (e.g. pdf, csv)"
// @Param        optionalMeta formData string false "Optional metadata in JSON or plain text format"
// @Success      200 {object} apihelpers.ApiRes{data=models.Document} "Document accepted successfully"
// @Failure      400 {object} apihelpers.ApiRes "Missing file or invalid parameters"
// @Failure      409 {object} apihelpers.ApiRes "Duplicate document: identical name and content"
// @Failure      500 {object} apihelpers.ApiRes "Server or database error"
// @Router       /documents [post]
func AcceptDocument(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, apihelpers.ApiRes{
			Data:    nil,
			Status:  false,
			Message: "file is required: " + err.Error(),
		})
		return
	}

	typeOfFile := c.PostForm("typeOfFile")
	optionalMeta := c.PostForm("optionalMeta")

	doc, err := services.AcceptDocument(c.Request.Context(), file, typeOfFile, optionalMeta)
	if err != nil {
		if errors.Is(err, services.ErrDuplicateDocument) {
			c.JSON(http.StatusConflict, apihelpers.ApiRes{
				Data:    nil,
				Status:  false,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, apihelpers.ApiRes{
			Data:    nil,
			Status:  false,
			Message: "failed to accept document: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apihelpers.ApiRes{
		Data:    doc,
		Status:  true,
		Message: "document accepted successfully",
	})
}

// GetDocumentEvents handles GET /api/documents/:id/events
// @Summary      Get document event history
// @Description  Queries Cassandra for all historical event snapshots and status transitions for a given document UUID
// @Tags         documents
// @Produce      json
// @Param        id path string true "Document UUID"
// @Success      200 {object} apihelpers.ApiRes{data=[]models.DocumentSnapshot} "Events retrieved successfully"
// @Failure      400 {object} apihelpers.ApiRes "Invalid document ID format"
// @Failure      500 {object} apihelpers.ApiRes "Server or database error"
// @Router       /documents/{id}/events [get]
func GetDocumentEvents(c *gin.Context) {
	docID := strings.TrimSpace(c.Param("id"))
	if docID == "" {
		c.JSON(http.StatusBadRequest, apihelpers.ApiRes{
			Data:    nil,
			Status:  false,
			Message: "document id is required",
		})
		return
	}

	if _, err := gocql.ParseUUID(docID); err != nil {
		c.JSON(http.StatusBadRequest, apihelpers.ApiRes{
			Data:    nil,
			Status:  false,
			Message: "invalid document ID: must be a valid UUID",
		})
		return
	}

	events, err := services.GetDocumentEvents(c.Request.Context(), docID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid document ID") {
			status = http.StatusBadRequest
		}
		c.JSON(status, apihelpers.ApiRes{
			Data:    nil,
			Status:  false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apihelpers.ApiRes{
		Data:    events,
		Status:  true,
		Message: "events retrieved successfully",
	})
}
