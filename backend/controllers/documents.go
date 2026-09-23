package controllers

import (
	"net/http"

	"doc-manager/apihelpers"
	_ "doc-manager/models"
	"doc-manager/services"

	"github.com/gin-gonic/gin"
)

// AcceptDocument handles POST /api/documents
// @Summary      Upload and accept a document
// @Description  Accepts a file (CSV, PDF, etc.) with metadata, stores record in DB as 'uploaded', and starts async background processing
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Document file to upload"
// @Param        typeOfFile formData string false "MIME type or file format (e.g. pdf, csv)"
// @Param        optionalMeta formData string false "Optional metadata in JSON or plain text format"
// @Success      200 {object} apihelpers.ApiRes{data=models.Document} "Document accepted successfully"
// @Failure      400 {object} apihelpers.ApiRes "Missing file or invalid parameters"
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
