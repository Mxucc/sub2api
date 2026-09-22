//go:build unit

package service

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// newGlobalDefaultsBilling builds a BillingService wired to a real PricingService whose
// override file is a temp file. Tests then tune __defaults__ through the real write path,
// so they cover the whole chain an operator's edit travels to the billing functions.
func newGlobalDefaultsBilling(t *testing.T) (*BillingService, *PricingService) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = dir
	cfg.Pricing.OverrideFile = filepath.Join(dir, "model_pricing_overrides.json")
	pricing := NewPricingService(cfg, nil)
	return NewBillingService(cfg, pricing), pricing
}

// TestGlobalDefaults_PerRequestPrice covers calculatePerRequestCost's three-way precedence:
// hardcoded default (0 here) < __defaults__ < an explicit resolved per-request price.
func TestGlobalDefaults_PerRequestPrice(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)
	resolver := NewModelPricingResolver(nil, billing)

	resolved := &ResolvedPricing{}
	input := CostInput{Resolved: resolved, Resolver: resolver, RequestCount: 1, RateMultiplier: 1}
	bd, err := billing.calculatePerRequestCost(resolved, input)
	require.NoError(t, err)
	require.InDelta(t, 0, bd.TotalCost, 1e-12)

	_, err = pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"per_request_price": 0.42})
	require.NoError(t, err)
	bd, err = billing.calculatePerRequestCost(resolved, input)
	require.NoError(t, err)
	require.InDelta(t, 0.42, bd.TotalCost, 1e-12, "__defaults__ must fill the zero fallback")

	// An explicitly resolved per-request price (group/channel) beats __defaults__.
	resolved2 := &ResolvedPricing{DefaultPerRequestPrice: 0.99}
	bd, err = billing.calculatePerRequestCost(resolved2, CostInput{Resolved: resolved2, Resolver: resolver, RequestCount: 1, RateMultiplier: 1})
	require.NoError(t, err)
	require.InDelta(t, 0.99, bd.TotalCost, 1e-12)
}

func TestGlobalDefaults_WebSearchPrice(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)

	bd := billing.CalculateWebSearchCost(1, nil, 1)
	require.InDelta(t, 0.01, bd.TotalCost, 1e-12, "hardcoded default")

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"web_search_price_per_call": 0.003})
	require.NoError(t, err)
	bd = billing.CalculateWebSearchCost(2, nil, 1)
	require.InDelta(t, 0.006, bd.TotalCost, 1e-12)

	group := 0.02
	bd = billing.CalculateWebSearchCost(1, &group, 1)
	require.InDelta(t, 0.02, bd.TotalCost, 1e-12, "explicit group price wins")
}

func TestGlobalDefaults_SearchPricePer1k(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)

	bd := billing.CalculateSearchCost(1000, nil, 1)
	require.InDelta(t, 5.0, bd.TotalCost, 1e-9, "hardcoded default")

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"search_price_per_1k": 1.0})
	require.NoError(t, err)
	bd = billing.CalculateSearchCost(1000, nil, 1)
	require.InDelta(t, 1.0, bd.TotalCost, 1e-9)

	group := 2.0
	bd = billing.CalculateSearchCost(1000, &group, 1)
	require.InDelta(t, 2.0, bd.TotalCost, 1e-9, "explicit group price wins")
}

func TestGlobalDefaults_AudioPrices(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)

	bd := billing.CalculateAudioCost("realtime", 2, nil, 1)
	require.InDelta(t, 0.10, bd.TotalCost, 1e-12, "hardcoded 0.05/min")

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"audio_realtime_price_per_min":      0.02,
		"audio_tts_price_per_million_chars": 10.0,
		"audio_stt_price_per_hour":          0.5,
	})
	require.NoError(t, err)

	bd = billing.CalculateAudioCost("realtime", 2, nil, 1)
	require.InDelta(t, 0.04, bd.TotalCost, 1e-12)
	bd = billing.CalculateAudioCost("tts", 1, nil, 1)
	require.InDelta(t, 10.0, bd.TotalCost, 1e-12)
	bd = billing.CalculateAudioCost("stt", 2, nil, 1)
	require.InDelta(t, 1.0, bd.TotalCost, 1e-12)

	realtime := 0.5
	bd = billing.CalculateAudioCost("realtime", 1, &audioPriceConfig{RealtimePerMin: &realtime}, 1)
	require.InDelta(t, 0.5, bd.TotalCost, 1e-12, "explicit group price wins")
}

func TestGlobalDefaults_ImagePricePerSize(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)

	bd := billing.CalculateImageCost("gemini-3-pro-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.134, bd.TotalCost, 1e-6, "hardcoded default")

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"image_price_1k": 0.10,
		"image_price_2k": 0.20,
		"image_price_4k": 0.30,
	})
	require.NoError(t, err)

	for size, want := range map[string]float64{"1K": 0.10, "2K": 0.20, "4K": 0.30} {
		bd = billing.CalculateImageCost("gemini-3-pro-image", size, 1, nil, 1)
		require.InDelta(t, want, bd.TotalCost, 1e-12, size)
	}

	// Explicit group price for 1K wins; the unset sizes still fall back to __defaults__.
	tier := 0.99
	groupConfig := &ImagePriceConfig{Price1K: &tier}
	bd = billing.CalculateImageCost("gemini-3-pro-image", "1K", 1, groupConfig, 1)
	require.InDelta(t, 0.99, bd.TotalCost, 1e-12)
	bd = billing.CalculateImageCost("gemini-3-pro-image", "4K", 1, groupConfig, 1)
	require.InDelta(t, 0.30, bd.TotalCost, 1e-12)
}

// TestGlobalDefaults_VideoPriceBeatsGrokHardcoded pins the deliberate ordering: the
// __defaults__ tune sits above getDefaultVideoPrice's model-aware hardcoded grok card,
// but below an explicit group price.
func TestGlobalDefaults_VideoPriceBeatsGrokHardcoded(t *testing.T) {
	billing, pricing := newGlobalDefaultsBilling(t)

	bd := billing.CalculateVideoCost("grok-imagine-video", "480p", 1, 10, nil, 1)
	require.InDelta(t, 0.5, bd.TotalCost, 1e-12, "grok hardcoded 0.05/s * 10s")

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"video_price_480p":  0.01,
		"video_price_720p":  0.02,
		"video_price_1080p": 0.03,
	})
	require.NoError(t, err)

	bd = billing.CalculateVideoCost("grok-imagine-video", "480p", 1, 10, nil, 1)
	require.InDelta(t, 0.1, bd.TotalCost, 1e-12, "__defaults__ beats the grok hardcoded card")
	bd = billing.CalculateVideoCost("grok-imagine-video", "1080p", 1, 10, nil, 1)
	require.InDelta(t, 0.3, bd.TotalCost, 1e-12)

	group480 := 0.5
	groupConfig := &VideoPriceConfig{Price480P: &group480}
	bd = billing.CalculateVideoCost("grok-imagine-video", "480p", 1, 10, groupConfig, 1)
	require.InDelta(t, 5.0, bd.TotalCost, 1e-12, "explicit group price wins")
}

// TestGlobalDefaults_NilPricingServiceIsSafe proves the helper is nil-safe: with no
// PricingService the billing functions fall back to their hardcoded defaults, no panic.
func TestGlobalDefaults_NilPricingServiceIsSafe(t *testing.T) {
	billing := &BillingService{}
	require.False(t, func() bool { _, ok := billing.globalDefaultPrice("per_request_price"); return ok }())
	bd := billing.CalculateImageCost("gemini-3-pro-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.134, bd.TotalCost, 1e-6)
}

// newGlobalDefaultsBillingWithCatalog builds a BillingService over a PricingService whose
// catalog really went through the load path (catalog + override merge), so
// getDefaultImagePrice reads the merged output_cost_per_image — the same value production
// sees. Tuning through pricing then travels the whole chain: file → merge → billing.
func newGlobalDefaultsBillingWithCatalog(t *testing.T, catalogJSON string) (*BillingService, *PricingService) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = dir
	cfg.Pricing.OverrideFile = filepath.Join(dir, "model_pricing_overrides.json")

	catalogPath := filepath.Join(dir, "model_pricing.json")
	require.NoError(t, os.WriteFile(catalogPath, []byte(catalogJSON), 0o644))

	pricing := NewPricingService(cfg, nil)
	require.NoError(t, pricing.loadPricingData(catalogPath))
	return NewBillingService(cfg, pricing), pricing
}

const globalDefaultsImageCatalogJSON = `{
	"tuned-image": {"litellm_provider": "openai", "mode": "image_generation",
		"output_cost_per_image": 0.05},
	"plain-image": {"litellm_provider": "openai", "mode": "image_generation",
		"output_cost_per_image": 0.09}
}`

// TestGlobalDefaults_ModelOverrideBeatsGlobalDefault pins the precedence the doc promises:
// a per-model tune of output_cost_per_image is more specific than the global
// image_price_* default, so it wins (with the 2K ×1.5 / 4K ×2 ladder applied).
func TestGlobalDefaults_ModelOverrideBeatsGlobalDefault(t *testing.T) {
	billing, pricing := newGlobalDefaultsBillingWithCatalog(t, globalDefaultsImageCatalogJSON)

	_, err := pricing.SetOverrideEntry("tuned-image", map[string]any{"output_cost_per_image": 0.20})
	require.NoError(t, err)
	_, err = pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"image_price_1k": 0.01,
		"image_price_2k": 0.02,
		"image_price_4k": 0.04,
	})
	require.NoError(t, err)

	bd := billing.CalculateImageCost("tuned-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.20, bd.TotalCost, 1e-12, "per-model tune wins over the global default")
	bd = billing.CalculateImageCost("tuned-image", "2K", 1, nil, 1)
	require.InDelta(t, 0.30, bd.TotalCost, 1e-12, "2K applies ×1.5 to the tuned base")
	bd = billing.CalculateImageCost("tuned-image", "4K", 1, nil, 1)
	require.InDelta(t, 0.40, bd.TotalCost, 1e-12, "4K applies ×2 to the tuned base")

	// An explicit group price still outranks the per-model tune.
	tier := 0.77
	bd = billing.CalculateImageCost("tuned-image", "1K", 1, &ImagePriceConfig{Price1K: &tier}, 1)
	require.InDelta(t, 0.77, bd.TotalCost, 1e-12)
}

// TestGlobalDefaults_GlobalDefaultBeatsCatalogValue is the other half of the rule: a model
// that was NOT tuned per model takes the global default even though the catalog carries a
// native output_cost_per_image. Otherwise the global default would be dead for every model
// the catalog already prices.
func TestGlobalDefaults_GlobalDefaultBeatsCatalogValue(t *testing.T) {
	billing, pricing := newGlobalDefaultsBillingWithCatalog(t, globalDefaultsImageCatalogJSON)

	// Before any tuning the catalog value applies.
	bd := billing.CalculateImageCost("plain-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.09, bd.TotalCost, 1e-12)

	_, err := pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"image_price_1k": 0.011,
		"image_price_2k": 0.022,
		"image_price_4k": 0.044,
	})
	require.NoError(t, err)

	bd = billing.CalculateImageCost("plain-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.011, bd.TotalCost, 1e-12, "global default beats the catalog's native per-image price")
	bd = billing.CalculateImageCost("plain-image", "2K", 1, nil, 1)
	require.InDelta(t, 0.022, bd.TotalCost, 1e-12)
	bd = billing.CalculateImageCost("plain-image", "4K", 1, nil, 1)
	require.InDelta(t, 0.044, bd.TotalCost, 1e-12)
}

// TestGlobalDefaults_ImageUntunedUnconfiguredKeepsLegacyBehavior: with neither a per-model
// tune nor a global default, the catalog value (or the hardcoded $0.134) still applies.
func TestGlobalDefaults_ImageUntunedUnconfiguredKeepsLegacyBehavior(t *testing.T) {
	billing, _ := newGlobalDefaultsBillingWithCatalog(t, globalDefaultsImageCatalogJSON)

	bd := billing.CalculateImageCost("plain-image", "1K", 1, nil, 1)
	require.InDelta(t, 0.09, bd.TotalCost, 1e-12, "catalog value unchanged")
	bd = billing.CalculateImageCost("plain-image", "2K", 1, nil, 1)
	require.InDelta(t, 0.135, bd.TotalCost, 1e-12, "catalog value with the 2K ladder")

	// A model the catalog does not price at all falls back to the hardcoded default.
	bd = billing.CalculateImageCost("unlisted-image", "2K", 1, nil, 1)
	require.InDelta(t, 0.201, bd.TotalCost, 1e-6)
}

// TestModelOverrideField covers the query itself: only whitelisted numeric keys of real
// model entries count, and the cache re-reads once the file changes.
func TestModelOverrideField(t *testing.T) {
	_, pricing := newGlobalDefaultsBilling(t)

	_, ok := pricing.ModelOverrideField("some-model", "output_cost_per_image")
	require.False(t, ok, "未微调时不得命中")

	// 白名单外的键根本写不进去（校验会拒绝并报出字段名）。
	_, err := pricing.SetOverrideEntry("some-model", map[string]any{
		"output_cost_per_image": 0.2,
		"context_window":        1000,
	})
	require.Error(t, err)

	_, err = pricing.SetOverrideEntry("some-model", map[string]any{"output_cost_per_image": 0.2})
	require.NoError(t, err)

	value, ok := pricing.ModelOverrideField("some-model", "output_cost_per_image")
	require.True(t, ok)
	require.InDelta(t, 0.2, value, 1e-12)
	_, ok = pricing.ModelOverrideField("some-model", "input_cost_per_token")
	require.False(t, ok, "没有被微调的字段不得命中")
	_, ok = pricing.ModelOverrideField("other-model", "output_cost_per_image")
	require.False(t, ok, "别的模型不得命中")

	// __defaults__ 不是模型。
	_, err = pricing.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"image_price_1k": 0.01})
	require.NoError(t, err)
	_, ok = pricing.ModelOverrideField(OverrideDefaultsKey, "image_price_1k")
	require.False(t, ok, "__defaults__ 不得被当成模型")

	// 手写坏值（非数值 / 负数 / null / 白名单外）不算命中；同一文件里的合法条目照常命中。
	overridePath := pricing.OverrideFilePath()
	require.NoError(t, os.WriteFile(overridePath, []byte(`{
		"bad-model": {
			"output_cost_per_image": "0.5",
			"input_cost_per_token": -1,
			"context_window": 1000,
			"output_cost_per_token": null
		},
		"good-model": {"output_cost_per_image": 0.3}
	}`), 0o644))

	_, ok = pricing.ModelOverrideField("bad-model", "output_cost_per_image")
	require.False(t, ok, "字符串不是合法价格")
	_, ok = pricing.ModelOverrideField("bad-model", "input_cost_per_token")
	require.False(t, ok, "负数不是合法价格")
	_, ok = pricing.ModelOverrideField("bad-model", "context_window")
	require.False(t, ok, "白名单外的键不算微调")
	_, ok = pricing.ModelOverrideField("bad-model", "output_cost_per_token")
	require.False(t, ok, "null 表示撤销，不算微调")
	value, ok = pricing.ModelOverrideField("good-model", "output_cost_per_image")
	require.True(t, ok, "同一文件里的合法条目仍要命中")
	require.InDelta(t, 0.3, value, 1e-12)

	// 指纹变化后能读到新值（与 defaults 共用同一个 mtime 判断）。
	require.NoError(t, os.WriteFile(overridePath, []byte(`{"good-model": {"output_cost_per_image": 0.6}}`), 0o644))
	value, ok = pricing.ModelOverrideField("good-model", "output_cost_per_image")
	require.True(t, ok)
	require.InDelta(t, 0.6, value, 1e-12, "改文件后必须拿到新值")

	// nil 安全：PricingService 与 BillingService 两侧都要挡住。
	var nilSvc *PricingService
	_, ok = nilSvc.ModelOverrideField("some-model", "output_cost_per_image")
	require.False(t, ok)
	_, ok = (&BillingService{}).modelOverridePrice("some-model", "output_cost_per_image")
	require.False(t, ok)
}

// TestOverrideCaches_ConcurrentReadsIsRaceFree drives both snapshots from many goroutines
// at once while the file is rewritten underneath: the two caches are built under the same
// mutex, so readers must never see a half-published map (run with -race).
func TestOverrideCaches_ConcurrentReadsIsRaceFree(t *testing.T) {
	_, pricing := newGlobalDefaultsBilling(t)
	require.NoError(t, os.WriteFile(pricing.OverrideFilePath(), []byte(`{
		"gpt-5.5": {"output_cost_per_image": 0.2},
		"__defaults__": {"image_price_1k": 0.01}
	}`), 0o644))

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = pricing.DefaultOverridePrice("image_price_1k")
				_, _ = pricing.ModelOverrideField("gpt-5.5", "output_cost_per_image")
			}
		}()
	}
	for i := 0; i < 20; i++ {
		require.NoError(t, os.WriteFile(pricing.OverrideFilePath(),
			[]byte(`{"__defaults__": {"image_price_1k": 0.02}}`), 0o644))
		require.NoError(t, os.WriteFile(pricing.OverrideFilePath(), []byte(`{
			"gpt-5.5": {"output_cost_per_image": 0.3},
			"__defaults__": {"image_price_1k": 0.01}
		}`), 0o644))
	}
	close(stop)
	wg.Wait()
}
