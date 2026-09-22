//go:build unit

package admin

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newOverrideDefaultsHandlerRouter wires the three __defaults__ endpoints onto a bare
// engine. Like the per-model router it uses a real service: the contract is about what
// the service writes to the override file, so a stub would test nothing.
func newOverrideDefaultsHandlerRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = dir
	overridePath := filepath.Join(dir, "model_pricing_overrides.json")
	cfg.Pricing.OverrideFile = overridePath

	h := &ChannelHandler{pricingService: service.NewPricingService(cfg, nil)}

	router := gin.New()
	router.GET("/channels/pricing/overrides/defaults", h.GetModelPricingOverrideDefaults)
	router.PUT("/channels/pricing/overrides/defaults", h.SetModelPricingOverrideDefaults)
	router.DELETE("/channels/pricing/overrides/defaults", h.DeleteModelPricingOverrideDefaults)
	return router, overridePath
}

// TestModelPricingOverrideDefaultsEndpoints_RoundTrip walks the page's flow: read the
// (initially empty) defaults, tune a couple of global prices, read them back, then clear.
func TestModelPricingOverrideDefaultsEndpoints_RoundTrip(t *testing.T) {
	router, overridePath := newOverrideDefaultsHandlerRouter(t)

	get := doRequest(t, router, http.MethodGet, "/channels/pricing/overrides/defaults", "")
	require.Equal(t, http.StatusOK, get.Code)
	var getBody struct {
		Fields map[string]float64 `json:"fields"`
		Path   string             `json:"path"`
	}
	decodeEnvelope(t, get, &getBody)
	require.Empty(t, getBody.Fields)
	require.Equal(t, overridePath, getBody.Path)

	put := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides/defaults",
		`{"fields": {"image_price_1k": 0.02, "web_search_price_per_call": 0.005}}`)
	require.Equal(t, http.StatusOK, put.Code, put.Body.String())
	var putBody struct {
		Entry service.OverrideEntry `json:"entry"`
	}
	decodeEnvelope(t, put, &putBody)
	require.Equal(t, service.OverrideDefaultsKey, putBody.Entry.Model)

	get = doRequest(t, router, http.MethodGet, "/channels/pricing/overrides/defaults", "")
	require.Equal(t, http.StatusOK, get.Code)
	decodeEnvelope(t, get, &getBody)
	require.InDelta(t, 0.02, getBody.Fields["image_price_1k"], 1e-12)
	require.InDelta(t, 0.005, getBody.Fields["web_search_price_per_call"], 1e-12)

	del := doRequest(t, router, http.MethodDelete, "/channels/pricing/overrides/defaults", "")
	require.Equal(t, http.StatusOK, del.Code)
	var delBody struct {
		Removed bool `json:"removed"`
	}
	decodeEnvelope(t, del, &delBody)
	require.True(t, delBody.Removed)

	again := doRequest(t, router, http.MethodDelete, "/channels/pricing/overrides/defaults", "")
	require.Equal(t, http.StatusOK, again.Code)
	decodeEnvelope(t, again, &delBody)
	require.False(t, delBody.Removed)
}

// TestModelPricingOverrideDefaultsEndpoints_InvalidIs400 pins the caller-mistake mapping:
// a token key or a negative price in __defaults__ must be a 400 naming the field, never a
// silent no-op and never a server error.
func TestModelPricingOverrideDefaultsEndpoints_InvalidIs400(t *testing.T) {
	router, overridePath := newOverrideDefaultsHandlerRouter(t)

	bad := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides/defaults",
		`{"fields": {"input_cost_per_token": 1e-6}}`)
	require.Equal(t, http.StatusBadRequest, bad.Code, bad.Body.String())
	require.Contains(t, bad.Body.String(), "input_cost_per_token")

	negative := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides/defaults",
		`{"fields": {"image_price_1k": -1}}`)
	require.Equal(t, http.StatusBadRequest, negative.Code)

	empty := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides/defaults",
		`{"fields": {}}`)
	require.Equal(t, http.StatusBadRequest, empty.Code)

	require.NoFileExists(t, overridePath, "被拒绝的请求不该建出覆盖文件")
}
