package service

import (
	"strings"
	"time"
)

// Declarative billing expressions for the models whose price cannot be
// expressed as a flat per-token rate.
//
// Why an expression instead of Go code: DeepSeek bills by time of day (peak and
// off-peak rates, with off-peak at half of peak) and publishes its prices in CNY.
// Encoding that as constants in Go made the price impossible to change without a
// release and let the catalog and the billed amount disagree. An expression owns
// the model's complete pricing at the default price-card layer instead, so the
// rate, the window and the tier name all live in one auditable string.
//
// Coefficients are real prices in USD per million tokens, matching the catalog
// entries for the same models, so the expression and the price table agree.
//
// DeepSeek official pricing (verified 2026-09-22): two current models,
// deepseek-flash (DeepSeek-V4.1-Flash) and deepseek-v4-pro
// (DeepSeek-V4-Pro-0813), each with its own peak/off-peak card. Peak hours are
// Monday-Friday 09:00-12:00 and 14:00-18:00 Beijing time; every other hour, plus
// weekends and Chinese public holidays, is off-peak at half the peak rate.
// The retired names deepseek-v4-flash and deepseek-v4-flash-vision-exp still
// resolve, and are served by V4.1-Flash at the Flash price.
// Source: https://api-docs.deepseek.com/zh-cn/quick_start/pricing
//
// Known gap: the expression language cannot see the Chinese public holiday
// calendar, so a holiday weekday is billed at peak rates. That matches the
// behaviour this expression replaced; a holiday() function is the follow-up.
const (
	// deepseekTimezone is written as the IANA zone rather than a fixed UTC
	// offset so the window reads exactly as DeepSeek documents it.
	deepseekTimezone = "Asia/Shanghai"

	// deepseekPeakWindow is the peak condition, shared by both price cards and
	// kept as one string so the two cannot drift apart.
	deepseekPeakWindow = `weekday("` + deepseekTimezone + `") >= 1 && weekday("` + deepseekTimezone + `") <= 5 && ` +
		`((hour("` + deepseekTimezone + `") >= 9 && hour("` + deepseekTimezone + `") < 12) || ` +
		`(hour("` + deepseekTimezone + `") >= 14 && hour("` + deepseekTimezone + `") < 18))`

	// deepseekCacheWrite is an explicit zero for cache-write tokens.
	//
	// DeepSeek's cache is automatic and its published card has
	// cache_creation_input_token_cost = 0: a cache write costs nothing, you only
	// pay the cheaper cache-hit rate for input that was cached. Writing the term
	// out is what keeps that true under the engine's token accounting: a variable
	// the expression does not mention stays inside `p` and is billed at the input
	// rate, so an omitted `cc` would silently start charging for cache writes.
	deepseekCacheWrite = ` + cc * 0 + cc1h * 0`

	// DeepSeek V4.1-Flash (= deepseek-flash). Peak: $0.30 input / $0.006 cache
	// read / $1.20 output per MTok. Off-peak is exactly half.
	deepseekFlashBillingExpr = `v1:` + deepseekPeakWindow +
		` ? tier("peak", p * 0.30 + cr * 0.006 + c * 1.20` + deepseekCacheWrite + `)` +
		` : tier("off_peak", p * 0.15 + cr * 0.003 + c * 0.60` + deepseekCacheWrite + `)`

	// DeepSeek V4-Pro (= deepseek-v4-pro). Peak: $1.32 input / $0.044 cache read
	// / $3.96 output per MTok. Off-peak is exactly half, matching the official
	// off-peak card of CNY 4.5 / 13.5 / 0.15 per MTok. V4-Pro is a current model
	// again: upstream no longer routes it to V4.1-Flash, so this card is what
	// every deepseek-v4-pro request is billed with.
	deepseekProBillingExpr = `v1:` + deepseekPeakWindow +
		` ? tier("peak", p * 1.32 + cr * 0.044 + c * 3.96` + deepseekCacheWrite + `)` +
		` : tier("off_peak", p * 0.66 + cr * 0.022 + c * 1.98` + deepseekCacheWrite + `)`
)

// builtinModelBillingExpr maps an expression key to its expression. Keys are the
// canonical model names used by the fallback price cards, so
// builtinBillingExprKey and getFallbackPricing agree on which card a model
// variant resolves to.
var builtinModelBillingExpr = map[string]string{
	"deepseek-flash":               deepseekFlashBillingExpr,
	"deepseek-v4-flash":            deepseekFlashBillingExpr,
	"deepseek-v4-flash-vision-exp": deepseekFlashBillingExpr,
	"deepseek-v4-pro":              deepseekProBillingExpr,
}

// builtinBillingExprKey resolves a model name to its expression key, mirroring
// the DeepSeek branch of getFallbackPricing: the two must stay in step so a
// variant never gets one card's rate with another card's expression.
//
// Order matters. "deepseek-v4-flash-vision-exp" contains "deepseek-v4-flash",
// and "deepseek-v4-pro" must be tested before the generic deepseek- fallback.
func builtinBillingExprKey(model string) string {
	modelLower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(modelLower, "deepseek-v4-flash-vision-exp"):
		return "deepseek-v4-flash-vision-exp"
	case strings.Contains(modelLower, "deepseek-v4-flash"):
		return "deepseek-v4-flash"
	case strings.Contains(modelLower, "deepseek-v4-pro"):
		return "deepseek-v4-pro"
	case strings.HasPrefix(modelLower, "deepseek-"):
		// deepseek-flash, the retired deepseek-chat / deepseek-reasoner, and any
		// unknown deepseek-* all bill at the Flash rate.
		return "deepseek-flash"
	default:
		return ""
	}
}

// billingExprSource* label which layer an effective expression came from, as
// reported to the admin price page and the model plaza. They are constants so
// the billing path and the display path cannot drift apart.
const (
	billingExprSourceCatalog = "catalog"
	billingExprSourceBuiltin = "builtin"
)

// builtinBillingExprForModel returns the compiled-in expression for a model, or
// "" when the built-in table has none and the model must be billed from its
// per-token rates.
func builtinBillingExprForModel(model string) string {
	key := builtinBillingExprKey(model)
	if key == "" {
		return ""
	}
	return builtinModelBillingExpr[key]
}

// catalogBillingExpr returns the expression carried by a price-catalog entry.
// It takes precedence over the built-in table so the rate can be changed from
// the catalog (or pricing.override_file) without a release.
func catalogBillingExpr(pricing *ModelPricing) string {
	if pricing == nil {
		return ""
	}
	return strings.TrimSpace(pricing.BillingExpr)
}

// catalogBillingExprFromCatalog is catalogBillingExpr for the raw catalog entry
// type, used where only the catalog data is in hand (for example deciding whether
// an entry may be kept despite carrying no token rates).
func catalogBillingExprFromCatalog(entry *LiteLLMModelPricing) string {
	if entry == nil {
		return ""
	}
	return strings.TrimSpace(entry.BillingExpr)
}

// resolveBillingExprWithSource is the single decision point for "which
// expression governs this model, and where does it come from", shared by the
// billing path and the display path so a surface can never show a different
// price rule than the one that charges the request.
//
// Precedence, highest first:
//
//  1. the catalog expression (which includes pricing.override_file, since
//     override entries are merged into the catalog) — reported as "catalog";
//  2. the built-in table (deepseek-v4-pro → the Pro peak/off-peak expression;
//     deepseek-flash and its aliases → the Flash one) — reported as "builtin".
//
// An entry that *explicitly* declares billing_expr (BillingExprExplicit, which
// includes `"billing_expr": ""`) ends the walk after step 1: the price data has
// taken the model over, so an empty declaration means "bill from this entry's
// own rates", never "fall back to the built-in table".
//
// Group and channel pricing is handled by the caller, which never reaches this
// function for those sources.
//
// pricingAt is accepted but unused: which expression governs a model no longer
// depends on the clock. The clock reaches the price only through the expression's
// own tier conditions at evaluation time.
func resolveBillingExprWithSource(model string, pricing *ModelPricing, _ time.Time) (expression, source string) {
	if fromCatalog := catalogBillingExpr(pricing); fromCatalog != "" {
		return fromCatalog, billingExprSourceCatalog
	}
	if pricing != nil && pricing.BillingExprExplicit {
		return "", ""
	}
	if fromBuiltin := builtinBillingExprForModel(model); fromBuiltin != "" {
		return fromBuiltin, billingExprSourceBuiltin
	}
	return "", ""
}

// resolveBillingExpr picks the expression that governs this request, or "" when
// the model bills from per-token rates. See resolveBillingExprWithSource for the
// precedence rule.
func resolveBillingExpr(model string, pricing *ModelPricing, pricingAt time.Time) string {
	expression, _ := resolveBillingExprWithSource(model, pricing, pricingAt)
	return expression
}
