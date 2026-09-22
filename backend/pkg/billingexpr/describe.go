package billingexpr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// Tier describes one leaf of a billing expression for display purposes.
//
// Expressions are evaluated by the VM, but a price list has to show a human what
// the expression means: which tiers exist, what each tier charges, and under what
// condition it applies. Parsing the AST gives that structure without evaluating
// anything, so the display works for models with no traffic at all.
type Tier struct {
	// Name is the label passed to tier(), e.g. "peak" / "off_peak" / "base".
	Name string `json:"name"`
	// Condition is the condition guarding this tier, or "" for the final else
	// branch / an unconditional expression.
	Condition string `json:"condition,omitempty"`
	// Coefficients maps each priced variable the tier references to its price in
	// USD per million tokens.
	Coefficients map[string]float64 `json:"coefficients,omitempty"`
	// Constant is a per-request amount added by the tier, in USD. Zero when the
	// tier prices tokens only.
	Constant float64 `json:"constant,omitempty"`
	// Unit is the price's meter: "per_million_tokens" for the usual case,
	// "per_request" when the tier is a flat request price.
	Unit string `json:"unit"`
	// Variables lists the priced variables in canonical order, for stable display.
	Variables []string `json:"variables,omitempty"`
	// TimeWindows lists the clock windows a time function in Condition implies,
	// e.g. "周一至周五 09:00-12:00, 14:00-18:00". Empty when the condition is
	// not a time predicate the parser recognises.
	TimeWindows []string `json:"time_windows,omitempty"`
	// Timezone is the zone the Condition's time functions were called with.
	Timezone string `json:"timezone,omitempty"`
}

// ParsedExpression is the display view of a billing expression.
type ParsedExpression struct {
	// Version is the expression's declared version.
	Version int `json:"version"`
	// Tiers lists every leaf, in source order.
	Tiers []Tier `json:"tiers"`
	// Variables lists every priced variable referenced anywhere, canonical order.
	Variables []string `json:"variables,omitempty"`
	// TimeDependent reports whether any tier condition reads the clock, which is
	// what makes the price change over the day.
	TimeDependent bool `json:"time_dependent"`
	// Recognized is false when the expression does not use the canonical
	// conditional-pricing shape. The tiers list is then empty and a caller should
	// show the raw expression instead of a table.
	Recognized bool `json:"recognized"`
}

// priceVarOrder fixes the order coefficients are displayed in, matching PricedVars.
var priceVarOrder = PricedVars

// ParseTiers parses a billing expression into its display structure.
//
// It never evaluates the expression, so a model with a price but no traffic still
// renders. A syntax error is returned as an error; a syntactically valid
// expression with a non-canonical shape returns Recognized=false rather than a
// misleading table.
func ParseTiers(expression string) (*ParsedExpression, error) {
	trimmed := strings.TrimSpace(expression)
	if trimmed == "" {
		return nil, fmt.Errorf("parse billing expression: expression is empty")
	}
	version, body := ParseExprVersion(trimmed)
	tree, err := parser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse billing expression: %w", err)
	}

	parsed := &ParsedExpression{Version: version}
	leaves := collectTierLeaves(tree.Node, "")

	for _, leaf := range leaves {
		tier, ok := parseTierLeaf(leaf)
		if !ok {
			// One non-canonical leaf makes the whole table unreliable: rendering a
			// partial price list would show a price the model may never charge.
			parsed.Tiers = nil
			parsed.Recognized = false
			return parsed, nil
		}
		parsed.Tiers = append(parsed.Tiers, *tier)
		if conditionUsesClock(leaf.condition) {
			// Tracked separately from the rendered windows: a price can depend on
			// the clock without the condition being a window this parser can name
			// (a one-sided bound such as `hour(tz) < 12`, for example). Reporting
			// it as static would hide a price that changes during the day.
			parsed.TimeDependent = true
		}
	}

	if len(parsed.Tiers) == 0 {
		parsed.Recognized = false
		return parsed, nil
	}

	seen := map[string]bool{}
	for _, tier := range parsed.Tiers {
		for _, name := range tier.Variables {
			seen[name] = true
		}
	}
	for _, name := range priceVarOrder {
		if seen[name] {
			parsed.Variables = append(parsed.Variables, name)
		}
	}
	parsed.Recognized = true
	return parsed, nil
}

// tierLeaf is one conditional branch: the tier() call plus the condition guarding it.
type tierLeaf struct {
	condition string
	call      *ast.CallNode
}

// collectTierLeaves walks the ternary chain that makes up a conditional price and
// returns one leaf per tier() call, tagged with its accumulated condition.
//
// A conditional price is a right-nested chain of Conditionals:
//
//	cond1 ? tier("a", ...) : cond2 ? tier("b", ...) : tier("c", ...)
//
// so descending only through the false branch reconstructs the tier order. A
// nested ternary inside a tier's value expression is not part of the chain and is
// left to parseTierLeaf, which rejects it.
func collectTierLeaves(node ast.Node, condition string) []tierLeaf {
	switch n := node.(type) {
	case *ast.ConditionalNode:
		condText := n.Cond.String()
		accumulated := condText
		if condition != "" {
			accumulated = condition + " && " + condText
		}
		return append(
			collectTierLeaves(n.Exp1, accumulated),
			collectTierLeaves(n.Exp2, condition)...,
		)
	case *ast.CallNode:
		return []tierLeaf{{condition: condition, call: n}}
	default:
		return nil
	}
}

// parseTierLeaf converts a tier() call into a display tier.
//
// The accepted shape is the canonical one: a single tier(name, <sum of priced
// variables times coefficients>). Anything else — nested ternaries, division,
// multiplied tiers, a missing tier() wrapper — is not recognised, and the caller
// falls back to showing the raw expression.
func parseTierLeaf(leaf tierLeaf) (*Tier, bool) {
	callee, ok := leaf.call.Callee.(*ast.IdentifierNode)
	if !ok || callee.Value != "tier" {
		return nil, false
	}
	if len(leaf.call.Arguments) != 2 {
		return nil, false
	}
	nameNode, ok := leaf.call.Arguments[0].(*ast.StringNode)
	if !ok {
		return nil, false
	}

	tier := &Tier{
		Name:         nameNode.Value,
		Condition:    leaf.condition,
		Coefficients: map[string]float64{},
		Unit:         "per_million_tokens",
	}

	// The value expression must be a flat sum of priced terms, or a bare
	// per-request amount.
	sum, isBinary := leaf.call.Arguments[1].(*ast.BinaryNode)
	if !isBinary {
		amount, isLiteral := numericLiteralValue(leaf.call.Arguments[1])
		if !isLiteral {
			return nil, false
		}
		tier.Constant = amount
		tier.Unit = "per_request"
		// A flat request price still has to describe its window: a
		// `cond ? tier("peak", 5) : tier("off", 2)` expression is time dependent
		// and must say so, or the price page reports a static price.
		tier.TimeWindows, tier.Timezone = describeTimeCondition(leaf.condition)
		return tier, true
	}

	recognized := true
	for _, term := range flattenSum(sum) {
		coefficient, variable, ok := parsePriceTerm(term)
		if !ok {
			recognized = false
			break
		}
		if variable == "" {
			// A constant term inside a token-priced tier: a per-request component.
			tier.Constant += coefficient
			tier.Unit = "per_million_tokens_plus_request"
			continue
		}
		tier.Coefficients[variable] += coefficient
	}
	if !recognized {
		return nil, false
	}

	for _, name := range priceVarOrder {
		if _, present := tier.Coefficients[name]; present {
			tier.Variables = append(tier.Variables, name)
		}
	}
	if len(tier.Variables) == 0 && tier.Constant == 0 {
		return nil, false
	}

	tier.TimeWindows, tier.Timezone = describeTimeCondition(leaf.condition)
	return tier, true
}

func numericLiteralValue(node ast.Node) (float64, bool) {
	switch n := node.(type) {
	case *ast.IntegerNode:
		return float64(n.Value), true
	case *ast.FloatNode:
		return n.Value, true
	case *ast.UnaryNode:
		if n.Operator != "-" {
			return 0, false
		}
		inner, ok := numericLiteralValue(n.Node)
		return -inner, ok
	default:
		return 0, false
	}
}

// flattenSum flattens an addition chain into its terms, so `a + b + c` yields
// three terms regardless of how the parser nested them.
func flattenSum(node *ast.BinaryNode) []ast.Node {
	if node.Operator != "+" {
		return []ast.Node{node}
	}
	var terms []ast.Node
	for _, side := range []ast.Node{node.Left, node.Right} {
		if binary, ok := side.(*ast.BinaryNode); ok {
			terms = append(terms, flattenSum(binary)...)
			continue
		}
		terms = append(terms, side)
	}
	return terms
}

// parsePriceTerm extracts `<number> * <variable>` or a bare literal.
//
// It returns variable=="" for a constant term. A term that is neither a numeric
// literal nor a numeric-literal-times-variable product is rejected, which is what
// keeps a non-canonical expression out of the price table.
func parsePriceTerm(node ast.Node) (coefficient float64, variable string, ok bool) {
	switch n := node.(type) {
	case *ast.IntegerNode:
		return float64(n.Value), "", true
	case *ast.FloatNode:
		return n.Value, "", true
	case *ast.UnaryNode:
		// A negated literal, e.g. -1 * p. Rare but representable.
		if n.Operator != "-" {
			return 0, "", false
		}
		inner, innerVariable, innerOK := parsePriceTerm(n.Node)
		if !innerOK {
			return 0, "", false
		}
		return -inner, innerVariable, true
	case *ast.IdentifierNode:
		// A bare priced variable means a coefficient of 1.
		if isPricedVar(n.Value) {
			return 1, n.Value, true
		}
		return 0, "", false
	case *ast.BinaryNode:
		if n.Operator != "*" {
			return 0, "", false
		}
		leftNumber, leftVariable, leftOK := parsePriceTerm(n.Left)
		rightNumber, rightVariable, rightOK := parsePriceTerm(n.Right)
		if !leftOK || !rightOK {
			return 0, "", false
		}
		switch {
		case leftVariable == "" && rightVariable != "":
			return leftNumber * rightNumber, rightVariable, true
		case rightVariable == "" && leftVariable != "":
			return leftNumber * rightNumber, leftVariable, true
		case leftVariable == "" && rightVariable == "":
			// A pure number product, e.g. 2 * 0.15.
			return leftNumber * rightNumber, "", true
		default:
			// Two different variables multiplied is not a linear price term.
			return 0, "", false
		}
	case *ast.CallNode:
		// max(..., 0) and friends are common in generated expressions; unwrap the
		// degenerate single-argument case and reject anything else.
		callee, isIdent := n.Callee.(*ast.IdentifierNode)
		if !isIdent || len(n.Arguments) == 0 {
			return 0, "", false
		}
		switch callee.Value {
		case "abs", "ceil", "floor", "max", "min":
			if len(n.Arguments) != 1 {
				return 0, "", false
			}
			return parsePriceTerm(n.Arguments[0])
		default:
			return 0, "", false
		}
	default:
		return 0, "", false
	}
}

// conditionUsesClock reports whether a tier condition reads the clock, whether or
// not the comparison is a window this package can render.
func conditionUsesClock(condition string) bool {
	if condition == "" {
		return false
	}
	for _, name := range timeFunctionNames {
		if strings.Contains(condition, name+"(") {
			return true
		}
	}
	return false
}

// isPricedVar reports whether a name is one of the token variables that carries a
// price, as opposed to `len` (condition-only) or a function name.
func isPricedVar(name string) bool {
	for _, candidate := range priceVarOrder {
		if candidate == name {
			return true
		}
	}
	return false
}

// timeFunctionNames are the clock predicates a condition can probe. Longer names
// come first so a scan for "day(" does not match the tail of "weekday(".
var timeFunctionNames = []string{"weekday", "minute", "month", "hour", "day"}

// comparison is one `<fn>("<tz>") <op> <int>` found in a condition.
type comparison struct {
	function string
	operator string
	value    int
}

// scanComparisons finds every clock comparison in a condition.
//
// It scans rather than splitting on the boolean operators because the conditions
// are written with redundant grouping — `((a && b) || (c && d))` — which defeats
// a naive split: the separators sit inside parentheses. Reading each call and the
// operator that follows it is insensitive to how the author grouped the terms.
func scanComparisons(condition string) []comparison {
	var found []comparison
	for index := 0; index < len(condition); index++ {
		function := ""
		for _, name := range timeFunctionNames {
			if !strings.HasPrefix(condition[index:], name+"(") {
				continue
			}
			// "weekday(" must not be read again as a bare "day(".
			if index > 0 && isIdentifierByte(condition[index-1]) {
				continue
			}
			function = name
			break
		}
		if function == "" {
			continue
		}
		open := index + len(function)
		closeOffset := strings.Index(condition[open:], ")")
		if closeOffset < 0 {
			continue
		}
		rest := strings.TrimLeft(condition[open+closeOffset+1:], " \t")
		operator, remainder := leadingOperator(rest)
		if operator == "" {
			continue
		}
		value, ok := leadingInt(remainder)
		if !ok {
			continue
		}
		found = append(found, comparison{function: function, operator: operator, value: value})
		index = open + closeOffset
	}
	return found
}

func isIdentifierByte(char byte) bool {
	return char == '_' ||
		(char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9')
}

func leadingOperator(text string) (operator, remainder string) {
	// Two-character operators first: ">=" must not be read as ">".
	for _, candidate := range []string{"==", ">=", "<=", "!=", ">", "<"} {
		if strings.HasPrefix(text, candidate) {
			return candidate, strings.TrimLeft(text[len(candidate):], " \t")
		}
	}
	return "", text
}

func leadingInt(text string) (int, bool) {
	text = strings.TrimLeft(text, " \t")
	end := 0
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	value, err := strconv.Atoi(text[:end])
	if err != nil {
		return 0, false
	}
	return value, true
}

// describeTimeCondition turns a condition's clock comparisons into a readable
// window label such as "周一至周五 09:00-12:00, 14:00-18:00".
//
// It returns nil when the condition is not a recognisable day-plus-hour
// predicate, so an unusual condition falls back to showing its raw text rather
// than an empty or wrong label.
func describeTimeCondition(condition string) (windows []string, timezone string) {
	if !strings.Contains(condition, "hour(") {
		return nil, ""
	}
	comparisons := scanComparisons(condition)
	if len(comparisons) == 0 {
		return nil, ""
	}
	timezone = extractTimeZone(condition)

	dayLabel := describeWeekdayComparisons(comparisons)
	ranges := describeHourComparisons(comparisons)
	if len(ranges) == 0 {
		return nil, ""
	}
	label := strings.Join(ranges, ", ")
	if dayLabel != "" {
		label = dayLabel + " " + label
	}
	return []string{label}, timezone
}

func extractTimeZone(condition string) string {
	// hour("Asia/Shanghai") — take the first quoted zone any time function uses.
	for _, function := range timeFunctionNames {
		index := strings.Index(condition, function+"(")
		if index < 0 {
			continue
		}
		rest := condition[index+len(function)+1:]
		quoteStart := strings.Index(rest, "\"")
		if quoteStart < 0 {
			continue
		}
		quoteEnd := strings.Index(rest[quoteStart+1:], "\"")
		if quoteEnd < 0 {
			continue
		}
		return rest[quoteStart+1 : quoteStart+1+quoteEnd]
	}
	return ""
}

// describeWeekdayComparisons renders the day bounds as a Chinese day range.
//
// A one-sided bound is read as its full span (`weekday >= 1` is Monday..Saturday,
// i.e. "not Sunday"), which is what the condition literally means. Returning
// nothing for it would leave a day-dependent price looking unconditional.
func describeWeekdayComparisons(comparisons []comparison) string {
	dayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	lower, upper := -1, -1
	for _, item := range comparisons {
		if item.function != "weekday" {
			continue
		}
		switch item.operator {
		case "==":
			lower, upper = item.value, item.value
		case ">=", ">":
			lower = max(item.value, lower)
		case "<=", "<":
			// Keep the widest bound when a condition names several.
			if upper < 0 {
				upper = item.value
			} else {
				upper = max(item.value, upper)
			}
		}
	}
	if lower < 0 && upper < 0 {
		return ""
	}
	if lower < 0 {
		lower = 0
	}
	if upper < 0 {
		upper = 6
	}
	if lower > upper || upper > 6 {
		return ""
	}
	if lower == upper {
		return dayNames[lower]
	}
	if lower == 1 && upper == 5 {
		return "周一至周五"
	}
	return dayNames[lower] + "至" + dayNames[upper]
}

// describeHourComparisons pairs each lower bound with the next upper bound,
// producing one clock range per hour window.
//
// Pairing in scan order matches how the windows are written — `(h >= 9 && h < 12)
// || (h >= 14 && h < 18)` scans as lower, upper, lower, upper — so no assumption
// about the boolean grouping is needed. A missing side is filled with the edge of
// the day (`hour < 12` is 00:00-12:00), which is what the condition means.
func describeHourComparisons(comparisons []comparison) []string {
	const (
		startOfDay = 0
		endOfDay   = 24
	)
	var windows []string
	pendingLower := -1
	for _, item := range comparisons {
		if item.function != "hour" {
			continue
		}
		switch item.operator {
		case ">=", ">":
			pendingLower = item.value
		case "<=", "<":
			lower := pendingLower
			if lower < 0 {
				lower = startOfDay
			}
			windows = append(windows, formatHour(lower)+"-"+formatHour(item.value))
			pendingLower = -1
		}
	}
	// A trailing lower bound runs to the end of the day.
	if pendingLower >= 0 {
		windows = append(windows, formatHour(pendingLower)+"-"+formatHour(endOfDay))
	}
	sort.Strings(windows)
	return windows
}

func formatHour(hour int) string {
	if hour < 10 {
		return "0" + strconv.Itoa(hour) + ":00"
	}
	return strconv.Itoa(hour) + ":00"
}
