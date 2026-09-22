package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ListModelPricingCatalog returns the read-only model price catalog.
//
// The price page exists so an operator can see what the system charges for every
// model the price sources know about — including models no channel or account
// currently serves. Nothing here depends on traffic, so a newly synced model
// shows up immediately after the catalog reloads.
//
// GET /api/v1/admin/model-pricing?q=&provider=&source=&time_dependent=&page=&page_size=
func (h *ChannelHandler) ListModelPricingCatalog(c *gin.Context) {
	if h.billingService == nil {
		response.Success(c, gin.H{"items": []service.ModelPricingView{}, "total": 0, "page": 1, "page_size": 0})
		return
	}

	views := h.billingService.ListModelPricingCatalog()

	// Filtering happens in the handler rather than in the service: the catalog is
	// a few hundred rows and is rebuilt per request anyway, so pushing it through
	// SQL-style query objects would add indirection for no gain.
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	source := strings.ToLower(strings.TrimSpace(c.Query("source")))
	timeDependent := strings.ToLower(strings.TrimSpace(c.Query("time_dependent")))
	onlyWithExpr := strings.ToLower(strings.TrimSpace(c.Query("has_expr")))

	filtered := make([]service.ModelPricingView, 0, len(views))
	for _, view := range views {
		if query != "" && !strings.Contains(view.Model, query) {
			continue
		}
		if provider != "" && strings.ToLower(view.Provider) != provider {
			continue
		}
		if source != "" && view.Source != source {
			continue
		}
		if timeDependent == "true" && !view.TimeDependent {
			continue
		}
		if timeDependent == "false" && view.TimeDependent {
			continue
		}
		if onlyWithExpr == "true" && view.BillingExpr == "" {
			continue
		}
		filtered = append(filtered, view)
	}

	page, pageSize := paginateCatalog(filtered, c.Query("page"), c.Query("page_size"))
	items := sliceCatalogPage(filtered, page, pageSize)

	response.Success(c, gin.H{
		"items":     items,
		"total":     len(filtered),
		"page":      page,
		"page_size": pageSize,
	})
}

// sliceCatalogPage returns the requested window, or an empty slice when the page
// starts past the end of the result set.
//
// An out-of-range page is a normal race — the UI pages forward, then a filter
// shrinks the set before the next request lands — so it must answer with an empty
// page rather than slicing out of bounds. The caller detects it (empty items with
// a non-zero total) and steps back to page 1.
func sliceCatalogPage[T any](items []T, page, pageSize int) []T {
	start := (page - 1) * pageSize
	if start < 0 || start > len(items) {
		start = len(items)
	}
	return items[start:min(start+pageSize, len(items))]
}

// defaultCatalogPageSize is the page size used when the caller does not ask for
// one. The catalog is a few hundred rows, so the default keeps a response small
// while still filling a screen.
const defaultCatalogPageSize = 50

// maxCatalogPageSize bounds a caller-supplied page size so one request cannot ask
// for an unbounded response.
const maxCatalogPageSize = 1000

// paginateCatalog resolves the requested page window.
//
// page_size=0 is a distinct request for the whole catalog (the model plaza and
// export paths use it); it is honoured with the full length rather than the
// default, and page is forced back to 1 because paging through "everything" is
// meaningless. An absent or malformed value keeps the default, so a typo in the
// query string cannot silently widen the response.
func paginateCatalog[T any](items []T, rawPage, rawPageSize string) (page, pageSize int) {
	page = 1
	if parsed, ok := parseBoundedInt(rawPage, 1, 1_000_000); ok {
		page = parsed
	}

	pageSize = defaultCatalogPageSize
	if parsed, ok := parseBoundedInt(rawPageSize, 0, maxCatalogPageSize); ok {
		pageSize = parsed
	}

	if pageSize == 0 {
		pageSize = len(items)
		if pageSize == 0 {
			// A zero page size would make the slice arithmetic below divide by
			// zero; report a usable window instead.
			pageSize = 1
		}
		page = 1
	}
	return page, pageSize
}

// parseBoundedInt parses a plain non-negative decimal, clamped into
// [minimum, maximum].
//
// The ok flag is what lets a caller tell "not supplied" from "supplied as 0":
// returning a bare 0 for both would make appending "&page_size=" disable
// pagination instead of keeping the default.
func parseBoundedInt(raw string, minimum, maximum int) (value int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	for index := 0; index < len(raw); index++ {
		if raw[index] < '0' || raw[index] > '9' {
			return 0, false
		}
	}

	// Accumulate with saturation: the string can be arbitrarily long, and the
	// clamp is the answer either way.
	total := 0
	for index := 0; index < len(raw); index++ {
		if total > maximum {
			return maximum, true
		}
		total = total*10 + int(raw[index]-'0')
	}
	if total > maximum {
		return maximum, true
	}
	if total < minimum {
		return minimum, true
	}
	return total, true
}
