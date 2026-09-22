package admin

import (
	"errors"
	"io"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// maxOverrideImportBytes bounds an uploaded override file. The file holds one
// small patch object per model, so a few hundred kilobytes is already generous;
// the cap only exists so a stray upload cannot be buffered without limit.
const maxOverrideImportBytes = 8 << 20

// overridePricingService returns the pricing service backing the override
// endpoints, or writes an error response and reports false.
func (h *ChannelHandler) overridePricingService(c *gin.Context) (*service.PricingService, bool) {
	if h.pricingService == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("PRICING_SERVICE_UNAVAILABLE", "pricing service is unavailable"))
		return nil, false
	}
	return h.pricingService, true
}

// respondOverrideError maps a service-layer override error onto the response
// envelope. Caller mistakes (unknown field name, negative price, bad expression)
// are 400s so the page can show the reason next to the input; anything else is an
// internal failure. The distinction lives in the service because only it knows
// which errors are about the request and which are about the file system.
func respondOverrideError(c *gin.Context, err error) {
	if service.IsOverrideValidationError(err) {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", err.Error()))
		return
	}
	response.ErrorFrom(c, infraerrors.InternalServer("PRICING_OVERRIDE_FAILED", err.Error()))
}

// setOverrideRequest is the PUT body. `fields` may not be validated with a
// binding tag: the whitelist and value rules belong to the service, and a
// failure there must name the offending field.
type setOverrideRequest struct {
	Model  string         `json:"model"`
	Fields map[string]any `json:"fields"`
}

// setOverrideDefaultsRequest is the PUT /pricing/overrides/defaults body. Like
// setOverrideRequest, `fields` carries no binding tag: the whitelist and value
// rules live in the service, and a failure there must name the offending field.
type setOverrideDefaultsRequest struct {
	Fields map[string]any `json:"fields"`
}

// refreshOverrideRequest is the POST /refresh body. An empty body means
// overwrite_tuned=false, which is the safe default (hand-tuned prices survive).
type refreshOverrideRequest struct {
	OverwriteTuned bool `json:"overwrite_tuned"`
}

// ListModelPricingOverrides returns every hand-tuned entry in the override file.
//
// The page needs the metadata (_tuned/_updated_at) as well as the price fields:
// without it an operator cannot tell their own edit from a synced catalogue row.
//
// GET /api/v1/admin/channels/pricing/overrides
func (h *ChannelHandler) ListModelPricingOverrides(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	items, err := pricing.ListOverrideEntries()
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{
		"items": items,
		"count": len(items),
		"path":  pricing.OverrideFilePath(),
	})
}

// SetModelPricingOverride writes (or replaces) one hand-tuned entry.
//
// The model name travels in the body rather than the path: catalogue keys contain
// dots and slashes, which no sane path pattern survives.
//
// PUT /api/v1/admin/channels/pricing/overrides
//
//	{"model": "deepseek-flash", "fields": {"input_cost_per_token": 1.5e-7}}
func (h *ChannelHandler) SetModelPricingOverride(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	var req setOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", err.Error()))
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MISSING_PARAMETER", "model is required").
			WithMetadata(map[string]string{"param": "model"}))
		return
	}

	entry, err := pricing.SetOverrideEntry(req.Model, req.Fields)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{"entry": entry})
}

// DeleteModelPricingOverride removes one hand-tuned entry, reverting that model
// to the catalogue price. Deleting a model that has no entry is not an error.
//
// DELETE /api/v1/admin/channels/pricing/overrides?model=deepseek-flash
func (h *ChannelHandler) DeleteModelPricingOverride(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	model := strings.TrimSpace(c.Query("model"))
	if model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MISSING_PARAMETER", "model parameter is required").
			WithMetadata(map[string]string{"param": "model"}))
		return
	}

	removed, err := pricing.DeleteOverrideEntry(model)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{"removed": removed})
}

// RefreshModelPricingOverrides re-pins the override file to the newest catalogue.
//
// POST /api/v1/admin/channels/pricing/overrides/refresh
//
//	{"overwrite_tuned": false}
func (h *ChannelHandler) RefreshModelPricingOverrides(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	var req refreshOverrideRequest
	// An absent body is a valid request for the default (keep hand-tuned values),
	// so only a malformed body is rejected.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", err.Error()))
		return
	}

	result, err := pricing.RefreshOverrides(req.OverwriteTuned)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, result)
}

// ExportModelPricingOverrides downloads the override file verbatim.
//
// This deliberately bypasses response.Success: the body is a file an operator
// keeps, edits by hand and re-imports, so wrapping it in the API envelope would
// make the previous export unusable as an input.
//
// GET /api/v1/admin/channels/pricing/overrides/export
func (h *ChannelHandler) ExportModelPricingOverrides(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	body, err := pricing.ExportOverrideFile()
	if err != nil {
		respondOverrideError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="model_pricing_overrides.json"`)
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

// ImportModelPricingOverrides replaces or merges the override file with an
// uploaded JSON object. Every entry is validated before anything is written, so
// a bad file leaves the existing overrides untouched.
//
// POST /api/v1/admin/channels/pricing/overrides/import?mode=replace|merge
func (h *ChannelHandler) ImportModelPricingOverrides(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "merge" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE_MODE",
			`mode must be "replace" or "merge"`).WithMetadata(map[string]string{"param": "mode"}))
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxOverrideImportBytes+1))
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", err.Error()))
		return
	}
	if len(body) > maxOverrideImportBytes {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", "override file is too large"))
		return
	}

	count, err := pricing.ImportOverrideEntries(body, mode)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{"imported": count, "mode": mode})
}

// GetModelPricingOverrideDefaults returns the global default prices stored under
// the reserved __defaults__ key of the override file.
//
// The response shape mirrors the per-model list (a `fields` map plus the file
// path) so the page can render both with the same code. `fields` only contains
// the keys the operator actually tuned; it may be an empty object.
//
// GET /api/v1/admin/channels/pricing/overrides/defaults
func (h *ChannelHandler) GetModelPricingOverrideDefaults(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}
	fields := pricing.DefaultOverridePrices()
	if fields == nil {
		fields = map[string]float64{}
	}
	response.Success(c, gin.H{
		"fields": fields,
		"path":   pricing.OverrideFilePath(),
	})
}

// SetModelPricingOverrideDefaults writes (or replaces) the __defaults__ segment.
//
// PUT /api/v1/admin/channels/pricing/overrides/defaults
//
//	{"fields": {"image_price_1k": 0.02, "web_search_price_per_call": 0.005}}
func (h *ChannelHandler) SetModelPricingOverrideDefaults(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}

	var req setOverrideDefaultsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PRICING_OVERRIDE", err.Error()))
		return
	}

	entry, err := pricing.SetOverrideEntry(service.OverrideDefaultsKey, req.Fields)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{"entry": entry})
}

// DeleteModelPricingOverrideDefaults removes the whole __defaults__ segment,
// reverting every non-token price to its hardcoded default. Deleting a segment
// that was never tuned is not an error.
//
// DELETE /api/v1/admin/channels/pricing/overrides/defaults
func (h *ChannelHandler) DeleteModelPricingOverrideDefaults(c *gin.Context) {
	pricing, ok := h.overridePricingService(c)
	if !ok {
		return
	}
	removed, err := pricing.DeleteOverrideEntry(service.OverrideDefaultsKey)
	if err != nil {
		respondOverrideError(c, err)
		return
	}
	response.Success(c, gin.H{"removed": removed})
}
