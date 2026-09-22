package service

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
)

// exprScale converts an expression result into dollars. Expression coefficients
// are real prices per million tokens, so the raw result is a per-million amount.
const exprScale = 1_000_000

// exprDecompositionTolerance is the allowed gap between the summed per-variable
// costs and a single evaluation of the whole expression. The decomposition is
// exact for the sum-of-products shape every supported expression uses; a larger
// gap means the expression is not linear in its variables (for example a tier
// condition that tests `p` or `c` directly), and the single evaluation wins.
const exprDecompositionTolerance = 1e-9

// billingExprTokenParams maps recorded usage onto expression variables.
//
// The expression language mixes two accounting styles: `len` is the whole input
// context for tier conditions, while `p`/`c` are the billable leftovers after
// the sub-categories the expression prices by name are removed. Removing them
// only when referenced is what lets a cache-unaware expression keep billing
// those tokens at the base rate instead of dropping them.
//
// sub2api reports InputTokens as cache-miss text input plus image input, with
// cache read/creation counted separately, so total input is their sum. Audio has
// no ledger field yet and is therefore always zero.
func billingExprTokenParams(tokens UsageTokens, usedVars map[string]bool) billingexpr.TokenParams {
	totalInput := tokens.InputTokens + tokens.CacheReadTokens + tokens.CacheCreationTokens
	p := float64(totalInput)
	params := billingexpr.TokenParams{
		Len: float64(totalInput),
		C:   float64(tokens.OutputTokens),
	}

	usesCR := usedVars[billingexpr.VarCR]
	if usesCR {
		params.CR = float64(tokens.CacheReadTokens)
		p -= params.CR
	}

	// Cache creation splits into a 5-minute/generic bucket and a 1-hour bucket.
	// Each is claimed only by the variable the expression names, so an expression
	// using neither leaves those tokens inside p.
	usesCC := usedVars[billingexpr.VarCC]
	usesCC1h := usedVars[billingexpr.VarCC1h]
	if usesCC || usesCC1h {
		cc5m, cc1h := normalizeCacheCreationBreakdown(tokens)
		if cc5m == 0 && cc1h == 0 && tokens.CacheCreationTokens > 0 {
			// The upstream did not report the ephemeral breakdown; treat the
			// aggregate as the generic bucket, matching computeCacheCreationCost.
			cc5m = tokens.CacheCreationTokens
		}
		if usesCC1h {
			params.CC1h = float64(cc1h)
			p -= params.CC1h
		}
		if usesCC {
			params.CC = float64(cc5m)
			p -= params.CC
		}
	}

	if usedVars[billingexpr.VarImg] {
		params.Img = float64(min(max(tokens.ImageInputTokens, 0), max(tokens.InputTokens, 0)))
		p -= params.Img
	}

	if usedVars[billingexpr.VarImgCR] {
		params.ImgCR = float64(min(max(tokens.ImageCacheReadTokens, 0), max(tokens.CacheReadTokens, 0)))
		if usesCR {
			// p was already reduced by the whole cache-read total, which includes
			// the image cache reads. Subtracting img_cr from p again would drop
			// those tokens out of every bucket and bill them at $0. Move them into
			// their own bucket instead, so cr and img_cr stay disjoint.
			params.CR -= params.ImgCR
		} else {
			// cr is not priced, so the image cache reads are still inside p and
			// have to be carved out here.
			p -= params.ImgCR
		}
	}

	params.P = math.Max(p, 0)
	return params
}

// applyBillingExpression prices a request with a declarative billing expression.
//
// The expression is the complete pricing truth at this layer, so the per-token
// rates are ignored. The charge is still reported per line item: each variable
// the expression references is priced on its own by evaluating the expression
// with only that variable's tokens present. That is exact for the
// sum-of-products shape these expressions use, and keeps the usage ledger's cost
// breakdown meaningful instead of collapsing everything into one number.
func (s *BillingService) applyBillingExpression(
	model, expression string,
	tokens UsageTokens,
	rateMultiplier float64,
	serviceTier string,
	pricingAt time.Time,
	pricing *ModelPricing,
) (*CostBreakdown, error) {
	params := billingExprTokenParams(tokens, billingexpr.UsedVars(expression))

	total, trace, err := billingexpr.Run(expression, params, pricingAt)
	if err != nil {
		// Fail closed: an expression that cannot be evaluated must not be billed
		// at a guessed rate.
		return nil, fmt.Errorf("billing expression for model %s: %w", model, err)
	}

	breakdown := &CostBreakdown{
		BillingExprApplied: true,
		MatchedTier:        trace.MatchedTier,
	}

	// Contribution per variable: isolate it while keeping `len`, so a tier
	// condition still selects the branch the full evaluation selected.
	var sum float64
	for _, name := range billingexpr.PricedVars {
		value := exprVarValue(params, name)
		if value <= 0 {
			continue
		}
		isolated := billingexpr.TokenParams{Len: params.Len}
		setExprVar(&isolated, name, value)
		contribution, _, runErr := billingexpr.Run(expression, isolated, pricingAt)
		if runErr != nil {
			return nil, fmt.Errorf("billing expression for model %s (variable %s): %w", model, name, runErr)
		}
		cost := contribution / exprScale
		sum += cost
		accumulateExprCost(breakdown, name, cost)
	}

	// A non-linear expression makes the per-variable split meaningless. Charge
	// the single evaluation and say so, rather than reporting line items that do
	// not add up to what was charged.
	if math.Abs(sum-total/exprScale) > exprDecompositionTolerance {
		slog.Warn("billing expression is not decomposable; reporting the evaluated total as input cost",
			"model", model, "expression", expression,
			"sum_of_parts", sum, "evaluated_total", total/exprScale)
		breakdown.ImageInputCost = 0
		breakdown.OutputCost = 0
		breakdown.ImageOutputCost = 0
		breakdown.CacheCreationCost = 0
		breakdown.CacheReadCost = 0
		breakdown.InputCost = total / exprScale
	}

	// Service-tier handling matches the rate path: a model with no explicit
	// Fast/Flex multiplier still gets the generic priority/flex factor.
	if tierMultiplier := configuredServiceTierMultiplier(serviceTier, pricing); tierMultiplier != 1.0 {
		applyCostBreakdownMultiplier(breakdown, tierMultiplier)
	}

	breakdown.TotalCost = breakdown.InputCost + breakdown.ImageInputCost + breakdown.OutputCost +
		breakdown.ImageOutputCost + breakdown.CacheCreationCost + breakdown.CacheReadCost
	breakdown.ActualCost = breakdown.TotalCost * rateMultiplier

	return breakdown, nil
}

// accumulateExprCost routes one variable's cost into the matching line item.
// Audio has no dedicated ledger field, so it lands in the closest existing
// bucket; sub2api does not populate audio usage yet, so it is zero in practice.
func accumulateExprCost(breakdown *CostBreakdown, name string, cost float64) {
	switch name {
	case billingexpr.VarP, billingexpr.VarAI:
		breakdown.InputCost += cost
	case billingexpr.VarImg:
		breakdown.ImageInputCost += cost
	case billingexpr.VarC, billingexpr.VarAO:
		breakdown.OutputCost += cost
	case billingexpr.VarCR, billingexpr.VarImgCR:
		breakdown.CacheReadCost += cost
	case billingexpr.VarCC, billingexpr.VarCC1h:
		breakdown.CacheCreationCost += cost
	}
}

func exprVarValue(params billingexpr.TokenParams, name string) float64 {
	switch name {
	case billingexpr.VarP:
		return params.P
	case billingexpr.VarC:
		return params.C
	case billingexpr.VarCR:
		return params.CR
	case billingexpr.VarCC:
		return params.CC
	case billingexpr.VarCC1h:
		return params.CC1h
	case billingexpr.VarImg:
		return params.Img
	case billingexpr.VarImgCR:
		return params.ImgCR
	case billingexpr.VarAI:
		return params.AI
	case billingexpr.VarAO:
		return params.AO
	default:
		return 0
	}
}

func setExprVar(params *billingexpr.TokenParams, name string, value float64) {
	switch name {
	case billingexpr.VarP:
		params.P = value
	case billingexpr.VarC:
		params.C = value
	case billingexpr.VarCR:
		params.CR = value
	case billingexpr.VarCC:
		params.CC = value
	case billingexpr.VarCC1h:
		params.CC1h = value
	case billingexpr.VarImg:
		params.Img = value
	case billingexpr.VarImgCR:
		params.ImgCR = value
	case billingexpr.VarAI:
		params.AI = value
	case billingexpr.VarAO:
		params.AO = value
	}
}
