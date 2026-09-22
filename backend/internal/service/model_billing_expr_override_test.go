//go:build unit

package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// catalogWithDeclaredBillingExpr 造一份小目录，覆盖「价格数据显式声明计费」的三种形态：
//
//   - deepseek-v4-flash：非空 billing_expr —— 该模型的价与计费规则由价格数据接管；
//   - deepseek-flash：没有 billing_expr 键 —— 内置规则照旧兜底（强制官方低谷价）；
//   - deepseek-v3-2-251201：litellm_provider 是 volcengine（DeepSeek 名字走别的上游），
//     显式写 `"billing_expr": ""` —— 既不被套官方峰谷表达式，也不被强制成官方价。
const catalogWithDeclaredBillingExpr = `{
	"deepseek-v4-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 9e-07, "output_cost_per_token": 8e-06,
		"cache_read_input_token_cost": 4e-08,
		"billing_expr": "tier(\"flat\", p * 2 + c * 4)"},
	"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 9e-07, "output_cost_per_token": 8e-06,
		"cache_read_input_token_cost": 4e-08},
	"deepseek-v3-2-251201": {"litellm_provider": "volcengine", "mode": "chat",
		"input_cost_per_token": 2e-07, "output_cost_per_token": 8e-07,
		"cache_read_input_token_cost": 2e-08,
		"billing_expr": ""}
}`

// TestApplyModelSpecificPricingPolicy_DeclaredBillingExprExemptsForcedDeepSeekRates 钉死
// 「显式声明 billing_expr 就不再被强制官方价」这一层豁免，以及它的边界：没有这个键的
// DeepSeek 模型行为必须与从前完全一致。
func TestApplyModelSpecificPricingPolicy_DeclaredBillingExprExemptsForcedDeepSeekRates(t *testing.T) {
	svc := newCatalogTestService(t, catalogWithDeclaredBillingExpr)

	cases := []struct {
		name                     string
		model                    string
		explicit                 bool
		input, output, cacheRead float64
	}{
		{
			// 目录/覆盖层显式声明了表达式 → 目录价原样保留：价格仓库因此能接管 DeepSeek。
			"declared expr keeps catalog rates",
			"deepseek-v4-flash", true, 9e-07, 8e-06, 4e-08,
		},
		{
			// 没有 billing_expr 键 → 仍被强制成官方低谷价（现状不变）。
			"missing expr still forces official rates",
			"deepseek-flash", false,
			deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead,
		},
		{
			// 显式空串同样豁免：volcengine 这类别的上游不再被套官方价（与峰谷表达式）。
			"empty expr keeps catalog rates",
			"deepseek-v3-2-251201", true, 2e-07, 8e-07, 2e-08,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pricing, err := svc.getModelPricingAt(tc.model, testPricingInstant())
			require.NoError(t, err)
			require.Equal(t, tc.explicit, pricing.BillingExprExplicit)
			require.InDelta(t, tc.input, pricing.InputPricePerToken, 1e-18)
			require.InDelta(t, tc.output, pricing.OutputPricePerToken, 1e-18)
			require.InDelta(t, tc.cacheRead, pricing.CacheReadPricePerToken, 1e-18)
		})
	}

	// 显式声明的 $0 价也不许被改：豁免的是「强制价」本身，不是「数据看起来像占位」。
	explicitZero := &ModelPricing{BillingExprExplicit: true}
	survived := svc.applyModelSpecificPricingPolicyEx("deepseek-flash", explicitZero, true, testPricingInstant())
	require.Zero(t, survived.InputPricePerToken,
		"显式声明后，代码内置的官方价不得把单价改回来")

	// 只豁免这一层：分组/渠道自定义价走 forceDeepSeekRates=false 的调用路径，行为不变。
	groupPriced := &ModelPricing{
		InputPricePerToken:     5e-07,
		OutputPricePerToken:    6e-06,
		CacheReadPricePerToken: 1e-08,
		BillingExprExplicit:    true,
	}
	require.Same(t, groupPriced, svc.applyModelSpecificPricingPolicyEx("deepseek-flash", groupPriced, false, testPricingInstant()),
		"分组/渠道自定义价必须原样返回，不被任何 DeepSeek 规则改写")
}

// TestResolveBillingExpr_DeclaredEmptyExpressionSuppressesBuiltin 钉死空串的语义：
// `"billing_expr": ""` 是「该模型的价由价格数据定义」，因此既不走内置表达式兜底
// （DeepSeek 的官方峰谷表达式就是内置规则），也不产生展示用的表达式，直接按单价计费。
func TestResolveBillingExpr_DeclaredEmptyExpressionSuppressesBuiltin(t *testing.T) {
	svc := newCatalogTestService(t, catalogWithDeclaredBillingExpr)

	declared, err := svc.getModelPricingAt("deepseek-v4-flash", testPricingInstant())
	require.NoError(t, err)
	require.Equal(t, `tier("flat", p * 2 + c * 4)`,
		resolveBillingExpr("deepseek-v4-flash", declared, testPricingInstant()),
		"显式非空表达式优先于内置表")

	missing, err := svc.getModelPricingAt("deepseek-flash", testPricingInstant())
	require.NoError(t, err)
	require.Equal(t, deepseekFlashBillingExpr, resolveBillingExpr("deepseek-flash", missing, testPricingInstant()),
		"没有 billing_expr 键时内置表达式照旧兜底")

	empty, err := svc.getModelPricingAt("deepseek-v3-2-251201", testPricingInstant())
	require.NoError(t, err)
	require.True(t, empty.BillingExprExplicit)
	require.Empty(t, resolveBillingExpr("deepseek-v3-2-251201", empty, testPricingInstant()),
		"显式空串必须压掉内置表达式兜底，否则 volcengine 的 deepseek-* 仍被套官方峰谷价")
	require.Nil(t, svc.DescribeModelBillingExpr("deepseek-v3-2-251201", testPricingInstant()),
		"展示层不能凭空报出一个实际没在用的内置表达式")

	// 计费口径：按目录单价收，且不标成「表达式计费」。
	resolver := NewModelPricingResolver(nil, svc)
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	cost, err := svc.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v3-2-251201", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver, PricingAt: testPricingInstant(),
	})
	require.NoError(t, err)
	require.False(t, cost.BillingExprApplied, "没有表达式就不该走表达式计费")
	require.InDelta(t, 1000*2e-07+500*8e-07+1000*2e-08, cost.TotalCost, 1e-15)
}

// TestDeepseekProExpressionComesFromCatalogWhenDeclared 钉死 pro 的表达式来源语义：
// 目录/覆盖层显式声明 billing_expr 时由价格数据接管（source=catalog，计费按该表达式），
// 没有声明时用内置 Pro 峰谷表达式（source=builtin）。没有任何「上游路由」硬规则：
// deepseek-v4-pro 是现行模型，任何时刻都走同一条解析链与同一份 Pro 价卡。
func TestDeepseekProExpressionComesFromCatalogWhenDeclared(t *testing.T) {
	const proExpr = `tier("custom_pro", p * 2 + c * 4)`
	catalog := `{"deepseek-v4-pro": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 9e-07, "output_cost_per_token": 8e-06,
		"cache_read_input_token_cost": 4e-08,
		"billing_expr": "tier(\"custom_pro\", p * 2 + c * 4)"}}`
	svc := newCatalogTestService(t, catalog)

	// 目录显式声明 → 目录表达式生效，来源 catalog；基准价原样保留（显式声明豁免强制价）。
	// 旧「路由切换点」（2026-09-14 04:00 UTC）两侧不再有任何差异。
	for _, at := range []time.Time{
		time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC),
		time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC),
		beijingAt(2026, 9, 22, 10, 0),
	} {
		described := svc.DescribeModelBillingExpr("deepseek-v4-pro", at)
		require.NotNil(t, described, "pricingAt=%v", at)
		require.Equal(t, proExpr, described.Expression, "目录表达式优先于内置表（pricingAt=%v）", at)
		require.Equal(t, "catalog", described.Source)

		pricing, err := svc.getModelPricingAt("deepseek-v4-pro", at)
		require.NoError(t, err)
		require.True(t, pricing.BillingExprExplicit)
		require.InDelta(t, 9e-07, pricing.InputPricePerToken, 1e-18)
		require.InDelta(t, 8e-06, pricing.OutputPricePerToken, 1e-18)
		require.InDelta(t, 4e-08, pricing.CacheReadPricePerToken, 1e-18)
		require.Equal(t, proExpr, resolveBillingExpr("deepseek-v4-pro", pricing, at))
	}

	// 计费与展示同源：自定义 pro 表达式 (2000 * 2 + 500 * 4) USD/MTok —— 表达式不引用 cr，
	// 那 1000 个缓存读 token 留在 p 里，因此 p = 1000 + 1000；两者都与时刻无关。
	resolver := NewModelPricingResolver(nil, svc)
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	for _, at := range []time.Time{
		time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC),
		beijingAt(2026, 9, 22, 10, 0),
	} {
		cost, err := svc.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens,
			RateMultiplier: 1.0, Resolver: resolver, PricingAt: at,
		})
		require.NoError(t, err)
		require.True(t, cost.BillingExprApplied)
		require.InDelta(t, (2000*2+500*4)/1e6, cost.TotalCost, 1e-15)
	}

	// 目录没有这个模型的表达式时：内置 Pro 表达式生效，来源 builtin，基准价被强制归一到
	// Pro 低谷价 —— 全部与时刻无关。
	builtinSvc := newCatalogTestService(t, `{"deepseek-v4-pro": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 9e-07, "output_cost_per_token": 8e-06,
		"cache_read_input_token_cost": 4e-08}}`)
	peakInstant := beijingAt(2026, 9, 22, 10, 0)
	described := builtinSvc.DescribeModelBillingExpr("deepseek-v4-pro", peakInstant)
	require.NotNil(t, described)
	require.Equal(t, deepseekProBillingExpr, described.Expression)
	require.Equal(t, "builtin", described.Source)

	pricing, err := builtinSvc.getModelPricingAt("deepseek-v4-pro", peakInstant)
	require.NoError(t, err)
	require.InDelta(t, deepseekProOffPeakInputPrice, pricing.InputPricePerToken, 1e-18)
	require.InDelta(t, deepseekProOffPeakOutputPrice, pricing.OutputPricePerToken, 1e-18)
	require.InDelta(t, deepseekProOffPeakCacheRead, pricing.CacheReadPricePerToken, 1e-18)

	builtinCost, err := builtinSvc.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: NewModelPricingResolver(nil, builtinSvc),
		PricingAt: peakInstant,
	})
	require.NoError(t, err)
	require.InDelta(t, deepseekProOffPeakTotal*2, builtinCost.TotalCost, 1e-12,
		"北京时间周二 10:00 高峰：pro 实收必须是 Pro 低谷总额 ×2")
}

// TestOverrideFileBillingExpr_TakesOverPricingAndValidates 覆盖 pricing.override_file 这
// 条路径：JSON 层浅合并 → parsePricingData 带上「显式声明」标记 → 不再被官方价改写、
// 表达式成为计费依据；同时钉死管理端写入校验（字符串 / 允许空串 / 编译错误原文）。
func TestOverrideFileBillingExpr_TakesOverPricingAndValidates(t *testing.T) {
	svc, _, path := newOverrideTestService(t)
	const expr = `tier("flat", p * 3 + c * 6)`

	require.NoError(t, os.WriteFile(path, []byte(`{
		"deepseek-v4-flash": {"billing_expr": "tier(\"flat\", p * 3 + c * 6)", "input_cost_per_token": 1e-06}
	}`), 0o644))

	catalog := `{"deepseek-v4-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 9e-07, "output_cost_per_token": 8e-06,
		"cache_read_input_token_cost": 4e-08}}`
	data, err := svc.parsePricingData([]byte(catalog))
	require.NoError(t, err)

	entry := data["deepseek-v4-flash"]
	require.True(t, entry.BillingExprExplicit, "覆盖文件里的 billing_expr 必须带上「显式声明」标记")
	require.Equal(t, expr, entry.BillingExpr)
	require.InDelta(t, 1e-06, entry.InputCostPerToken, 1e-15, "覆盖层的单价按浅合并生效")

	svc.pricingData = data
	bs := NewBillingService(&config.Config{}, svc)
	pricing, err := bs.getModelPricingAt("deepseek-v4-flash", testPricingInstant())
	require.NoError(t, err)
	require.InDelta(t, 1e-06, pricing.InputPricePerToken, 1e-15,
		"显式声明后官方价不得再盖掉覆盖文件里的单价")
	require.InDelta(t, 8e-06, pricing.OutputPricePerToken, 1e-15)
	require.Equal(t, expr, resolveBillingExpr("deepseek-v4-flash", pricing, testPricingInstant()))

	resolver := NewModelPricingResolver(nil, bs)
	cost, err := bs.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash",
		Tokens:         UsageTokens{InputTokens: 1000, OutputTokens: 500},
		RateMultiplier: 1.0, Resolver: resolver, PricingAt: testPricingInstant(),
	})
	require.NoError(t, err)
	require.True(t, cost.BillingExprApplied)
	require.InDelta(t, (1000*3+500*6)/1e6, cost.TotalCost, 1e-15)

	// 管理端写入：空串合法，且语义是「用价格数据里的单价」。
	updated, err := svc.SetOverrideEntry("deepseek-v4-flash", map[string]any{"billing_expr": ""})
	require.NoError(t, err)
	require.Equal(t, "", updated.Fields[overrideExprKey])

	// 非法表达式：400 语义 + 编译错误原文（管理员要照着它改）。
	_, err = svc.SetOverrideEntry("deepseek-v4-flash", map[string]any{"billing_expr": "tier("})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err), "非法表达式是调用方的错误，HTTP 层必须回 400")
	require.Contains(t, err.Error(), "billing_expr")
	require.Contains(t, err.Error(), "expr compile error", "编译错误原文必须回给管理员")

	// 只接受字符串：数字与 null 都拒绝（null 在浅合并里是「删键」，与用途相反）。
	_, err = svc.SetOverrideEntry("deepseek-v4-flash", map[string]any{"billing_expr": 42.0})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err))
	require.Contains(t, err.Error(), "must be a string")

	_, err = svc.SetOverrideEntry("deepseek-v4-flash", map[string]any{"billing_expr": nil})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err))
	require.Contains(t, err.Error(), "must be a string")
}

// TestCatalogOverridePatch_ExcludesBillingExpr：「获取最新价格 / RefreshOverrides」只从
// 目录挑单价字段回写覆盖文件，billing_expr 必须排除，否则表达式会被固化成快照。
func TestCatalogOverridePatch_ExcludesBillingExpr(t *testing.T) {
	patch := catalogOverridePatch(json.RawMessage(`{
		"input_cost_per_token": 1e-06,
		"billing_expr": "tier(\"flat\", p * 3 + c * 6)",
		"context_window": 400000}`))

	require.InDelta(t, 1e-06, patch["input_cost_per_token"], 1e-15)
	_, exprCopied := patch[overrideExprKey]
	require.False(t, exprCopied, "表达式不得被抄进覆盖文件")
	_, contextCopied := patch["context_window"]
	require.False(t, contextCopied, "非白名单字段不得被抄进覆盖文件")
}
