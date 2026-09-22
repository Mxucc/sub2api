//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
	"github.com/stretchr/testify/require"
)

// catalogTestConfig returns the minimal config the billing service needs. The
// expression path touches neither the config nor the price catalog, so a bare
// value keeps these tests independent of the shipped price files.
func catalogTestConfig() *config.Config { return &config.Config{} }

// beijingInstantAt returns 2026-09-15 at the given Beijing hour, a Tuesday, so a
// time-conditional expression can be evaluated at a chosen point in its window.
func beijingInstantAt(hour int) time.Time {
	return time.Date(2026, 9, 15, hour, 0, 0, 0, beijingZone)
}

// TestBillingExprTokenParams_Deduction pins the accounting that decides whether a
// token is charged once, twice or not at all.
//
// This is where a silent error hides: a bucket that is subtracted from p twice
// removes tokens from every price bucket, so the request is billed as if those
// tokens never existed. Asserting each bucket (and their sum) is what makes that
// visible.
func TestBillingExprTokenParams_Deduction(t *testing.T) {
	// InputTokens already excludes cache reads and creations in sub2api's ledger,
	// and ImageInputTokens is a subset of InputTokens.
	tokens := UsageTokens{
		InputTokens:          1000,
		ImageInputTokens:     200,
		OutputTokens:         500,
		CacheReadTokens:      300,
		CacheCreationTokens:  100,
		ImageCacheReadTokens: 120,
	}
	// Every input token must land in exactly one bucket.
	const totalInput = 1000 + 300 + 100

	for _, tc := range []struct {
		name       string
		expression string
		wantP      float64
		wantCR     float64
		wantCC     float64
		wantImg    float64
		wantImgCR  float64
	}{
		{
			name:       "no sub-category referenced: everything stays in p",
			expression: `tier("t", p * 1 + c * 1)`,
			wantP:      totalInput,
		},
		{
			name:       "cr referenced: cache reads leave p",
			expression: `tier("t", p * 1 + cr * 1 + c * 1)`,
			wantP:      totalInput - 300,
			wantCR:     300,
		},
		{
			name:       "cc referenced: cache creations leave p",
			expression: `tier("t", p * 1 + cc * 1 + c * 1)`,
			wantP:      totalInput - 100,
			wantCC:     100,
		},
		{
			name:       "cc and cc1h both referenced: the 1h bucket is split out first",
			expression: `tier("t", p * 1 + cc * 1 + cc1h * 1 + c * 1)`,
			wantP:      totalInput - 100,
			wantCC:     100,
		},
		{
			name:       "img referenced: image input leaves p",
			expression: `tier("t", p * 1 + img * 1 + c * 1)`,
			wantP:      totalInput - 200,
			wantImg:    200,
		},
		{
			// The regression this guards: p was already reduced by the whole
			// cache-read total (which includes the image cache reads), so
			// subtracting img_cr from p a second time would drop those tokens out
			// of every bucket and bill them at $0.
			name:       "cr and img_cr together: image cache reads are counted exactly once",
			expression: `tier("t", p * 1 + cr * 1 + img_cr * 1 + c * 1)`,
			wantP:      totalInput - 300,
			wantCR:     300 - 120,
			wantImgCR:  120,
		},
		{
			name:       "img_cr without cr: image cache reads leave p",
			expression: `tier("t", p * 1 + img_cr * 1 + c * 1)`,
			wantP:      totalInput - 120,
			wantImgCR:  120,
		},
		{
			name:       "explicit zero coefficient still carves the bucket out",
			expression: `tier("t", p * 1 + cc * 0 + cc1h * 0 + c * 1)`,
			wantP:      totalInput - 100,
			wantCC:     100,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params := billingExprTokenParams(tokens, billingexpr.UsedVars(tc.expression))

			require.InDelta(t, tc.wantP, params.P, 1e-9, "p")
			require.InDelta(t, tc.wantCR, params.CR, 1e-9, "cr")
			require.InDelta(t, tc.wantCC, params.CC, 1e-9, "cc")
			require.InDelta(t, tc.wantImg, params.Img, 1e-9, "img")
			require.InDelta(t, tc.wantImgCR, params.ImgCR, 1e-9, "img_cr")
			require.InDelta(t, 500, params.C, 1e-9, "outputs are never reduced")
			require.InDelta(t, totalInput, params.Len, 1e-9,
				"len is the whole input context, never reduced by the buckets")

			covered := params.P + params.CR + params.CC + params.CC1h + params.Img + params.ImgCR
			require.InDelta(t, totalInput, covered, 1e-9,
				"every input token must be accounted for by exactly one bucket")
		})
	}
}

// TestBillingExprTokenParams_NeverNegative guards the clamp: a contradictory
// ledger (image cache reads exceeding total cache reads, image input exceeding
// text input) must not produce a negative billable count, which would flip the
// sign of a cost term.
func TestBillingExprTokenParams_NeverNegative(t *testing.T) {
	tokens := UsageTokens{
		InputTokens:          10,
		CacheReadTokens:      5,
		CacheCreationTokens:  5,
		ImageInputTokens:     500,
		ImageCacheReadTokens: 500,
	}
	for _, expression := range []string{
		`tier("t", p * 1 + cr * 1 + img_cr * 1)`,
		`tier("t", p * 1 + cc * 1 + cc1h * 1 + img * 1)`,
		`tier("t", p * 1 + cr * 1 + cc * 1 + cc1h * 1 + img * 1 + img_cr * 1)`,
	} {
		params := billingExprTokenParams(tokens, billingexpr.UsedVars(expression))
		require.GreaterOrEqual(t, params.P, 0.0, expression)
		require.GreaterOrEqual(t, params.CR, 0.0, expression)
		require.GreaterOrEqual(t, params.CC, 0.0, expression)
		require.GreaterOrEqual(t, params.CC1h, 0.0, expression)
		require.GreaterOrEqual(t, params.Img, 0.0, expression)
		require.GreaterOrEqual(t, params.ImgCR, 0.0, expression)
	}
}

// TestBillingExprTokenParams_ZeroUsage covers a rejected or cache-only upstream
// call: every bucket is zero and nothing may divide by zero downstream.
func TestBillingExprTokenParams_ZeroUsage(t *testing.T) {
	params := billingExprTokenParams(UsageTokens{}, billingexpr.UsedVars(`tier("t", p * 1 + cr * 1 + c * 1)`))
	require.Zero(t, params.P)
	require.Zero(t, params.CR)
	require.Zero(t, params.Len)
}

// TestApplyBillingExpression_LineItemsSumToTheCharge pins the split that fills the
// usage ledger: for a linear expression the per-line-item costs must add up to
// what the whole expression evaluated to.
func TestApplyBillingExpression_LineItemsSumToTheCharge(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)
	tokens := UsageTokens{
		InputTokens:         1000,
		OutputTokens:        500,
		CacheReadTokens:     300,
		CacheCreationTokens: 100,
	}
	const expression = `tier("base", p * 3 + c * 15 + cr * 0.3)`

	breakdown, err := svc.applyBillingExpression(
		"some-model", expression, tokens, 1.0, "", testPricingInstant(), nil,
	)
	require.NoError(t, err)
	require.True(t, breakdown.BillingExprApplied)
	require.Equal(t, "base", breakdown.MatchedTier)

	// p = 1000 + 100 (cache creations stay in p; cr is priced separately)
	expectedInput := (1000.0 + 100.0) * 3 / 1e6
	require.InDelta(t, expectedInput, breakdown.InputCost, 1e-15)
	require.InDelta(t, 500.0*15/1e6, breakdown.OutputCost, 1e-15)
	require.InDelta(t, 300.0*0.3/1e6, breakdown.CacheReadCost, 1e-15)
	require.Zero(t, breakdown.CacheCreationCost)

	require.InDelta(t, expectedInput+500.0*15/1e6+300.0*0.3/1e6, breakdown.TotalCost, 1e-15)
	require.InDelta(t, breakdown.TotalCost, breakdown.ActualCost, 1e-15)
}

// TestApplyBillingExpression_RateMultiplier pins that the group multiplier is
// applied to the actual charge only, never to the reported unit costs.
func TestApplyBillingExpression_RateMultiplier(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)
	tokens := UsageTokens{InputTokens: 1000}

	breakdown, err := svc.applyBillingExpression(
		"some-model", `tier("base", p * 10)`, tokens, 2.5, "", testPricingInstant(), nil,
	)
	require.NoError(t, err)
	require.InDelta(t, 1000.0*10/1e6, breakdown.TotalCost, 1e-15)
	require.InDelta(t, 1000.0*10/1e6*2.5, breakdown.ActualCost, 1e-15)
}

// TestApplyBillingExpression_TimeFunctionUsesPricingAt pins that a time-conditional
// price is evaluated at the request's billing moment, not at the wall clock. This
// is what makes historical recomputation correct.
func TestApplyBillingExpression_TimeFunctionUsesPricingAt(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)
	tokens := UsageTokens{InputTokens: 1000}
	const expression = `hour("Asia/Shanghai") < 12 ? tier("morning", p * 2) : tier("evening", p * 4)`

	// 2026-09-15 02:00 UTC is 10:00 Beijing; 14:00 UTC is 22:00 Beijing.
	morning, err := svc.applyBillingExpression("m", expression, tokens, 1.0, "", beijingInstantAt(10), nil)
	require.NoError(t, err)
	evening, err := svc.applyBillingExpression("m", expression, tokens, 1.0, "", beijingInstantAt(22), nil)
	require.NoError(t, err)

	require.Equal(t, "morning", morning.MatchedTier)
	require.Equal(t, "evening", evening.MatchedTier)
	require.InDelta(t, 1000.0*2/1e6, morning.TotalCost, 1e-15)
	require.InDelta(t, 1000.0*4/1e6, evening.TotalCost, 1e-15)
}

// TestApplyBillingExpression_FailsClosedOnBadExpression is the safety property: an
// expression that cannot be evaluated must surface an error rather than charge $0.
func TestApplyBillingExpression_FailsClosedOnBadExpression(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)

	for _, expression := range []string{
		"tier(",
		`tier("t", unknown_var * 1)`,
		`tier("t", p - 1000000)`, // negative result
	} {
		breakdown, err := svc.applyBillingExpression(
			"m", expression, UsageTokens{InputTokens: 1000}, 1.0, "", testPricingInstant(), nil,
		)
		require.Error(t, err, "expression %q must not be billed", expression)
		require.Nil(t, breakdown)
	}
}

// TestApplyBillingExpression_NonLinearFallsBackToASingleTotal documents the
// decomposition contract: when the expression is not a sum of per-variable terms,
// the parts do not add up to the whole. The charge is then the single evaluation
// (never the sum of parts, which would double it), reported as one line item, and
// a warning records that the split is not meaningful.
func TestApplyBillingExpression_NonLinearFallsBackToASingleTotal(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 1000}

	// max(p, c) couples the two variables: each isolated evaluation still sees a
	// full-size argument, so the parts sum to twice the charge. Charging the sum
	// would be the bug this guards.
	const expression = `tier("coupled", max(p, c) * 2)`
	breakdown, err := svc.applyBillingExpression("m", expression, tokens, 1.0, "", testPricingInstant(), nil)
	require.NoError(t, err)

	want := 1000.0 * 2 / 1e6
	require.InDelta(t, want, breakdown.TotalCost, 1e-15,
		"the single evaluation is the charge, not the sum of per-variable parts")
	require.InDelta(t, want, breakdown.InputCost, 1e-15,
		"an undecomposable expression reports the whole charge as the input line")
	require.Zero(t, breakdown.OutputCost)
	require.InDelta(t, want, breakdown.ActualCost, 1e-15)
}

// TestCalculateCost_NonExpressionLeavesBillingExpressionFieldsEmpty is the
// mirror of applyBillingExpression: the ordinary per-token path must not claim
// expression billing or carry a tier, so the usage_logs columns stay at their
// defaults (” and false).
func TestCalculateCost_NonExpressionLeavesBillingExpressionFieldsEmpty(t *testing.T) {
	svc := NewBillingService(catalogTestConfig(), nil)

	cost, err := svc.CalculateCost("claude-sonnet-4", UsageTokens{InputTokens: 1000, OutputTokens: 500}, 1.0)
	require.NoError(t, err)
	require.False(t, cost.BillingExprApplied)
	require.Equal(t, "", cost.MatchedTier)
}
