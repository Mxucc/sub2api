package billingexpr

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"
)

// shanghai renders an instant in Asia/Shanghai so table entries can be written
// with Beijing wall-clock times, matching how DeepSeek documents its windows.
func shanghai(t *testing.T, year int, month time.Month, day, hour, minute int) time.Time {
	t.Helper()
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load Asia/Shanghai: %v", err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}

func TestParseExprVersion(t *testing.T) {
	for _, tc := range []struct {
		name        string
		expression  string
		wantVersion int
		wantBody    string
	}{
		{"versioned", "v1:tier(\"base\", p * 1)", 1, "tier(\"base\", p * 1)"},
		{"unversioned defaults to v1", "tier(\"base\", p * 1)", DefaultExprVersion, "tier(\"base\", p * 1)"},
		{"v1 prefix only at start", "tier(\"v1:x\", p)", DefaultExprVersion, "tier(\"v1:x\", p)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version, body := ParseExprVersion(tc.expression)
			if version != tc.wantVersion {
				t.Errorf("version = %d, want %d", version, tc.wantVersion)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}

func TestRun_ArithmeticComparisonAndTernary(t *testing.T) {
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)

	// Coefficients are USD per million tokens and the result stays in that
	// unit: p * 3 + c * 15 on 1000/2000 tokens is 3000 + 30000 = 33000.
	result, trace, err := Run(`tier("base", p * 3 + c * 15)`, TokenParams{P: 1000, C: 2000}, at)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := 33000.0; result != want {
		t.Errorf("result = %v, want %v", result, want)
	}
	if trace.MatchedTier != "base" {
		t.Errorf("MatchedTier = %q, want %q", trace.MatchedTier, "base")
	}
	if trace.Cost != result {
		t.Errorf("trace.Cost = %v, want %v", trace.Cost, result)
	}

	// A ternary selects exactly one branch, so only the matching tier is traced.
	result, trace, err = Run(`len <= 1000 ? tier("short", p * 2 + c * 8) : tier("long", p * 6 + c * 22.5)`,
		TokenParams{P: 500, Len: 1000, C: 500}, at)
	if err != nil {
		t.Fatalf("Run ternary: %v", err)
	}
	if trace.MatchedTier != "short" {
		t.Fatalf("MatchedTier = %q, want %q", trace.MatchedTier, "short")
	}
	if want := 500*2.0 + 500*8.0; result != want {
		t.Errorf("result = %v, want %v", result, want)
	}

	result, trace, err = Run(`len <= 1000 ? tier("short", p * 2) : tier("long", p * 6)`,
		TokenParams{P: 1000, Len: 1001}, at)
	if err != nil {
		t.Fatalf("Run ternary long: %v", err)
	}
	if trace.MatchedTier != "long" {
		t.Errorf("MatchedTier = %q, want %q", trace.MatchedTier, "long")
	}
	if want := 6000.0; result != want {
		t.Errorf("result = %v, want %v", result, want)
	}
}

func TestRun_MathHelpers(t *testing.T) {
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	result, _, err := Run(`tier("m", max(p, 10) + min(c, 5) + abs(0 - 3) + ceil(1.2) + floor(2.8))`,
		TokenParams{P: 1, C: 99}, at)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := 10.0 + 5.0 + 3.0 + 2.0 + 2.0; result != want {
		t.Errorf("result = %v, want %v", result, want)
	}
}

// TestRun_TimeFunctions pins the weekday/hour contract that time-conditional
// pricing relies on: weekday() is 0=Sunday..6=Saturday and hour() is local.
func TestRun_TimeFunctions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		at      time.Time
		wantDay int
		wantHr  int
	}{
		{"tuesday morning Beijing", shanghai(t, 2026, time.September, 15, 10, 30), 2, 10},
		{"saturday Beijing", shanghai(t, 2026, time.September, 19, 10, 30), 6, 10},
		{"sunday Beijing", shanghai(t, 2026, time.September, 20, 10, 30), 0, 10},
		{"monday midnight Beijing", shanghai(t, 2026, time.September, 21, 0, 0), 1, 0},
		{"monday 23:59 Beijing", shanghai(t, 2026, time.September, 21, 23, 59), 1, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := Run(`tier("t", weekday("Asia/Shanghai") * 100 + hour("Asia/Shanghai"))`, TokenParams{}, tc.at)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if want := float64(tc.wantDay*100 + tc.wantHr); result != want {
				t.Errorf("result = %v, want weekday=%d hour=%d", result, tc.wantDay, tc.wantHr)
			}
		})
	}
}

// TestRun_TimeFunctionsAreEvaluateTimeNotWallClock verifies the instant passed
// by the caller drives the time functions, so back-filling an old request
// recomputes the price that applied then.
func TestRun_TimeFunctionsAreEvaluateTimeNotWallClock(t *testing.T) {
	expression := `tier("t", hour("Asia/Shanghai"))`
	// 2026-09-15 02:00 UTC is 10:00 Beijing; 2026-09-15 14:00 UTC is 22:00.
	morning, _, err := Run(expression, TokenParams{}, time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Run morning: %v", err)
	}
	evening, _, err := Run(expression, TokenParams{}, time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Run evening: %v", err)
	}
	if morning != 10 || evening != 22 {
		t.Errorf("got morning=%v evening=%v, want 10 and 22", morning, evening)
	}
}

func TestRun_UnknownTimezoneFallsBackToUTC(t *testing.T) {
	// 2026-09-15 02:00 UTC.
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	for _, zone := range []string{`"Not/AZone"`, `""`} {
		result, _, err := Run(`tier("t", hour(`+zone+`))`, TokenParams{}, at)
		if err != nil {
			t.Fatalf("Run zone %s: %v", zone, err)
		}
		if result != 2 {
			t.Errorf("zone %s: hour = %v, want 2 (UTC)", zone, result)
		}
	}
}

func TestRun_RejectsNonFiniteAndNegative(t *testing.T) {
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name       string
		expression string
	}{
		{"division by zero input", `tier("t", p / len * 1)`},
		{"negative result", `tier("t", p - 1000)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Run(tc.expression, TokenParams{}, at); err == nil {
				t.Errorf("Run(%q) succeeded, want error", tc.expression)
			}
		})
	}
}

func TestRun_CompileErrors(t *testing.T) {
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	for _, expression := range []string{
		"",
		"v1:",
		"tier(",
		`tier("t", unknown_variable * 1)`,
		`tier("t", p * "text")`,
	} {
		if _, _, err := Run(expression, TokenParams{P: 1}, at); err == nil {
			t.Errorf("Run(%q) succeeded, want error", expression)
		}
	}
}

func TestRun_ZeroParamsAreValid(t *testing.T) {
	// A zero-token request must still evaluate: cache-only or a rejected
	// upstream call produces all-zero usage and must not divide by zero.
	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	result, _, err := Run(`tier("t", p * 3 + c * 15 + cr * 0.3)`, TokenParams{}, at)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != 0 {
		t.Errorf("result = %v, want 0", result)
	}
}

func TestUsedVars(t *testing.T) {
	used := UsedVars(`tier("t", p * 3 + c * 15 + cr * 0.3)`)
	for _, want := range []string{"p", "c", "cr", "tier"} {
		if !used[want] {
			t.Errorf("usedVars missing %q", want)
		}
	}
	for _, unwanted := range []string{"img", "ai", "ao", "cc"} {
		if used[unwanted] {
			t.Errorf("usedVars unexpectedly contains %q", unwanted)
		}
	}

	if got := UsedVars(""); got != nil {
		t.Errorf("UsedVars(\"\") = %v, want nil", got)
	}
	if got := UsedVars("tier("); got != nil {
		t.Errorf("UsedVars(invalid) = %v, want nil", got)
	}
}

// TestUsedVars_DetectsBranchSkippedAtRuntime pins the AST-introspection
// contract: a variable referenced only in an unreachable branch must still be
// reported, otherwise the caller would leave those tokens inside p and
// double-charge them when the branch does become reachable.
func TestUsedVars_DetectsBranchSkippedAtRuntime(t *testing.T) {
	used := UsedVars(`len > 1000 ? tier("long", p * 6 + cr * 0.6) : tier("short", p * 3)`)
	if !used["cr"] {
		t.Error("usedVars must include cr even though the short branch wins at runtime")
	}
}

func TestPricedVarsUsed(t *testing.T) {
	got := PricedVarsUsed(`tier("t", c * 15 + p * 3 + img * 2.5)`)
	want := []string{VarP, VarC, VarImg}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("PricedVarsUsed = %v, want %v (canonical order)", got, want)
	}
	if got := PricedVarsUsed(""); got != nil {
		t.Errorf("PricedVarsUsed(\"\") = %v, want nil", got)
	}
}

func TestValidate(t *testing.T) {
	valid := []string{
		`tier("base", p * 2.5 + c * 15)`,
		`v1:tier("base", p * 2.5 + c * 15)`,
		`len <= 272000 ? tier("standard", p * 10 + c * 50) : tier("long", p * 20 + c * 75)`,
		`weekday("Asia/Shanghai") >= 1 ? tier("peak", p * 0.30 + c * 1.20) : tier("off", p * 0.15 + c * 0.60)`,
	}
	for _, expression := range valid {
		if err := Validate(expression); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", expression, err)
		}
	}

	invalid := []string{
		"",
		"tier(",
		`tier("t", p * 1) + tier("u", c * 1) - 999999`,
	}
	for _, expression := range invalid {
		if err := Validate(expression); err == nil {
			t.Errorf("Validate(%q) = nil, want error", expression)
		}
	}
}

// TestValidate_CoversBranchesThatOnlyRunAtAnotherTime is the guard for the
// two-instant smoke test: a conditional price executes only the branch its
// condition selects, so validating at a single instant would leave the other
// branch unchecked and a bad coefficient there would mis-charge later in the day.
func TestValidate_CoversBranchesThatOnlyRunAtAnotherTime(t *testing.T) {
	// The peak branch is valid, the off-peak branch is not: subtracting a large
	// constant makes the whole expression negative, but only when that branch runs.
	brokenOffPeak := `hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 18 ? tier("peak", p * 1) : tier("off", p * 1) - 500000`
	if err := Validate(brokenOffPeak); err == nil {
		t.Error("Validate accepted an expression whose off-peak branch is invalid")
	}

	// A negative constant in the branch that does not run at the peak instant is
	// the same class of defect.
	brokenNegative := `hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 18 ? tier("peak", p * 1) : tier("off", 0 - 1)`
	if err := Validate(brokenNegative); err == nil {
		t.Error("Validate accepted a branch with a negative constant")
	}

	// A correct expression of the same shape must still pass, or the assertions
	// above would be satisfied by rejecting everything.
	valid := `hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 18 ? tier("peak", p * 2) : tier("off", p * 1)`
	if err := Validate(valid); err != nil {
		t.Errorf("Validate rejected a valid expression: %v", err)
	}
}

// TestCompileCache is the guard for the cache's correctness contract: an
// identical string is served from cache, and a changed string is a miss even
// when it would hash-collide in a naive key scheme.
func TestCompileCache(t *testing.T) {
	InvalidateCache()
	t.Cleanup(InvalidateCache)

	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	if _, err := CompileFromCache(`tier("t", p * 1)`); err != nil {
		t.Fatalf("first compile: %v", err)
	}
	cacheMu.RLock()
	sizeAfterFirst := len(cache)
	cacheMu.RUnlock()
	if sizeAfterFirst != 1 {
		t.Fatalf("cache size = %d, want 1", sizeAfterFirst)
	}

	if _, err := CompileFromCache(`tier("t", p * 1)`); err != nil {
		t.Fatalf("cached compile: %v", err)
	}
	cacheMu.RLock()
	sizeAfterSecond := len(cache)
	cacheMu.RUnlock()
	if sizeAfterSecond != 1 {
		t.Fatalf("cache size = %d, want 1 (cache miss on identical expression)", sizeAfterSecond)
	}

	// A different expression must produce a different program.
	result, _, err := Run(`tier("t", p * 2)`, TokenParams{P: 10}, at)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != 20 {
		t.Errorf("result = %v, want 20 (stale program served from cache?)", result)
	}
}

// TestRun_ConcurrentEvaluation exercises the shared cache under contention:
// -race turns a missing lock into a failure.
func TestRun_ConcurrentEvaluation(t *testing.T) {
	InvalidateCache()
	t.Cleanup(InvalidateCache)

	at := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	expressions := []string{
		`tier("a", p * 3 + c * 15)`,
		`tier("b", p * 1 + cr * 0.3 + c * 5)`,
		`weekday("Asia/Shanghai") >= 1 ? tier("peak", p * 0.30) : tier("off", p * 0.15)`,
	}

	var wg sync.WaitGroup
	for i := 0; i < 48; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			expression := expressions[index%len(expressions)]
			result, _, err := Run(expression, TokenParams{P: 100, C: 100, CR: 50, Len: 100}, at)
			if err != nil {
				t.Errorf("Run: %v", err)
				return
			}
			if math.IsNaN(result) || math.IsInf(result, 0) || result < 0 {
				t.Errorf("Run returned non-finite/negative %v", result)
			}
		}(i)
	}
	wg.Wait()
}
