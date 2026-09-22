// Package billingexpr implements the declarative billing expression engine.
//
// A single expression string fully describes how a model is billed at the
// default price-card layer: token rates, tier conditions, cache/image/audio
// differentiation and time-of-day pricing all live in one evaluated expression.
// The grammar and semantics follow new-api's pkg/billingexpr so expressions are
// portable between the two projects.
//
// Coefficients are real prices in USD per million tokens, so `p * 0.30` means
// $0.30 per 1M prompt tokens. Callers convert the result to a dollar amount by
// dividing by 1e6.
//
// This is the v1 subset: it deliberately omits request probes (param/header),
// per-request fixed pricing, task-usage facts and request-rule tracing.
package billingexpr

import (
	"crypto/sha256"
	"fmt"
)

// Version prefix handling. An expression may carry a `v1:` tag; without a tag
// the expression is v1. The version selects the compile environment and is
// reserved for future grammar evolution.
const (
	DefaultExprVersion = 1
	versionPrefix      = "v1:"
)

// Billing variables. Names match new-api so expressions can be shared.
const (
	VarP     = "p"
	VarC     = "c"
	VarLen   = "len"
	VarCR    = "cr"
	VarCC    = "cc"
	VarCC1h  = "cc1h"
	VarImg   = "img"
	VarImgCR = "img_cr"
	VarAI    = "ai"
	VarAO    = "ao"
)

// PricedVars lists the token variables that carry a price. `len` is excluded:
// it only feeds tier conditions. Callers use this to split an evaluated total
// into the per-line-item costs reported to the ledger.
var PricedVars = []string{VarP, VarC, VarCR, VarCC, VarCC1h, VarImg, VarImgCR, VarAI, VarAO}

// TokenParams holds every token dimension passed into an evaluation.
//
// All fields are raw token counts — never pre-scaled by 1e6; the expression
// carries the per-million price and the caller divides the result by 1e6.
//
// P and C are the fallback variables: they stand for every token that the
// expression does not price separately. Callers subtract the sub-categories an
// expression references, so a cache-unaware expression keeps working unchanged.
type TokenParams struct {
	P     float64 // prompt tokens, excluding separately priced sub-categories
	C     float64 // completion tokens, excluding separately priced sub-categories
	Len   float64 // full input context length; never reduced, used by tier conditions
	CR    float64 // cache read (hit) tokens
	CC    float64 // cache creation tokens (5-minute TTL / generic)
	CC1h  float64 // cache creation tokens, 1-hour TTL
	Img   float64 // image input tokens
	ImgCR float64 // image cache read tokens, split only when explicitly priced
	AI    float64 // audio input tokens
	AO    float64 // audio output tokens
}

// TraceResult carries side-channel information captured while an expression ran.
type TraceResult struct {
	// MatchedTier is the tier name passed to the outermost tier() call that
	// evaluated. Empty when the expression never calls tier().
	MatchedTier string
	// Cost is the value given to that tier() call.
	Cost float64
}

// ExprHashString returns the SHA-256 hex digest used as the compile-cache key.
// ExprHashString returns the SHA-256 hex digest used as the compile-cache key.
func ExprHashString(expr string) string {
	sum := sha256.Sum256([]byte(expr))
	return fmt.Sprintf("%x", sum)
}
