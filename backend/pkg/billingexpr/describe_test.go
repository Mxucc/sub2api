package billingexpr

import (
	"fmt"
	"testing"
)

// deepseekFlashExpr mirrors the built-in DeepSeek expression so the display
// parser is exercised against the shape it actually has to render.
const deepseekFlashExpr = `v1:weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18)) ? tier("peak", p * 0.30 + cr * 0.006 + c * 1.20) : tier("off_peak", p * 0.15 + cr * 0.003 + c * 0.60)`

func TestParseTiers_FlatPricing(t *testing.T) {
	parsed, err := ParseTiers(`tier("base", p * 2.5 + c * 15 + cr * 0.25)`)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if !parsed.Recognized {
		t.Fatal("Recognized = false, want true")
	}
	if parsed.TimeDependent {
		t.Error("TimeDependent = true, want false for a flat price")
	}
	if len(parsed.Tiers) != 1 {
		t.Fatalf("tiers = %d, want 1", len(parsed.Tiers))
	}
	tier := parsed.Tiers[0]
	if tier.Name != "base" {
		t.Errorf("name = %q, want %q", tier.Name, "base")
	}
	if tier.Condition != "" {
		t.Errorf("condition = %q, want empty", tier.Condition)
	}
	want := map[string]float64{VarP: 2.5, VarC: 15, VarCR: 0.25}
	for name, value := range want {
		if tier.Coefficients[name] != value {
			t.Errorf("coefficient[%s] = %v, want %v", name, tier.Coefficients[name], value)
		}
	}
	if fmt.Sprint(parsed.Variables) != fmt.Sprint([]string{VarP, VarC, VarCR}) {
		t.Errorf("variables = %v, want canonical order", parsed.Variables)
	}
}

// TestParseTiers_ConditionalPricing is the main case: a length-tiered price must
// render as two tiers with their conditions while keeping source order.
func TestParseTiers_ConditionalPricing(t *testing.T) {
	parsed, err := ParseTiers(`len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12)`)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if !parsed.Recognized {
		t.Fatal("Recognized = false, want true")
	}
	if len(parsed.Tiers) != 2 {
		t.Fatalf("tiers = %d, want 2", len(parsed.Tiers))
	}
	if parsed.Tiers[0].Name != "standard" || parsed.Tiers[0].Condition != "len <= 200000" {
		t.Errorf("tier 0 = %+v, want standard guarded by len <= 200000", parsed.Tiers[0])
	}
	if parsed.Tiers[1].Name != "long_context" || parsed.Tiers[1].Condition != "" {
		t.Errorf("tier 1 = %+v, want long_context as the final else branch", parsed.Tiers[1])
	}
	if parsed.Tiers[1].Coefficients[VarCC1h] != 12 {
		t.Errorf("long_context cc1h = %v, want 12", parsed.Tiers[1].Coefficients[VarCC1h])
	}
}

// TestParseTiers_DeepSeekTimePricing pins the case the whole feature exists for:
// a time-of-day price must render both tiers, flag itself as time dependent and
// describe its clock windows in a form a human can read.
func TestParseTiers_DeepSeekTimePricing(t *testing.T) {
	parsed, err := ParseTiers(deepseekFlashExpr)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if !parsed.Recognized {
		t.Fatal("Recognized = false, want true")
	}
	if !parsed.TimeDependent {
		t.Error("TimeDependent = false, want true")
	}
	if parsed.Version != 1 {
		t.Errorf("version = %d, want 1", parsed.Version)
	}
	if len(parsed.Tiers) != 2 {
		t.Fatalf("tiers = %d, want 2", len(parsed.Tiers))
	}

	peak, offPeak := parsed.Tiers[0], parsed.Tiers[1]
	if peak.Name != "peak" {
		t.Errorf("tier 0 name = %q, want peak", peak.Name)
	}
	if offPeak.Name != "off_peak" {
		t.Errorf("tier 1 name = %q, want off_peak", offPeak.Name)
	}

	for name, value := range map[string]float64{VarP: 0.30, VarCR: 0.006, VarC: 1.20} {
		if peak.Coefficients[name] != value {
			t.Errorf("peak[%s] = %v, want %v", name, peak.Coefficients[name], value)
		}
	}
	for name, value := range map[string]float64{VarP: 0.15, VarCR: 0.003, VarC: 0.60} {
		if offPeak.Coefficients[name] != value {
			t.Errorf("off_peak[%s] = %v, want %v", name, offPeak.Coefficients[name], value)
		}
	}

	if peak.Timezone != "Asia/Shanghai" {
		t.Errorf("peak timezone = %q, want Asia/Shanghai", peak.Timezone)
	}
	if len(peak.TimeWindows) != 1 {
		t.Fatalf("peak time windows = %v, want one rendered window", peak.TimeWindows)
	}
	if want := "周一至周五 09:00-12:00, 14:00-18:00"; peak.TimeWindows[0] != want {
		t.Errorf("peak time window = %q, want %q", peak.TimeWindows[0], want)
	}
	if len(offPeak.TimeWindows) != 0 {
		t.Errorf("off_peak time windows = %v, want none (it is the else branch)", offPeak.TimeWindows)
	}
}

func TestParseTiers_PeakRateIsDoubleOffPeak(t *testing.T) {
	parsed, err := ParseTiers(deepseekFlashExpr)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	// The published rule is "off-peak is half of peak". Encoding it as an
	// assertion makes a typo in either coefficient fail here instead of silently
	// charging the wrong rate.
	for _, variable := range []string{VarP, VarC, VarCR} {
		peak := parsed.Tiers[0].Coefficients[variable]
		offPeak := parsed.Tiers[1].Coefficients[variable]
		if peak == 0 || offPeak == 0 {
			t.Fatalf("%s missing from a tier: peak=%v off_peak=%v", variable, peak, offPeak)
		}
		if ratio := peak / offPeak; ratio < 1.999 || ratio > 2.001 {
			t.Errorf("%s peak/off_peak = %v, want 2", variable, ratio)
		}
	}
}

func TestParseTiers_PerRequestPrice(t *testing.T) {
	parsed, err := ParseTiers(`len <= 32000 ? tier("short", 0.01) : tier("long", p * 2 + c * 8)`)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if !parsed.Recognized {
		t.Fatal("Recognized = false, want true")
	}
	if parsed.Tiers[0].Unit != "per_request" {
		t.Errorf("short tier unit = %q, want per_request", parsed.Tiers[0].Unit)
	}
	if parsed.Tiers[0].Constant != 0.01 {
		t.Errorf("short tier constant = %v, want 0.01", parsed.Tiers[0].Constant)
	}
	if parsed.Tiers[1].Unit != "per_million_tokens" {
		t.Errorf("long tier unit = %q, want per_million_tokens", parsed.Tiers[1].Unit)
	}
}

// TestParseTiers_FlatRequestPriceKeepsItsTimeWindow guards a display-only trap: a
// flat request price is easy to read as unconditional, so the window has to be
// carried through or the page reports a static price for a price that varies by
// the hour.
func TestParseTiers_FlatRequestPriceKeepsItsTimeWindow(t *testing.T) {
	parsed, err := ParseTiers(`v1:hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 18 ? tier("peak", 0.05) : tier("off_peak", 0.02)`)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if !parsed.Recognized {
		t.Fatal("Recognized = false, want true")
	}
	if !parsed.TimeDependent {
		t.Error("TimeDependent = false, want true for a windowed request price")
	}
	if len(parsed.Tiers[0].TimeWindows) != 1 || parsed.Tiers[0].TimeWindows[0] != "09:00-18:00" {
		t.Errorf("peak time windows = %v, want [09:00-18:00]", parsed.Tiers[0].TimeWindows)
	}
	if parsed.Tiers[0].Timezone != "Asia/Shanghai" {
		t.Errorf("peak timezone = %q, want Asia/Shanghai", parsed.Tiers[0].Timezone)
	}
	if parsed.Tiers[1].Unit != "per_request" || parsed.Tiers[1].Constant != 0.02 {
		t.Errorf("off_peak tier = %+v, want a $0.02 request price", parsed.Tiers[1])
	}
}

func TestParseTiers_UnversionedExpression(t *testing.T) {
	parsed, err := ParseTiers(`tier("base", p * 1)`)
	if err != nil {
		t.Fatalf("ParseTiers: %v", err)
	}
	if parsed.Version != DefaultExprVersion {
		t.Errorf("version = %d, want %d", parsed.Version, DefaultExprVersion)
	}
}

// TestParseTiers_RejectsNonCanonical is the guard that keeps a misleading table
// off the price page: an expression the parser does not fully understand must
// report itself unrecognised rather than render a partial price list.
func TestParseTiers_RejectsNonCanonical(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
	}{
		{"no tier wrapper", `p * 3 + c * 15`},
		{"variables multiplied", `tier("base", p * c * 1)`},
		{"division inside tier", `tier("base", p * 3 / len)`},
		{"tier multiplied", `tier("base", p * 3) * 2`},
		{"tiers added", `tier("a", p * 1) + tier("b", c * 1)`},
		{"conditional inside tier", `tier("base", len > 100 ? p * 3 : p * 1)`},
		{"nested call inside tier", `tier("base", hour("UTC") * p * 1)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseTiers(tc.expression)
			if err != nil {
				t.Fatalf("ParseTiers returned an error (%v); want Recognized=false instead", err)
			}
			if parsed.Recognized {
				t.Errorf("Recognized = true for %q, want false", tc.expression)
			}
			if len(parsed.Tiers) != 0 {
				t.Errorf("tiers = %v, want none", parsed.Tiers)
			}
		})
	}
}

func TestParseTiers_SyntaxError(t *testing.T) {
	for _, expression := range []string{"", "   ", "tier(", "v1:"} {
		if _, err := ParseTiers(expression); err == nil {
			t.Errorf("ParseTiers(%q) = nil error, want a syntax error", expression)
		}
	}
}

// TestParseTiers_EveryBuiltinExpressionRenders guards the built-in table: a new
// expression that the display parser cannot read would otherwise show a raw
// string on the price page instead of a rate table.
func TestParseTiers_EveryBuiltinExpressionRenders(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
	}{
		{"deepseek flash", deepseekFlashExpr},
		{"deepseek pro", `v1:weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18)) ? tier("peak", p * 1.32 + cr * 0.044 + c * 3.96) : tier("off_peak", p * 0.66 + cr * 0.022 + c * 1.98)`},
		{"gpt long context", `len <= 272000 ? tier("standard", p * 10 + c * 50 + cr * 1 + cc * 12.5) : tier("long_context", p * 20 + c * 75 + cr * 2 + cc * 25)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseTiers(tc.expression)
			if err != nil {
				t.Fatalf("ParseTiers: %v", err)
			}
			if !parsed.Recognized {
				t.Fatalf("Recognized = false for %q", tc.expression)
			}
			if len(parsed.Tiers) < 2 {
				t.Errorf("tiers = %d, want at least 2", len(parsed.Tiers))
			}
		})
	}
}

func TestDescribeHourComparisons(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single window",
			input: `hour("UTC") >= 1 && hour("UTC") < 4`,
			want:  []string{"01:00-04:00"},
		},
		{
			name:  "two windows inside redundant grouping",
			input: `((hour("UTC") >= 9 && hour("UTC") < 12) || (hour("UTC") >= 14 && hour("UTC") < 18))`,
			want:  []string{"09:00-12:00", "14:00-18:00"},
		},
		{
			name:  "inclusive upper bound",
			input: `hour("UTC") >= 0 && hour("UTC") <= 8`,
			want:  []string{"00:00-08:00"},
		},
		{
			name:  "no hour comparison",
			input: `len > 100`,
			want:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := describeHourComparisons(scanComparisons(tc.input))
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("describeHourComparisons = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestScanComparisons_WeekdayIsNotReadAsDay guards the identifier boundary: a
// scan for "day(" must not also match the tail of "weekday(", which would double
// count every day bound and render the window wrong.
func TestScanComparisons_WeekdayIsNotReadAsDay(t *testing.T) {
	found := scanComparisons(`weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5`)
	if len(found) != 2 {
		t.Fatalf("comparisons = %d, want 2: %+v", len(found), found)
	}
	for _, item := range found {
		if item.function != "weekday" {
			t.Errorf("function = %q, want weekday", item.function)
		}
	}
}

// TestDescribeWeekdayComparisons pins the day-range rendering, including the
// interpretation of a half-open condition: `weekday >= 1` means "not Sunday", so it
// renders as the whole span rather than as nothing — otherwise a day-dependent
// price would look unconditional.
func TestDescribeWeekdayComparisons(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{`weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5`, "周一至周五"},
		{`weekday("Asia/Shanghai") == 6`, "周六"},
		{`weekday("Asia/Shanghai") == 0`, "周日"},
		{`weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 3`, "周一至周三"},
		{`weekday("Asia/Shanghai") >= 1`, "周一至周六"},
		{`weekday("Asia/Shanghai") <= 5`, "周日至周五"},
		{`weekday("Asia/Shanghai") >= 0`, "周日至周六"},
		{`len > 100`, ""},
	} {
		if got := describeWeekdayComparisons(scanComparisons(tc.input)); got != tc.want {
			t.Errorf("describeWeekdayComparisons(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
