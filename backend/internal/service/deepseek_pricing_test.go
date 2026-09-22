//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DeepSeek 官方峰谷口径（2026-08-23 起生效的时段）现由声明式计费表达式表达，
// 权威定义见 model_billing_expression.go 的 deepseekFlashBillingExpr /
// deepseekProBillingExpr：
//   高峰时段 = 北京时间周一至周五 09:00-12:00 与 14:00-18:00（半开区间）；
//   其余时段（含北京时间周六/周日全天）为低谷，低谷价 = 高峰价 ÷ 2。
// 已删除 deepseekPeakMultiplierAt，因此以下测试一律按「表达式计费结果」断言，
// 用显式北京时间时点覆盖时段边界与周末。
// 2026-09-15 为周二、2026-09-11/18 为周五、2026-09-12/19 为周六、2026-09-13/20 为周日。
// ---------------------------------------------------------------------------

// beijingZone 固定北京时间 = UTC+8（无夏令时）。
var beijingZone = time.FixedZone("Asia/Shanghai", 8*3600)

// beijingAt 构造北京时间 instant：表达式按 Asia/Shanghai 判定峰谷时段。
func beijingAt(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, beijingZone)
}

const (
	// 2026-09-10 官方降价后的低谷价口径（1000 输入 / 500 输出 / 1000 缓存读）：
	// flash: 1000*1.5e-7 + 500*6e-7 + 1000*3e-9 = 4.53e-4
	deepseekFlashOffPeakTotal = 1000*1.5e-7 + 500*6e-7 + 1000*3e-9
	// pro: 1000*6.6e-7 + 500*1.98e-6 + 1000*2.2e-8 = 1.672e-3
	deepseekProOffPeakTotal = 1000*6.6e-7 + 500*1.98e-6 + 1000*2.2e-8
)

func TestIsDeepSeekModel(t *testing.T) {
	deepseek := []string{
		"deepseek-flash", "deepseek-v4-flash", "deepseek-v4-pro", "deepseek-v4-flash-vision-exp",
		"deepseek-chat", "deepseek-reasoner", "deepseek-v3-2-251201",
		"deepseek-coder", "deepseek-foo", "deepseek-v4-pro-0813",
		"DEEPSEEK-V4-PRO", " deepseek-v4-flash ",
	}
	for _, m := range deepseek {
		require.True(t, isDeepSeekModel(m), "model %q should be deepseek", m)
	}

	nonDeepseek := []string{
		"gpt-5.4", "claude-sonnet-4", "deepseekcoder", // 无连字符不算 deepseek- 前缀
		"", " deepseek", // 无连字符后缀
	}
	for _, m := range nonDeepseek {
		require.False(t, isDeepSeekModel(m), "model %q should not be deepseek", m)
	}
}

// ---------------------------------------------------------------------------
// 默认价卡（Source=LiteLLM/Fallback）按官方峰谷表达式计费：高峰实收 = 低谷实收 × 2；
// 分组/渠道自定义定价不走表达式，不受时段影响。
// ---------------------------------------------------------------------------

func TestCalculateCostUnified_DeepseekDefaultCardPeakMultiplier(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}

	// 北京时间周二 2026-09-15：高峰窗口 09:00-12:00 与 14:00-18:00（半开区间），
	// 其余时段为低谷（低谷 = 高峰 ÷ 2）；周六 2026-09-19 同一时刻为低谷。
	tests := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"weekday 08:59 off-peak", beijingAt(2026, 9, 15, 8, 59), deepseekFlashOffPeakTotal},
		{"weekday 09:00 peak start", beijingAt(2026, 9, 15, 9, 0), deepseekFlashOffPeakTotal * 2},
		{"weekday 11:59 peak upper bound", beijingAt(2026, 9, 15, 11, 59), deepseekFlashOffPeakTotal * 2},
		{"weekday 12:00 peak end", beijingAt(2026, 9, 15, 12, 0), deepseekFlashOffPeakTotal},
		{"weekday 14:00 peak start", beijingAt(2026, 9, 15, 14, 0), deepseekFlashOffPeakTotal * 2},
		{"weekday 17:59 peak upper bound", beijingAt(2026, 9, 15, 17, 59), deepseekFlashOffPeakTotal * 2},
		{"weekday 18:00 peak end", beijingAt(2026, 9, 15, 18, 0), deepseekFlashOffPeakTotal},
		{"weekday 23:59 off-peak", beijingAt(2026, 9, 15, 23, 59), deepseekFlashOffPeakTotal},
		{"saturday 10:00 off-peak", beijingAt(2026, 9, 19, 10, 0), deepseekFlashOffPeakTotal},
		{"sunday 15:00 off-peak", beijingAt(2026, 9, 20, 15, 0), deepseekFlashOffPeakTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: "deepseek-v4-flash", Tokens: tokens,
				RateMultiplier: 1.0, Resolver: resolver, PricingAt: tt.at,
			})
			require.NoError(t, err)
			require.InDelta(t, tt.want, cost.TotalCost, 1e-10, "pricingAt=%v", tt.at)
		})
	}
}

func TestCalculateCostUnified_DeepseekProDefaultCardPeakMultiplier(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}

	// 任何时刻都按 Pro 价计费（官方 2026-09-22 口径：V4-Pro 是现行模型，上游不再把它
	// 路由到 V4.1-Flash）：周五 2026-09-11 高峰 ×2，周末与夜间低谷。
	tests := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"friday 10:00 peak", beijingAt(2026, 9, 11, 10, 0), deepseekProOffPeakTotal * 2},
		{"friday 15:00 peak", beijingAt(2026, 9, 11, 15, 0), deepseekProOffPeakTotal * 2},
		{"friday 12:00 off-peak", beijingAt(2026, 9, 11, 12, 0), deepseekProOffPeakTotal},
		{"friday 20:00 off-peak", beijingAt(2026, 9, 11, 20, 0), deepseekProOffPeakTotal},
		{"saturday 10:00 off-peak", beijingAt(2026, 9, 12, 10, 0), deepseekProOffPeakTotal},
		{"sunday 15:00 off-peak", beijingAt(2026, 9, 13, 15, 0), deepseekProOffPeakTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens,
				RateMultiplier: 1.0, Resolver: resolver, PricingAt: tt.at,
			})
			require.NoError(t, err)
			require.InDelta(t, tt.want, cost.TotalCost, 1e-10, "pricingAt=%v", tt.at)
		})
	}
}

func TestCalculateCostUnified_DeepseekVersionedNamePeakMultiplier(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}

	// 版本化名称（不在价格表精确条目中）同样落到 flash 价卡对应的表达式。
	tests := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"weekday 10:00 peak", beijingAt(2026, 9, 15, 10, 0), deepseekFlashOffPeakTotal * 2},
		{"weekday 20:00 off-peak", beijingAt(2026, 9, 15, 20, 0), deepseekFlashOffPeakTotal},
		{"saturday 10:00 off-peak", beijingAt(2026, 9, 19, 10, 0), deepseekFlashOffPeakTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: "deepseek-v4-flash-0731", Tokens: tokens,
				RateMultiplier: 1.0, Resolver: resolver, PricingAt: tt.at,
			})
			require.NoError(t, err)
			require.InDelta(t, tt.want, cost.TotalCost, 1e-10, "pricingAt=%v", tt.at)
		})
	}
}

func TestCalculateCostUnified_DeepseekGroupPricingNotScaledByTimeOfDay(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	inputPrice := 1e-6
	outputPrice := 2e-6
	group := &Group{
		ID: 1, Name: "ds-group", Platform: PlatformDeepseek, Status: StatusActive,
		ModelPricing: []ChannelModelPricing{{
			Models: []string{"deepseek-v4-flash"}, BillingMode: BillingModeToken,
			InputPrice: &inputPrice, OutputPrice: &outputPrice,
		}},
	}
	resolved := resolver.Resolve(context.Background(), PricingInput{Model: "deepseek-v4-flash", Group: group})
	require.Equal(t, PricingSourceGroup, resolved.Source)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	// 分组自定义价：1000*1e-6 + 500*2e-6 + 1000*3e-9（缓存读沿用官方 flash 价）
	groupTotal := 1000*1e-6 + 500*2e-6 + 1000*3e-9

	for _, pricingAt := range []time.Time{
		beijingAt(2026, 9, 15, 20, 0), // 低谷
		beijingAt(2026, 9, 15, 10, 0), // 高峰
	} {
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "deepseek-v4-flash", Group: group,
			Tokens: tokens, RateMultiplier: 1.0, Resolver: resolver, PricingAt: pricingAt,
		})
		require.NoError(t, err)
		require.InDelta(t, groupTotal, cost.TotalCost, 1e-10,
			"分组自定义定价不走声明式表达式，不应随时段变化（pricingAt=%v）", pricingAt)
	}
}

func TestCalculateCostUnified_NonDeepseekDefaultCardNotScaledByPeak(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}
	total := 1000*3e-6 + 500*15e-6 // claude-sonnet-4 fallback

	for _, pricingAt := range []time.Time{
		beijingAt(2026, 9, 15, 20, 0), // 低谷
		beijingAt(2026, 9, 15, 10, 0), // 高峰
	} {
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "claude-sonnet-4", Tokens: tokens,
			RateMultiplier: 1.0, Resolver: resolver, PricingAt: pricingAt,
		})
		require.NoError(t, err)
		require.InDelta(t, total, cost.TotalCost, 1e-10,
			"非 DeepSeek 模型不受官方峰谷时段影响（pricingAt=%v）", pricingAt)
	}
}

func TestCalculateCostUnified_DeepseekPricingAtZeroFallsBackToNow(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}
	base := CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver,
	}

	// PricingAt 零值 → 回退 timezone.Now()，与显式传入当前时刻结果一致。
	costZero, err := bs.CalculateCostUnified(base)
	require.NoError(t, err)

	costNow, err := bs.CalculateCostUnified(CostInput{
		Ctx: base.Ctx, Model: base.Model, Tokens: base.Tokens,
		RateMultiplier: base.RateMultiplier, Resolver: base.Resolver,
		PricingAt: timezone.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, costZero.TotalCost, costNow.TotalCost)

	// 时段口径：北京时间周二 2026-09-15 高峰 10:00 实收 = 低谷 20:00 实收 × 2。
	offPeak, err := bs.CalculateCostUnified(CostInput{
		Ctx: base.Ctx, Model: base.Model, Tokens: base.Tokens,
		RateMultiplier: base.RateMultiplier, Resolver: base.Resolver,
		PricingAt: beijingAt(2026, 9, 15, 20, 0),
	})
	require.NoError(t, err)
	peak, err := bs.CalculateCostUnified(CostInput{
		Ctx: base.Ctx, Model: base.Model, Tokens: base.Tokens,
		RateMultiplier: base.RateMultiplier, Resolver: base.Resolver,
		PricingAt: beijingAt(2026, 9, 15, 10, 0),
	})
	require.NoError(t, err)
	require.InDelta(t, offPeak.TotalCost*2, peak.TotalCost, 1e-12)
}

// ---------------------------------------------------------------------------
// 官方价强制覆盖（远端旧价兜底）与未知 deepseek-* flash 兜底
// ---------------------------------------------------------------------------

func TestGetModelPricing_DeepseekForcesOfficialRatesOverJSON(t *testing.T) {
	// JSON 给任意价（模拟远端旧价/占位价），deepseek-* 必须被强制覆盖为官方低谷价。
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-flash":               {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-flash":            {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-pro":              {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-flash-vision-exp": {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-chat":                {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-reasoner":            {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
	}}
	bs := NewBillingService(&config.Config{}, pricingSvc)

	tests := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		// 2026-09-10 官方降价后：deepseek-flash（V4.1-Flash 新名）与旧名
		// deepseek-v4-flash 同按 Flash 新价。
		{"deepseek-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash-vision-exp", 1.5e-7, 6e-7, 3e-9},
		// 已停服的 chat/reasoner：即使 JSON 有旧条目也按 flash 价兜底。
		{"deepseek-chat", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-reasoner", 1.5e-7, 6e-7, 3e-9},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			// 价卡字段是「低谷口径」的基准价（表达式负责峰谷时段，展示价不随时段翻转）。
			pricing, err := bs.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
			// 固定时点（2026-10-01）复核：同一时刻结果不变。
			atPricing, err := bs.getModelPricingAt(tt.model, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
			require.NoError(t, err)
			require.InDelta(t, tt.input, atPricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, atPricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, atPricing.CacheReadPricePerToken, 1e-15)
			require.True(t, bs.HasIdentifiedTokenPricing(tt.model))
		})
	}

	// pro 档（含版本化名称）：基准价在任何时刻都是 Pro 低谷价。旧「路由切换点」
	// （2026-09-14 04:00 UTC）已不存在，其两侧与任何其它时点都必须给出同一个价。
	proInstants := []struct {
		name string
		at   time.Time
	}{
		{"2026-08-01", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{"just before the retired switch", time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC)},
		{"at the retired switch instant", time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC)},
		{"2026-10-01", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{"2027-01-01", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, model := range []string{"deepseek-v4-pro", "deepseek-v4-pro-0813"} {
		for _, instant := range proInstants {
			t.Run(model+"/"+instant.name, func(t *testing.T) {
				pricing, err := bs.getModelPricingAt(model, instant.at)
				require.NoError(t, err)
				require.InDelta(t, 6.6e-7, pricing.InputPricePerToken, 1e-15)
				require.InDelta(t, 1.98e-6, pricing.OutputPricePerToken, 1e-15)
				require.InDelta(t, 2.2e-8, pricing.CacheReadPricePerToken, 1e-15)
			})
		}
	}

	// 版本化名称（不在 JSON / fallbackPrices 精确表中）：按子串归档计价。
	// flash-0731 归 flash 档，三档价与切换无关，GetModelPricing 断言稳定。
	versioned := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-v4-flash-0731", 1.5e-7, 6e-7, 3e-9},
	}
	for _, tt := range versioned {
		t.Run(tt.model, func(t *testing.T) {
			pricing, err := bs.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

func TestGetModelPricing_UnknownDeepseekMapsToFlash(t *testing.T) {
	// JSON 含 $0 占位条目（如旧 deepseek-v3-2-251201）：未知 deepseek-* 不再
	// fail-closed，统一按 flash 价兜底（1.5e-7/6e-7/3e-9），不得按 $0 计费。
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-v3-2-251201": {InputCostPerToken: 0, OutputCostPerToken: 0},
	}}
	bs := NewBillingService(&config.Config{}, pricingSvc)

	for _, m := range []string{"deepseek-v3-2-251201", "deepseek-chat", "deepseek-reasoner", "deepseek-foo"} {
		t.Run(m, func(t *testing.T) {
			pricing, err := bs.GetModelPricing(m)
			require.NoError(t, err)
			require.InDelta(t, 1.5e-7, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, 6e-7, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, 3e-9, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

// ---------------------------------------------------------------------------
// 本地兜底 JSON：无 $0 占位条目，价格表登记的官方模型价格为官方低谷价
// （峰谷时段由表达式接管，价格表只存低谷基准价）。
// ---------------------------------------------------------------------------

func TestDeepseekPricingFileMatchesOfficialRates(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)

	pricingSvc := &PricingService{}
	pricingData, err := pricingSvc.parsePricingData(data)
	require.NoError(t, err)

	_, ok := pricingData["deepseek-v3-2-251201"]
	require.False(t, ok, "deepseek-v3-2-251201（$0 占位条目）必须从价格表中移除")
	for _, discontinued := range []string{"deepseek-chat", "deepseek-reasoner"} {
		_, ok := pricingData[discontinued]
		require.False(t, ok, "%s 已停止服务，必须从价格表中移除", discontinued)
	}

	tests := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash-vision-exp", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-pro", 6.6e-7, 1.98e-6, 2.2e-8},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			entry, ok := pricingData[tt.model]
			require.True(t, ok, "model %s must exist in pricing file", tt.model)
			require.InDelta(t, tt.input, entry.InputCostPerToken, 1e-15)
			require.InDelta(t, tt.output, entry.OutputCostPerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, entry.CacheReadInputTokenCost, 1e-15)
		})
	}
}

// ---------------------------------------------------------------------------
// deepseek-flash（V4.1-Flash 新名）与旧名共用 Flash 口径；
// deepseek-v4-pro 是现行模型，任何时刻都按 Pro 口径计费。
// ---------------------------------------------------------------------------

func TestCalculateCostUnified_DeepseekFlashAndLegacyFlashShareNewRates(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}

	// 同一时刻、同一 token 下，deepseek-flash 与旧名 deepseek-v4-flash /
	// deepseek-v4-flash-vision-exp、已停服的 deepseek-chat / deepseek-reasoner
	// 共用 flash 表达式，费用必须完全相同。
	instants := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"weekday peak", beijingAt(2026, 9, 15, 10, 0), deepseekFlashOffPeakTotal * 2},
		{"weekday off-peak", beijingAt(2026, 9, 15, 20, 0), deepseekFlashOffPeakTotal},
	}
	for _, instant := range instants {
		t.Run(instant.name, func(t *testing.T) {
			for _, model := range []string{"deepseek-flash", "deepseek-v4-flash", "deepseek-v4-flash-vision-exp", "deepseek-chat", "deepseek-reasoner"} {
				cost, err := bs.CalculateCostUnified(CostInput{
					Ctx: context.Background(), Model: model, Tokens: tokens,
					RateMultiplier: 1.0, Resolver: resolver, PricingAt: instant.at,
				})
				require.NoError(t, err)
				require.InDelta(t, instant.want, cost.TotalCost, 1e-12,
					"model %s must use the shared flash expression rate", model)
			}
		})
	}
}

// TestCalculateCostUnified_DeepseekProBillsProExpressionAtEveryInstant 钉死本次改动的核心：
// deepseek-v4-pro 是现行模型（官方 2026-09-22 口径：V4-Pro 已恢复，不再被路由到
// V4.1-Flash），因此任何时刻都按内置 Pro 表达式计费 —— 高峰 = 低谷 ×2，
// 既不因旧「切换点」（2026-09-14 04:00 UTC）而改按 Flash 计费，也不使用 Flash 价。
func TestCalculateCostUnified_DeepseekProBillsProExpressionAtEveryInstant(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	proOffPeak := deepseekProOffPeakTotal
	proPeak := deepseekProOffPeakTotal * 2

	calculate := func(t *testing.T, model string, at time.Time) *CostBreakdown {
		t.Helper()
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: model, Tokens: tokens,
			RateMultiplier: 1.0, Resolver: resolver, PricingAt: at,
		})
		require.NoError(t, err)
		return cost
	}

	// 周二 2026-09-22 北京 10:00 = 高峰；周六 2026-09-26 北京 10:00 = 低谷。
	// 两个时点都在旧「切换点」之后：若路由规则还在，前者会被算成 flash 高峰价。
	for _, model := range []string{"deepseek-v4-pro", "deepseek-v4-pro-0813"} {
		require.InDelta(t, proPeak, calculate(t, model, beijingAt(2026, 9, 22, 10, 0)).TotalCost, 1e-10,
			"%s 高峰必须 = Pro 低谷总额 ×2（不得按 Flash 计费）", model)
		require.InDelta(t, proOffPeak, calculate(t, model, beijingAt(2026, 9, 26, 10, 0)).TotalCost, 1e-10,
			"%s 周六 10:00 北京必须是低谷，且按 Pro 低谷价", model)
		// 旧切换点的两侧同为 Pro 价：pro→Flash 路由规则已经整条删除。
		require.InDelta(t, proPeak, calculate(t, model, time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC)).TotalCost, 1e-10,
			"%s 在旧切换点前一刻仍按 Pro 高峰价", model)
		require.InDelta(t, proOffPeak, calculate(t, model, time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC)).TotalCost, 1e-10,
			"%s 在旧切换点整必须还是 Pro 价（北京 12:00 低谷），不得变成 Flash 价", model)
	}

	// pro 与 flash 不得再共用一条曲线：同一时刻 pro 的实收显著高于 flash。
	flashPeak := calculate(t, "deepseek-flash", beijingAt(2026, 9, 22, 10, 0)).TotalCost
	require.InDelta(t, deepseekFlashOffPeakTotal*2, flashPeak, 1e-10)
	require.Greater(t, proPeak, flashPeak)
}

// ---------------------------------------------------------------------------
// 表达式口径回归：低谷价 = 高峰价 ÷ 2（含缓存创建分项），周末全天低谷。
// ---------------------------------------------------------------------------

func TestCalculateCostUnified_DeepseekOffPeakIsHalfOfPeak(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	// 混合 token：输入 / 缓存读 / 缓存创建 / 输出四类都非零，覆盖表达式的
	// p / cr / c 分项；缓存创建由表达式的 + cc * 0 + cc1h * 0 显式摘出，按 0 计价。
	tokens := UsageTokens{InputTokens: 1234, OutputTokens: 567, CacheReadTokens: 890, CacheCreationTokens: 321}

	calculate := func(t *testing.T, model string, at time.Time) *CostBreakdown {
		t.Helper()
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: model, Tokens: tokens,
			RateMultiplier: 1.0, Resolver: resolver, PricingAt: at,
		})
		require.NoError(t, err)
		return cost
	}

	// 北京时间周二 2026-09-15 10:00 = 高峰，20:00 = 低谷。
	flashPeak := calculate(t, "deepseek-flash", beijingAt(2026, 9, 15, 10, 0))
	flashOffPeak := calculate(t, "deepseek-flash", beijingAt(2026, 9, 15, 20, 0))
	require.InDelta(t, flashOffPeak.TotalCost*2, flashPeak.TotalCost, 1e-12,
		"flash 高峰实收必须等于低谷实收 × 2")

	// 周末（北京时间周六 2026-09-19 10:00）为低谷：与工作日低谷同价。
	flashSaturday := calculate(t, "deepseek-flash", beijingAt(2026, 9, 19, 10, 0))
	require.InDelta(t, flashOffPeak.TotalCost, flashSaturday.TotalCost, 1e-12,
		"北京时间周六 10:00 必须是低谷价")

	// pro 档同口径：旧「切换点」之后的时点同样按 Pro 高峰/低谷计费。
	proPeak := calculate(t, "deepseek-v4-pro", beijingAt(2026, 9, 22, 10, 0))
	proOffPeak := calculate(t, "deepseek-v4-pro", beijingAt(2026, 9, 22, 20, 0))
	require.InDelta(t, proOffPeak.TotalCost*2, proPeak.TotalCost, 1e-12,
		"pro 高峰实收必须等于低谷实收 × 2")
}

// TestBuiltinDeepSeekExpressionIsRenderable 钉死内建 DeepSeek 表达式可校验、可展示：
// 每个表达式都能通过 Validate（编译 + 冒烟求值），并能被 ParseTiers 解析成
// 两个时段档（peak / off_peak），且低谷系数恰为高峰系数的一半。
func TestBuiltinDeepSeekExpressionIsRenderable(t *testing.T) {
	require.NotEmpty(t, builtinModelBillingExpr)

	for key, expression := range builtinModelBillingExpr {
		t.Run(key, func(t *testing.T) {
			require.NoError(t, billingexpr.Validate(expression), "expression must pass validation")

			parsed, err := billingexpr.ParseTiers(expression)
			require.NoError(t, err)
			require.True(t, parsed.Recognized, "expression must use the canonical tier shape")
			require.True(t, parsed.TimeDependent, "expression must depend on the clock")
			require.Len(t, parsed.Tiers, 2)

			peak := parsed.Tiers[0]
			offPeak := parsed.Tiers[1]
			require.Equal(t, "peak", peak.Name)
			require.Equal(t, "off_peak", offPeak.Name)
			require.NotEmpty(t, peak.TimeWindows, "peak tier must expose its clock window")
			// else 分支没有时段条件，因此只有 peak 档带时区标签。
			require.Equal(t, deepseekTimezone, peak.Timezone)

			require.Equal(t, peak.Variables, offPeak.Variables)
			require.NotEmpty(t, peak.Coefficients)
			for name, coefficient := range peak.Coefficients {
				require.InDelta(t, coefficient/2, offPeak.Coefficients[name], 1e-15,
					"off-peak coefficient for %s must be half of the peak coefficient", name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 绝对金额回归：期望值按计费口径手写（token 数 × 该档低谷单价 × 峰谷倍率），
// 不从表达式反推。表达式与文档口径一旦分叉，以下测试即失败。
// ---------------------------------------------------------------------------

// TestCalculateCostUnified_DeepseekMatchesLegacyAbsoluteAmounts 用一组「四类 token
// 全部非零」的用法覆盖 flash / pro 两档价卡在高峰、低谷与周末的实收绝对金额，
// 并逐个分项核对 InputCost / OutputCost / CacheReadCost / CacheCreationCost。
//
// 缓存创建 321 个 token 是重点：价卡 cache_creation_input_token_cost = 0（缓存写入
// 免费），但表达式若不显式引用 cc / cc1h，这些 token 会留在 p 里按输入价收费。
func TestCalculateCostUnified_DeepseekMatchesLegacyAbsoluteAmounts(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	// 输入 / 输出 / 缓存读 / 缓存创建四类都非零。
	tokens := UsageTokens{InputTokens: 1234, OutputTokens: 567, CacheReadTokens: 890, CacheCreationTokens: 321}

	// 官方价卡登记的低谷单价（USD / token，峰谷时段由表达式接管）。
	const (
		deepseekFlashInput     = 0.15e-6
		deepseekFlashOutput    = 0.60e-6
		deepseekFlashCacheRead = 0.003e-6
		deepseekProInput       = 0.66e-6
		deepseekProOutput      = 1.98e-6
		deepseekProCacheRead   = 0.022e-6
	)

	tests := []struct {
		name                     string
		model                    string
		at                       time.Time
		multiplier               float64 // 峰谷倍率：高峰 2.0，低谷 1.0
		input, output, cacheRead float64 // 该档低谷单价
	}{
		{
			"flash peak", "deepseek-flash", beijingAt(2026, 9, 15, 10, 0), 2.0,
			deepseekFlashInput, deepseekFlashOutput, deepseekFlashCacheRead,
		},
		{
			"flash off-peak", "deepseek-flash", beijingAt(2026, 9, 15, 20, 0), 1.0,
			deepseekFlashInput, deepseekFlashOutput, deepseekFlashCacheRead,
		},
		{
			// 北京时间周六 2026-09-19 全天低谷。
			"flash weekend is off-peak", "deepseek-flash", beijingAt(2026, 9, 19, 10, 0), 1.0,
			deepseekFlashInput, deepseekFlashOutput, deepseekFlashCacheRead,
		},
		{
			// 旧别名（deepseek-v4-flash）与 deepseek-flash 共用同一档价。
			"legacy alias shares flash card", "deepseek-v4-flash", beijingAt(2026, 9, 15, 10, 0), 2.0,
			deepseekFlashInput, deepseekFlashOutput, deepseekFlashCacheRead,
		},
		{
			// pro 档：2026-09-22 周二北京 10:00 高峰。旧「切换点」已不存在，按官方现行
			// 口径 V4-Pro 仍是现行模型，必须收 Pro 价而不是 Flash 价。
			"pro peak", "deepseek-v4-pro", beijingAt(2026, 9, 22, 10, 0), 2.0,
			deepseekProInput, deepseekProOutput, deepseekProCacheRead,
		},
		{
			"pro off-peak", "deepseek-v4-pro", beijingAt(2026, 9, 22, 20, 0), 1.0,
			deepseekProInput, deepseekProOutput, deepseekProCacheRead,
		},
		{
			// 北京时间周六 2026-09-26 10:00 为低谷：pro 同样按 Pro 低谷价。
			"pro weekend is off-peak", "deepseek-v4-pro", beijingAt(2026, 9, 26, 10, 0), 1.0,
			deepseekProInput, deepseekProOutput, deepseekProCacheRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: tt.model, Tokens: tokens,
				RateMultiplier: 1.0, Resolver: resolver, PricingAt: tt.at,
			})
			require.NoError(t, err)

			// 旧硬编码口径：InputTokens 走低谷输入价、OutputTokens 走低谷输出价、
			// CacheReadTokens 走低谷缓存读价，三者再乘峰谷倍率；
			// CacheCreationTokens 一律 × 0（缓存写入免费）。
			wantInput := float64(tokens.InputTokens) * tt.input * tt.multiplier
			wantOutput := float64(tokens.OutputTokens) * tt.output * tt.multiplier
			wantCacheRead := float64(tokens.CacheReadTokens) * tt.cacheRead * tt.multiplier

			require.InDelta(t, wantInput, cost.InputCost, 1e-15, "InputCost")
			require.InDelta(t, wantOutput, cost.OutputCost, 1e-15, "OutputCost")
			require.InDelta(t, wantCacheRead, cost.CacheReadCost, 1e-15, "CacheReadCost")
			// 缓存写入免费：表达式必须显式引用 cc / cc1h 并按 0 计价；否则这 321 个
			// token 留在 p 内会按输入价收费（正是本次修复的语义差异）。
			require.Zero(t, cost.CacheCreationCost, "cache writes must be free")
			require.InDelta(t, wantInput+wantOutput+wantCacheRead, cost.TotalCost, 1e-15, "TotalCost")
			// RateMultiplier = 1.0，无 service tier 倍率 → 实收等于总额。
			require.InDelta(t, cost.TotalCost, cost.ActualCost, 1e-15, "ActualCost must equal TotalCost")
		})
	}
}

// TestCalculateCostUnified_DeepseekCacheWriteOnlyIsFree 从反方向钉死同一性质：
// 只有缓存创建 token 时，高峰与低谷的实收都必须为 0。表达式若不引用 cc / cc1h，
// 这 5000 个 token 会落进 p，高峰按 0.30e-6、低谷按 0.15e-6 收费。
func TestCalculateCostUnified_DeepseekCacheWriteOnlyIsFree(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{CacheCreationTokens: 5000}

	instants := []struct {
		name string
		at   time.Time
	}{
		{"peak", beijingAt(2026, 9, 15, 10, 0)},
		{"off-peak", beijingAt(2026, 9, 15, 20, 0)},
	}
	for _, instant := range instants {
		t.Run(instant.name, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: "deepseek-flash", Tokens: tokens,
				RateMultiplier: 1.0, Resolver: resolver, PricingAt: instant.at,
			})
			require.NoError(t, err)
			require.Zero(t, cost.TotalCost, "cache-write-only usage must cost nothing")
			require.Zero(t, cost.InputCost, "cache-write tokens must not fall into p and bill at the input rate")
			require.Zero(t, cost.CacheCreationCost, "cache writes are priced at zero")
		})
	}
}

// TestDescribeModelBillingExpr_DeepseekProUsesProExpressionAtEveryInstant 钉死展示层：
// deepseek-v4-pro 的 billing_expr 来自内置表（source=builtin），档位是 peak / off_peak
// 两档且系数必须是 Pro 的（高峰 1.32 / 0.044 / 3.96，低谷恰为一半），不是 flash 的
// 0.30 / 0.006 / 1.20 —— 否则价格页会按 Flash 价解释 pro 的计费。旧「切换点」两侧
// 同样一致：路由规则已删除，展示层不再随时点切换表达式。
func TestDescribeModelBillingExpr_DeepseekProUsesProExpressionAtEveryInstant(t *testing.T) {
	bs := newTestBillingService()

	instants := []struct {
		name string
		at   time.Time
	}{
		{"tuesday peak", testPricingInstant()},
		{"just before the retired switch", time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC)},
		{"at the retired switch instant", time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC)},
		{"saturday off-peak", beijingAt(2026, 9, 26, 10, 0)},
	}
	for _, instant := range instants {
		t.Run(instant.name, func(t *testing.T) {
			described := bs.DescribeModelBillingExpr("deepseek-v4-pro", instant.at)
			require.NotNil(t, described)
			require.Equal(t, deepseekProBillingExpr, described.Expression)
			require.Equal(t, "builtin", described.Source, "pro 表达式来自内置表")
			require.NotEqual(t, deepseekFlashBillingExpr, described.Expression,
				"pro 不得再按 flash 表达式计费")

			require.NotNil(t, described.Parsed)
			require.True(t, described.Parsed.Recognized)
			require.True(t, described.Parsed.TimeDependent)
			require.Len(t, described.Parsed.Tiers, 2)

			peak := described.Parsed.Tiers[0]
			offPeak := described.Parsed.Tiers[1]
			require.Equal(t, "peak", peak.Name)
			require.Equal(t, "off_peak", offPeak.Name)
			require.InDelta(t, 1.32, peak.Coefficients[billingexpr.VarP], 1e-12)
			require.InDelta(t, 0.044, peak.Coefficients[billingexpr.VarCR], 1e-12)
			require.InDelta(t, 3.96, peak.Coefficients[billingexpr.VarC], 1e-12)
			require.NotEqual(t, 0.30, peak.Coefficients[billingexpr.VarP],
				"pro 高峰输入价不得是 flash 的 0.30")
			for name, coefficient := range peak.Coefficients {
				require.InDelta(t, coefficient/2, offPeak.Coefficients[name], 1e-12,
					"off_peak 系数 %s 必须是 peak 的一半", name)
			}
		})
	}
}
