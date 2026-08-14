package admin

import (
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MediaProductHandler struct {
	service *service.MediaCatalogService
}

func NewMediaProductHandler(catalog *service.MediaCatalogService) *MediaProductHandler {
	return &MediaProductHandler{service: catalog}
}

func (h *MediaProductHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	products, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("search"), c.Query("modality"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, products, total, page, pageSize)
}

func (h *MediaProductHandler) Get(c *gin.Context) {
	id, ok := mediaProductID(c)
	if !ok {
		return
	}
	product, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, product)
}

func (h *MediaProductHandler) Create(c *gin.Context) {
	var product service.MediaCatalogProduct
	if err := c.ShouldBindJSON(&product); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	created, err := h.service.Create(c.Request.Context(), &product)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, created)
}

func (h *MediaProductHandler) Update(c *gin.Context) {
	id, ok := mediaProductID(c)
	if !ok {
		return
	}
	var product service.MediaCatalogProduct
	if err := c.ShouldBindJSON(&product); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	updated, err := h.service.Update(c.Request.Context(), id, &product)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *MediaProductHandler) Delete(c *gin.Context) {
	id, ok := mediaProductID(c)
	if !ok {
		return
	}
	if err := h.service.Disable(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Media product disabled successfully"})
}

func mediaProductID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_MEDIA_PRODUCT_ID", "invalid media product ID"))
		return 0, false
	}
	return id, true
}
