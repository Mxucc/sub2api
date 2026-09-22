package billingexpr

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/expr-lang/expr/vm"
)

// maxCacheSize bounds the compiled-program cache. Reaching it drops the whole
// map rather than evicting individual entries: expressions change rarely, so a
// cold rebuild is cheaper than maintaining an LRU and no stale entry can
// survive a flush.
const maxCacheSize = 256

// ParseExprVersion splits a `v1:` tag from the expression body. Expressions
// without a tag are version 1, matching new-api.
func ParseExprVersion(expression string) (version int, body string) {
	if strings.HasPrefix(expression, versionPrefix) {
		return 1, expression[len(versionPrefix):]
	}
	return DefaultExprVersion, expression
}

// compileEnvPrototypeV1 is the type-checking prototype. Every token variable is
// a float64 count.
//
// The numeric functions declare `any` rather than float64 on purpose. expr's
// type checker accepts an int where a float64 is declared, but the VM then calls
// the function through reflection with the int unconverted, so a natural
// expression such as `tier("t", hour(tz))` would compile and then fail at run
// time. Declaring `any` and coercing keeps int-returning helpers usable; a
// non-numeric argument is reported as an error instead of being charged.
var compileEnvPrototypeV1 = map[string]any{
	VarP:      float64(0),
	VarC:      float64(0),
	VarLen:    float64(0),
	VarCR:     float64(0),
	VarCC:     float64(0),
	VarCC1h:   float64(0),
	VarImg:    float64(0),
	VarImgCR:  float64(0),
	VarAI:     float64(0),
	VarAO:     float64(0),
	"tier":    func(string, any) float64 { return 0 },
	"hour":    func(string) int { return 0 },
	"minute":  func(string) int { return 0 },
	"weekday": func(string) int { return 0 },
	"month":   func(string) int { return 0 },
	"day":     func(string) int { return 0 },
	"max":     func(any, any) float64 { return 0 },
	"min":     func(any, any) float64 { return 0 },
	"abs":     func(any) float64 { return 0 },
	"ceil":    func(any) float64 { return 0 },
	"floor":   func(any) float64 { return 0 },
}

func getCompileEnv(int) map[string]any { return compileEnvPrototypeV1 }

type cachedEntry struct {
	prog     *vm.Program
	usedVars map[string]bool
	version  int
}

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]*cachedEntry, 64)
)

// CompileFromCache compiles an expression, reusing the cached program when the
// same string was compiled before. The cache is keyed by the expression digest,
// so a changed expression is always a cache miss and never returns a stale
// program.
func CompileFromCache(expression string) (*vm.Program, error) {
	entry, err := compileEntry(expression, ExprHashString(expression))
	if err != nil {
		return nil, err
	}
	return entry.prog, nil
}

func compileEntry(expression, hash string) (*cachedEntry, error) {
	cacheMu.RLock()
	if entry, ok := cache[hash]; ok {
		cacheMu.RUnlock()
		return entry, nil
	}
	cacheMu.RUnlock()

	version, body := ParseExprVersion(expression)
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("expr compile error: expression is empty")
	}
	// Parse before compiling so a syntax error reports the parser's message
	// rather than a generic runtime failure.
	if _, err := parser.Parse(body); err != nil {
		return nil, fmt.Errorf("expr compile error: %w", err)
	}

	prog, err := expr.Compile(body, expr.Env(getCompileEnv(version)), expr.AsFloat64())
	if err != nil {
		return nil, fmt.Errorf("expr compile error: %w", err)
	}

	entry := &cachedEntry{
		prog:     prog,
		usedVars: extractUsedVars(prog),
		version:  version,
	}

	cacheMu.Lock()
	if len(cache) >= maxCacheSize {
		cache = make(map[string]*cachedEntry, 64)
	}
	cache[hash] = entry
	cacheMu.Unlock()

	return entry, nil
}

// extractUsedVars collects the identifiers an expression references. Callers
// use it to decide which token sub-categories to split out of p/c, so this runs
// once per expression instead of once per request.
func extractUsedVars(prog *vm.Program) map[string]bool {
	vars := make(map[string]bool)
	if prog == nil {
		return vars
	}
	ast.Find(prog.Node(), func(node ast.Node) bool {
		if id, ok := node.(*ast.IdentifierNode); ok {
			vars[id.Value] = true
		}
		return false
	})
	return vars
}

// UsedVars returns the set of identifiers an expression references, or nil when
// the expression is empty or fails to compile.
func UsedVars(expression string) map[string]bool {
	if strings.TrimSpace(expression) == "" {
		return nil
	}
	entry, err := compileEntry(expression, ExprHashString(expression))
	if err != nil {
		return nil
	}
	return entry.usedVars
}

// InvalidateCache clears the compiled-program cache. Each entry is dropped
// safely by hash keying, so this exists for tests and for an explicit reset.
func InvalidateCache() {
	cacheMu.Lock()
	cache = make(map[string]*cachedEntry, 64)
	cacheMu.Unlock()
}

// Validate compiles an expression and evaluates it against smoke vectors,
// rejecting non-finite or negative results. It is the guard for expressions
// arriving from configuration, where a bad coefficient would otherwise surface
// as a wrong charge on live traffic.
//
// Every vector is evaluated at more than one instant. A conditional price only
// executes the branch its condition selects, so evaluating at a single instant
// would leave the other branches unchecked — a wrong coefficient in the branch
// that did not run would pass validation and then mis-charge later in the day.
func Validate(expression string) error {
	if strings.TrimSpace(expression) == "" {
		return fmt.Errorf("expr validate error: expression is empty")
	}
	if _, err := CompileFromCache(expression); err != nil {
		return err
	}
	vectors := []TokenParams{
		{},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
		{P: 300, C: 100, Len: 1000, CR: 100, Img: 400, ImgCR: 200},
		{P: 800, C: 50, Len: 1000, AI: 200, AO: 50, CC: 20, CC1h: 10},
	}
	for _, at := range smokeVectorInstants {
		for _, vector := range vectors {
			result, _, err := Run(expression, vector, at)
			if err != nil {
				return fmt.Errorf("expr validate error: %w", err)
			}
			if math.IsNaN(result) || math.IsInf(result, 0) || result < 0 {
				return fmt.Errorf("expr validate error: result must be finite and non-negative, got %v", result)
			}
		}
	}
	return nil
}
