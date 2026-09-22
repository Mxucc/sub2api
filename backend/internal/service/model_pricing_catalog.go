package service

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
)

// ModelPricingView is one row of the admin model price catalog.
//
// The page is a static, read-only view: it answers "what does this system
// charge for model X" for every model the price sources know about, whether or
// not a channel or account ever serves that model. Prices are the default
// price-card rates, before any group or channel override.
type ModelPricingView struct {
	Model string `json:"model"`
	// Provider is the upstream provider string from the catalog.
	Provider string `json:"provider,omitempty"`
	// Mode is the catalog's billing mode hint ("chat", "embedding", ...).
	Mode string `json:"mode,omitempty"`
	// Source reports where the base rates came from: "catalog" (the downloaded
	// price table / pricing.override_file) or "builtin" (the compiled-in table).
	Source string `json:"source"`

	// Per-token rates, exactly as the billing code consumes them.
	InputPricePerToken         float64 `json:"input_price_per_token"`
	OutputPricePerToken        float64 `json:"output_price_per_token"`
	CacheReadPricePerToken     float64 `json:"cache_read_price_per_token"`
	CacheCreationPricePerToken float64 `json:"cache_creation_price_per_token"`
	ImageInputPricePerToken    float64 `json:"image_input_price_per_token"`
	ImageOutputPricePerToken   float64 `json:"image_output_price_per_token"`

	// Per-million rates, the unit this project prices and displays in.
	InputPricePerMillion         float64 `json:"input_price_per_million"`
	OutputPricePerMillion        float64 `json:"output_price_per_million"`
	CacheReadPricePerMillion     float64 `json:"cache_read_price_per_million"`
	CacheCreationPricePerMillion float64 `json:"cache_creation_price_per_million"`

	// BillingExpr is the effective declarative expression governing this model
	// at the default price-card layer, or "" when the model bills from the rates
	// above.
	BillingExpr string `json:"billing_expr,omitempty"`
	// BillingExprSource is "catalog" or "builtin": the layer the expression text
	// came from, so an operator knows where to edit it.
	BillingExprSource string `json:"billing_expr_source,omitempty"`
	// Tiers is the parsed, human-readable form of BillingExpr. Empty when there
	// is no expression or the parser did not recognise its shape.
	Tiers []billingexpr.Tier `json:"tiers,omitempty"`
	// TimeDependent reports whether the expression's price changes with the
	// clock, which is what makes a model cheaper or dearer by the hour.
	TimeDependent bool `json:"time_dependent"`
	// ExprRecognized is false when an expression exists but the display parser
	// could not read it; the UI should then show the raw string.
	ExprRecognized bool `json:"expr_recognized"`

	// TokenPricingAbsent marks a catalog entry that carries no token rates (an
	// image-only model). Such an entry cannot price token traffic.
	TokenPricingAbsent bool `json:"token_pricing_absent"`
	// FallbackOnly marks a model priced only by the compiled-in table, i.e. one
	// the price catalog does not serve.
	FallbackOnly bool `json:"fallback_only"`
}

// ListModelPricingCatalog builds the admin price catalog: every model known to
// any price source, with the effective expression and its parsed tiers.
//
// It reads the in-memory price data, so it reflects whatever the last catalog
// sync loaded — no channel, account or traffic is needed for a model to appear.
func (s *BillingService) ListModelPricingCatalog() []ModelPricingView {
	names := s.modelPricingCatalogNames()
	// The list is assembled from maps; sorting here makes the API stable so the
	// page does not reshuffle between refreshes.
	sort.Strings(names)

	// One instant drives every lookup on the page so the rows stay mutually
	// consistent.
	pricingAt := timezone.Now()

	views := make([]ModelPricingView, 0, len(names))
	for _, model := range names {
		views = append(views, s.buildModelPricingView(model, pricingAt))
	}
	return views
}

// buildModelPricingView assembles one row.
//
// The catalog entry is read first and independently of the rate lookup: an entry
// can be listed without usable per-token rates (an image-only model, or one priced
// only by an expression), and the page still has to show its expression and its
// origin so an operator can tell which table to edit.
func (s *BillingService) buildModelPricingView(model string, pricingAt time.Time) ModelPricingView {
	view := ModelPricingView{Model: model}
	entry := s.catalogEntry(model)

	if entry != nil {
		view.Provider = entry.LiteLLMProvider
		view.Mode = entry.Mode
		view.Source = "catalog"
	} else {
		view.Source = "builtin"
		view.FallbackOnly = true
	}

	pricing, err := s.GetModelPricing(model)
	if err != nil || pricing == nil {
		// No usable token rates. Report whether the catalog says that is
		// intentional (an image-only entry) rather than inferring it from zero
		// rates, and still surface any expression so the row is not blank.
		view.TokenPricingAbsent = true
		if entry != nil {
			view.TokenPricingAbsent = entry.TokenPricingAbsent
		}
		view.BillingExpr, view.BillingExprSource = s.effectiveBillingExpr(model, nil, pricingAt)
		view.annotateBillingExpression()
		return view
	}

	view.InputPricePerToken = pricing.InputPricePerToken
	view.OutputPricePerToken = pricing.OutputPricePerToken
	view.CacheReadPricePerToken = pricing.CacheReadPricePerToken
	view.CacheCreationPricePerToken = pricing.CacheCreationPricePerToken
	view.ImageInputPricePerToken = pricing.ImageInputPricePerToken
	view.ImageOutputPricePerToken = pricing.ImageOutputPricePerToken
	view.InputPricePerMillion = pricing.InputPricePerToken * exprScale
	view.OutputPricePerMillion = pricing.OutputPricePerToken * exprScale
	view.CacheReadPricePerMillion = pricing.CacheReadPricePerToken * exprScale
	view.CacheCreationPricePerMillion = pricing.CacheCreationPricePerToken * exprScale

	// Prefer the catalog's own flag: an entry that explicitly prices input and
	// output at zero is a free model, not a model with no token pricing, and the
	// two must not be conflated on a page an operator reads to debug charges.
	if entry != nil {
		view.TokenPricingAbsent = entry.TokenPricingAbsent
	} else {
		view.TokenPricingAbsent = pricing.InputPricePerToken == 0 && pricing.OutputPricePerToken == 0
	}

	view.BillingExpr, view.BillingExprSource = s.effectiveBillingExpr(model, pricing, pricingAt)
	view.annotateBillingExpression()
	return view
}

// annotateBillingExpression fills in the parsed, display-ready form of the
// expression. A parse failure or a shape the parser does not recognise leaves
// Tiers empty while keeping BillingExpr: the UI then shows the raw string instead
// of a table it cannot vouch for.
func (view *ModelPricingView) annotateBillingExpression() {
	if view.BillingExpr == "" {
		return
	}
	parsed, err := billingexpr.ParseTiers(view.BillingExpr)
	if err != nil || parsed == nil {
		return
	}
	view.Tiers = parsed.Tiers
	view.TimeDependent = parsed.TimeDependent
	view.ExprRecognized = parsed.Recognized
}

// effectiveBillingExpr resolves which expression governs a model and labels its
// origin, so the price page can tell an operator where to edit it.
//
// It delegates to resolveBillingExprWithSource, the single decision point the
// billing path also uses, so a surface can never show a rule that did not charge
// the request.
func (s *BillingService) effectiveBillingExpr(model string, pricing *ModelPricing, pricingAt time.Time) (expression, source string) {
	return resolveBillingExprWithSource(model, pricing, pricingAt)
}

// modelPricingCatalogNames unions every name a price source can price.
//
// The union matters because the sources overlap only partly: the catalog serves
// most models, the compiled-in table prices a few the catalog never heard of, and
// the built-in expression table names a couple more (deepseek-flash and its
// variants are reached by prefix fallback in the price-card lookup, so they never
// appear as a card key of their own). Listing only the first two would hide models
// that are actively billed.
func (s *BillingService) modelPricingCatalogNames() []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}

	if s.pricingService != nil {
		for _, name := range s.pricingService.ListModelNames() {
			add(name)
		}
	}
	for name := range s.fallbackPrices {
		add(name)
	}
	for name := range builtinModelBillingExpr {
		add(name)
	}
	return names
}

// catalogEntry returns the raw price-catalog entry for a model, or nil when the
// catalog does not serve it.
func (s *BillingService) catalogEntry(model string) *LiteLLMModelPricing {
	if s.pricingService == nil {
		return nil
	}
	return s.pricingService.GetCatalogEntry(model)
}

// DescribeModelBillingExpr returns the effective billing expression for a model
// along with its parsed, display-ready form.
//
// It is the read-only counterpart of the billing path: the same resolveBillingExpr
// decides which expression wins, so a surface can never show a different price
// rule than the one that actually charges the request.
func (s *BillingService) DescribeModelBillingExpr(model string, pricingAt time.Time) *PlazaBillingExpr {
	if pricingAt.IsZero() {
		pricingAt = timezone.Now()
	}
	pricing, err := s.GetModelPricing(model)
	if err != nil || pricing == nil {
		return nil
	}
	expression, source := s.effectiveBillingExpr(model, pricing, pricingAt)
	if expression == "" {
		return nil
	}
	described := &PlazaBillingExpr{Expression: expression, Source: source}
	if parsed, parseErr := billingexpr.ParseTiers(expression); parseErr == nil {
		described.Parsed = parsed
	}
	return described
}
