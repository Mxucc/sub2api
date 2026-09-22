//go:build unit

package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newOverrideHandlerRouter wires the six override endpoints onto a bare engine.
// The service is real (not a stub): the endpoints' contract is mostly about what
// the service writes to the override file, so a stub would test nothing.
func newOverrideHandlerRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = dir
	overridePath := filepath.Join(dir, "model_pricing_overrides.json")
	cfg.Pricing.OverrideFile = overridePath

	h := &ChannelHandler{pricingService: service.NewPricingService(cfg, nil)}

	router := gin.New()
	router.GET("/channels/pricing/overrides", h.ListModelPricingOverrides)
	router.PUT("/channels/pricing/overrides", h.SetModelPricingOverride)
	router.DELETE("/channels/pricing/overrides", h.DeleteModelPricingOverride)
	router.POST("/channels/pricing/overrides/refresh", h.RefreshModelPricingOverrides)
	router.GET("/channels/pricing/overrides/export", h.ExportModelPricingOverrides)
	router.POST("/channels/pricing/overrides/import", h.ImportModelPricingOverrides)
	return router, overridePath
}

func doRequest(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// envelope is the success payload shape shared by the admin API.
type overrideEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, out any) overrideEnvelope {
	t.Helper()
	var envelope overrideEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	if out != nil {
		require.NoError(t, json.Unmarshal(envelope.Data, out))
	}
	return envelope
}

// TestModelPricingOverrideEndpoints_RoundTrip walks the page's flow: tune one
// price, list it, then delete it.
func TestModelPricingOverrideEndpoints_RoundTrip(t *testing.T) {
	router, overridePath := newOverrideHandlerRouter(t)

	put := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides",
		`{"model": "deepseek-flash", "fields": {"input_cost_per_token": 1.5e-7}}`)
	require.Equal(t, http.StatusOK, put.Code, put.Body.String())

	var putBody struct {
		Entry service.OverrideEntry `json:"entry"`
	}
	decodeEnvelope(t, put, &putBody)
	require.Equal(t, "deepseek-flash", putBody.Entry.Model)
	require.True(t, putBody.Entry.Tuned)
	require.NotEmpty(t, putBody.Entry.UpdatedAt)

	list := doRequest(t, router, http.MethodGet, "/channels/pricing/overrides", "")
	require.Equal(t, http.StatusOK, list.Code)
	var listBody struct {
		Items []service.OverrideEntry `json:"items"`
		Count int                     `json:"count"`
		Path  string                  `json:"path"`
	}
	decodeEnvelope(t, list, &listBody)
	require.Equal(t, 1, listBody.Count)
	require.Len(t, listBody.Items, 1)
	require.Equal(t, "deepseek-flash", listBody.Items[0].Model)
	require.Equal(t, overridePath, listBody.Path)
	require.InDelta(t, 1.5e-7, listBody.Items[0].Fields["input_cost_per_token"], 1e-15)

	deleted := doRequest(t, router, http.MethodDelete, "/channels/pricing/overrides?model=deepseek-flash", "")
	require.Equal(t, http.StatusOK, deleted.Code)
	var deleteBody struct {
		Removed bool `json:"removed"`
	}
	decodeEnvelope(t, deleted, &deleteBody)
	require.True(t, deleteBody.Removed)

	again := doRequest(t, router, http.MethodDelete, "/channels/pricing/overrides?model=deepseek-flash", "")
	require.Equal(t, http.StatusOK, again.Code)
	decodeEnvelope(t, again, &deleteBody)
	require.False(t, deleteBody.Removed)

	missing := doRequest(t, router, http.MethodDelete, "/channels/pricing/overrides", "")
	require.Equal(t, http.StatusBadRequest, missing.Code)
}

// TestModelPricingOverrideEndpoints_InvalidInputIs400 pins the mapping from a
// caller mistake to a 400: an unknown field name must never look like a server
// failure, or the page cannot tell the operator what to fix.
func TestModelPricingOverrideEndpoints_InvalidInputIs400(t *testing.T) {
	router, overridePath := newOverrideHandlerRouter(t)

	unknown := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides",
		`{"model": "deepseek-flash", "fields": {"context_window": 1000}}`)
	require.Equal(t, http.StatusBadRequest, unknown.Code, unknown.Body.String())
	require.Contains(t, unknown.Body.String(), "context_window")

	badValue := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides",
		`{"model": "deepseek-flash", "fields": {"input_cost_per_token": -1}}`)
	require.Equal(t, http.StatusBadRequest, badValue.Code)

	badExpr := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides",
		`{"model": "deepseek-flash", "fields": {"billing_expr": "tier("}}`)
	require.Equal(t, http.StatusBadRequest, badExpr.Code)
	require.Contains(t, badExpr.Body.String(), "billing_expr")

	noModel := doRequest(t, router, http.MethodPut, "/channels/pricing/overrides",
		`{"fields": {"input_cost_per_token": 1e-7}}`)
	require.Equal(t, http.StatusBadRequest, noModel.Code)

	badMode := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/import?mode=overwrite",
		`{"gpt-5.5": {"input_cost_per_token": 1e-7}}`)
	require.Equal(t, http.StatusBadRequest, badMode.Code)

	require.NoFileExists(t, overridePath, "被拒绝的请求不该建出覆盖文件")
}

// TestModelPricingOverrideEndpoints_ExportIsAFile：导出走的是文件下载，不是 API 信封。
func TestModelPricingOverrideEndpoints_ExportIsAFile(t *testing.T) {
	router, overridePath := newOverrideHandlerRouter(t)

	empty := doRequest(t, router, http.MethodGet, "/channels/pricing/overrides/export", "")
	require.Equal(t, http.StatusOK, empty.Code)
	require.Equal(t, "application/json; charset=utf-8", empty.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="model_pricing_overrides.json"`, empty.Header().Get("Content-Disposition"))
	require.JSONEq(t, `{}`, empty.Body.String())

	require.NoError(t, os.WriteFile(overridePath, []byte(`{"gpt-5.5": {"input_cost_per_token": 1e-06}}`), 0o644))
	body := doRequest(t, router, http.MethodGet, "/channels/pricing/overrides/export", "")
	require.Equal(t, http.StatusOK, body.Code)
	require.JSONEq(t, `{"gpt-5.5": {"input_cost_per_token": 1e-06}}`, body.Body.String())
	require.NotContains(t, body.Body.String(), `"code":0`, "导出正文必须是文件本身，不套 response.Success")
}

// TestModelPricingOverrideEndpoints_ImportAndRefresh：导入两种模式与刷新（目录缺失
// 时刷新必须报错而不是静默成功）。
func TestModelPricingOverrideEndpoints_ImportAndRefresh(t *testing.T) {
	router, overridePath := newOverrideHandlerRouter(t)

	replace := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/import?mode=replace",
		`{"gpt-5.5": {"input_cost_per_token": 1e-06}, "deepseek-flash": {"output_cost_per_token": 2e-06}}`)
	require.Equal(t, http.StatusOK, replace.Code, replace.Body.String())
	var importBody struct {
		Imported int    `json:"imported"`
		Mode     string `json:"mode"`
	}
	decodeEnvelope(t, replace, &importBody)
	require.Equal(t, 2, importBody.Imported)
	require.Equal(t, "replace", importBody.Mode)

	merge := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/import?mode=merge",
		`{"qwen-max": {"input_cost_per_token": 1.2e-06}}`)
	require.Equal(t, http.StatusOK, merge.Code)
	decodeEnvelope(t, merge, &importBody)
	require.Equal(t, 1, importBody.Imported)
	require.Equal(t, "merge", importBody.Mode)

	onDisk, err := os.ReadFile(overridePath)
	require.NoError(t, err)
	var entries map[string]map[string]any
	require.NoError(t, json.Unmarshal(onDisk, &entries))
	require.Len(t, entries, 3, "merge 必须保留未提及的条目")

	// 目录文件还没下载：刷新必须失败，而不是写出一份空覆盖文件。
	refresh := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/refresh", `{"overwrite_tuned": false}`)
	require.Equal(t, http.StatusInternalServerError, refresh.Code, refresh.Body.String())

	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(overridePath), "model_pricing.json"),
		[]byte(`{"gpt-5.5": {"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05}}`), 0o644))
	ok := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/refresh", `{"overwrite_tuned": true}`)
	require.Equal(t, http.StatusOK, ok.Code, ok.Body.String())
	var refreshBody service.OverrideRefreshResult
	decodeEnvelope(t, ok, &refreshBody)
	require.Equal(t, 1, refreshBody.Updated)
	require.Equal(t, 0, refreshBody.Added)

	// 空 body 的刷新也要能工作（默认 overwrite_tuned=false）。
	emptyBody := doRequest(t, router, http.MethodPost, "/channels/pricing/overrides/refresh", "")
	require.Equal(t, http.StatusOK, emptyBody.Code, emptyBody.Body.String())
}
