package billingexpr

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
)

// smokeVectorInstants are the instants Validate evaluates at. They straddle a
// time-conditional price from both sides so every branch runs:
//
//   - 2026-09-15 02:00 UTC is Tuesday 10:00 Beijing, inside DeepSeek's peak
//     window.
//   - 2026-09-15 14:00 UTC is Tuesday 22:00 Beijing, outside every peak window.
//
// An expression with no time condition is unaffected by the choice.
var smokeVectorInstants = []time.Time{
	time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC),
	time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC),
}

// locationCache memoises IANA zone lookups. A time-conditional expression calls
// hour()/weekday() several times per evaluation, and the same handful of zones
// (Asia/Shanghai, UTC) recur across every model.
var locationCache sync.Map

// resolveLocation returns the named IANA zone, falling back to UTC for an empty
// or unknown name. A bad zone name must never fail billing: it degrades to UTC.
func resolveLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.UTC
	}
	if cached, ok := locationCache.Load(name); ok {
		if location, valid := cached.(*time.Location); valid && location != nil {
			return location
		}
		locationCache.Delete(name)
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		// Cache the fallback so an unresolvable zone does not retry the
		// filesystem lookup on every hour()/weekday() call.
		locationCache.Store(name, time.UTC)
		return time.UTC
	}
	actual, _ := locationCache.LoadOrStore(name, location)
	if cached, ok := actual.(*time.Location); ok && cached != nil {
		return cached
	}
	return location
}

// timeInZone converts an instant into the named zone. The instant is passed in
// by the caller rather than read from the clock so historical recomputation
// (back-filling a past request) evaluates time-conditional pricing against the
// request's own billing moment.
func timeInZone(at time.Time, name string) time.Time {
	return at.In(resolveLocation(name))
}

// Run evaluates an expression and returns its raw result plus the captured
// trace. The result is in USD per million tokens; divide by 1e6 for dollars.
//
// `at` is the billing instant that time functions observe. A zero value falls
// back to UTC now, which keeps callers that have no request timestamp working
// while making the normal path fully deterministic.
func Run(expression string, params TokenParams, at time.Time) (float64, TraceResult, error) {
	entry, err := compileEntry(expression, ExprHashString(expression))
	if err != nil {
		return 0, TraceResult{}, err
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	trace := TraceResult{}
	// tierErr records a malformed tier() argument. The tier callback cannot
	// return an error, so the failure is surfaced after evaluation and the whole
	// run is rejected: charging $0 for an expression the engine could not read
	// would silently give the request away.
	var tierErr error
	coerce := func(value any) (float64, bool) {
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case int32:
			return float64(v), true
		case uint:
			return float64(v), true
		case uint64:
			return float64(v), true
		case uint32:
			return float64(v), true
		default:
			return 0, false
		}
	}
	traceNum := func(name string, value any) float64 {
		number, ok := coerce(value)
		if !ok {
			if tierErr == nil {
				tierErr = fmt.Errorf("expr run error: %s() value must be numeric, got %T", name, value)
			}
			return 0
		}
		return number
	}

	env := map[string]any{
		VarP:     params.P,
		VarC:     params.C,
		VarLen:   params.Len,
		VarCR:    params.CR,
		VarCC:    params.CC,
		VarCC1h:  params.CC1h,
		VarImg:   params.Img,
		VarImgCR: params.ImgCR,
		VarAI:    params.AI,
		VarAO:    params.AO,
		"tier": func(name string, value any) float64 {
			// Every tier() call along the evaluated path is recorded; ternary
			// branches are mutually exclusive, so the value that survives is the
			// tier that actually priced the request.
			number := traceNum("tier", value)
			if tierErr == nil {
				trace.MatchedTier = name
				trace.Cost = number
			}
			return number
		},
		"hour":    func(tz string) int { return timeInZone(at, tz).Hour() },
		"minute":  func(tz string) int { return timeInZone(at, tz).Minute() },
		"weekday": func(tz string) int { return int(timeInZone(at, tz).Weekday()) },
		"month":   func(tz string) int { return int(timeInZone(at, tz).Month()) },
		"day":     func(tz string) int { return timeInZone(at, tz).Day() },
		"max":     func(a, b any) float64 { return math.Max(traceNum("max", a), traceNum("max", b)) },
		"min":     func(a, b any) float64 { return math.Min(traceNum("min", a), traceNum("min", b)) },
		"abs":     func(a any) float64 { return math.Abs(traceNum("abs", a)) },
		"ceil":    func(a any) float64 { return math.Ceil(traceNum("ceil", a)) },
		"floor":   func(a any) float64 { return math.Floor(traceNum("floor", a)) },
	}

	out, err := expr.Run(entry.prog, env)
	if err != nil {
		return 0, trace, fmt.Errorf("expr run error: %w", err)
	}
	// The check belongs after the run: tierErr is only set while evaluating, so
	// testing it earlier would let a malformed tier() argument produce a silent
	// $0 charge.
	if tierErr != nil {
		return 0, TraceResult{}, tierErr
	}
	result, ok := out.(float64)
	if !ok {
		return 0, trace, fmt.Errorf("expr run error: result is %T, want float64", out)
	}
	// An expression can produce Inf from division by zero on a zero-token
	// request. Rejecting it here lets callers fail closed instead of charging an
	// absurd amount.
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, trace, fmt.Errorf("expr run error: result must be finite, got %v", result)
	}
	if result < 0 {
		return 0, trace, fmt.Errorf("expr run error: result must be non-negative, got %v", result)
	}
	return result, trace, nil
}

// PricedVarsUsed filters PricedVars down to the variables an expression
// actually references, in the canonical order of PricedVars.
func PricedVarsUsed(expression string) []string {
	used := UsedVars(expression)
	if len(used) == 0 {
		return nil
	}
	result := make([]string, 0, len(used))
	for _, name := range PricedVars {
		if used[name] {
			result = append(result, name)
		}
	}
	return result
}
