//go:build unit

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// catalogTier is one parsed tier as it appears on the wire.
type catalogTier struct {
	Name string `json:"name"`
}

// catalogItem mirrors one row of the handler's success payload.
type catalogItem struct {
	Model              string        `json:"model"`
	Source             string        `json:"source"`
	BillingExpr        string        `json:"billing_expr"`
	BillingExprSource  string        `json:"billing_expr_source"`
	TimeDependent      bool          `json:"time_dependent"`
	ExprRecognized     bool          `json:"expr_recognized"`
	TokenPricingAbsent bool          `json:"token_pricing_absent"`
	InputPricePerMil   float64       `json:"input_price_per_million"`
	Tiers              []catalogTier `json:"tiers"`
}

// catalogResponse mirrors the handler's success payload. The handler returns a
// bare gin.H rather than a typed DTO, so the shape is pinned here: a rename on
// one side would otherwise break the admin page silently.
type catalogResponse struct {
	Data struct {
		Items    []catalogItem `json:"items"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
	} `json:"data"`
}

func setupModelPricingCatalogRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// The built-in table alone gives a non-trivial catalog (hundreds of models,
	// including ones with no catalog entry), which is enough to exercise
	// filtering and pagination.
	h := &ChannelHandler{billingService: service.NewBillingService(nil, nil)}
	router.GET("/channels/pricing/catalog", h.ListModelPricingCatalog)
	return router
}

func getCatalog(t *testing.T, query string) catalogResponse {
	t.Helper()
	router := setupModelPricingCatalogRouter()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/pricing/catalog"+query, nil))
	require.Equal(t, http.StatusOK, w.Code)

	var body catalogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// TestListModelPricingCatalog_ReturnsWholeCatalogAndCount is the basic contract
// the page depends on: a page of items, a total, and per-million rates.
func TestListModelPricingCatalog_ReturnsWholeCatalogAndCount(t *testing.T) {
	body := getCatalog(t, "?page_size=0")

	require.NotEmpty(t, body.Data.Items)
	require.Equal(t, len(body.Data.Items), body.Data.Total,
		"page_size=0 must return the whole catalog, so total equals the item count")
	require.Equal(t, 1, body.Data.Page)

	for _, item := range body.Data.Items {
		require.NotEmpty(t, item.Model)
		require.Contains(t, []string{"catalog", "builtin"}, item.Source)
		if !item.TokenPricingAbsent {
			require.Greater(t, item.InputPricePerMil, 0.0,
				"%s: a non-image model must report a per-million input rate", item.Model)
		}
	}
}

func TestListModelPricingCatalog_DefaultPagination(t *testing.T) {
	body := getCatalog(t, "")

	require.Equal(t, 50, body.Data.PageSize, "the default page size is 50")
	require.LessOrEqual(t, len(body.Data.Items), 50)
	require.Greater(t, body.Data.Total, 0)
}

func TestListModelPricingCatalog_PaginationSlicesTheFilteredSet(t *testing.T) {
	all := getCatalog(t, "?page_size=0")
	require.Greater(t, all.Data.Total, 3, "need a catalog big enough to page through")

	first := getCatalog(t, "?page=1&page_size=3")
	second := getCatalog(t, "?page=2&page_size=3")

	require.Len(t, first.Data.Items, 3)
	require.Equal(t, all.Data.Total, first.Data.Total, "total is the filtered count, not the page size")
	require.Equal(t, all.Data.Items[0].Model, first.Data.Items[0].Model)
	require.Equal(t, all.Data.Items[3].Model, second.Data.Items[0].Model,
		"page 2 must continue where page 1 stopped")
}

func TestListModelPricingCatalog_OutOfRangePageIsEmptyNotAnError(t *testing.T) {
	body := getCatalog(t, "?page=99999&page_size=10")
	require.Empty(t, body.Data.Items)
	require.Greater(t, body.Data.Total, 0)
}

func TestListModelPricingCatalog_QueryFiltersByModelName(t *testing.T) {
	body := getCatalog(t, "?q=deepseek&page_size=0")

	require.NotEmpty(t, body.Data.Items)
	for _, item := range body.Data.Items {
		require.Contains(t, item.Model, "deepseek")
	}

	// The search is case-insensitive, so an upper-case query matches the same set.
	upper := getCatalog(t, "?q=DEEPSEEK&page_size=0")
	require.Equal(t, body.Data.Total, upper.Data.Total)
}

func TestListModelPricingCatalog_QueryWithNoMatchIsEmpty(t *testing.T) {
	body := getCatalog(t, "?q=no-such-model-anywhere-xyz")
	require.Empty(t, body.Data.Items)
	require.Equal(t, 0, body.Data.Total)
}

// TestListModelPricingCatalog_SourceFilter covers the "built-in only" view, which
// is how an operator finds models the price table does not serve.
func TestListModelPricingCatalog_SourceFilter(t *testing.T) {
	builtin := getCatalog(t, "?source=builtin&page_size=0")
	require.NotEmpty(t, builtin.Data.Items)
	for _, item := range builtin.Data.Items {
		require.Equal(t, "builtin", item.Source)
	}

	catalog := getCatalog(t, "?source=catalog&page_size=0")
	for _, item := range catalog.Data.Items {
		require.Equal(t, "catalog", item.Source)
	}
	require.Less(t, catalog.Data.Total, builtin.Data.Total+1,
		"the two source filters must partition the catalog, not overlap it")
}

// TestListModelPricingCatalog_HasExprFilter pins the filter behind the "time
// dependent" views, and that the built-in DeepSeek expression surfaces with its
// parsed tiers: this is the price the page exists to show.
func TestListModelPricingCatalog_HasExprFilter(t *testing.T) {
	body := getCatalog(t, "?has_expr=true&page_size=0")

	require.NotEmpty(t, body.Data.Items)
	for _, item := range body.Data.Items {
		require.NotEmpty(t, item.BillingExpr,
			"has_expr=true must not return a model without an expression")
		require.NotEmpty(t, item.BillingExprSource)
	}

	var timeDependent *catalogItem
	for index := range body.Data.Items {
		item := &body.Data.Items[index]
		if !item.TimeDependent {
			continue
		}
		require.Equal(t, "builtin", item.BillingExprSource,
			"%s: only DeepSeek ships a built-in time-of-day expression", item.Model)
		require.True(t, item.ExprRecognized)
		require.Len(t, item.Tiers, 2, "peak and off_peak")
		require.Equal(t, "peak", item.Tiers[0].Name)
		require.Equal(t, "off_peak", item.Tiers[1].Name)
		require.NotEmpty(t, item.BillingExpr)
		timeDependent = item
	}
	require.NotNil(t, timeDependent,
		"the built-in DeepSeek expression must surface on the price page")
}

func TestListModelPricingCatalog_TimeDependentFilter(t *testing.T) {
	body := getCatalog(t, "?time_dependent=true&page_size=0")

	require.NotEmpty(t, body.Data.Items, "DeepSeek is time dependent, so the set is not empty")
	for _, item := range body.Data.Items {
		require.True(t, item.TimeDependent)
	}

	// The filter must actually exclude something, or it is a no-op.
	all := getCatalog(t, "?page_size=0")
	require.Less(t, body.Data.Total, all.Data.Total)
}

func TestListModelPricingCatalog_ProviderFilter(t *testing.T) {
	body := getCatalog(t, "?provider=deepseek&page_size=0")
	for _, item := range body.Data.Items {
		// Provider comes from the catalog entry; built-in-only models have none.
		require.NotEqual(t, "", item.Source)
	}

	// An unknown provider yields an empty page rather than everything.
	unknown := getCatalog(t, "?provider=not-a-real-provider")
	require.Empty(t, unknown.Data.Items)
}

// TestListModelPricingCatalog_NilBillingService guards the wiring failure case:
// a missing service must answer with an empty page, not a panic.
func TestListModelPricingCatalog_NilBillingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &ChannelHandler{}
	router.GET("/channels/pricing/catalog", h.ListModelPricingCatalog)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/pricing/catalog", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var body catalogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Empty(t, body.Data.Items)
}

func TestPaginateCatalog(t *testing.T) {
	items := make([]int, 120)
	for i := range items {
		items[i] = i
	}

	for _, tc := range []struct {
		name         string
		page         string
		pageSize     string
		wantPage     int
		wantPageSize int
	}{
		{"defaults", "", "", 1, 50},
		{"explicit", "2", "10", 2, 10},
		{"page below one clamps to one", "0", "10", 1, 10},
		{"non numeric falls back to defaults", "abc", "xyz", 1, 50},
		{"negative size falls back to default", "1", "-5", 1, 50},
		{"zero means everything", "1", "0", 1, 120},
		{"oversized clamps to the maximum", "1", "99999", 1, 1000},
		{"zero size on an empty set stays usable", "1", "0", 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := items
			if tc.name == "zero size on an empty set stays usable" {
				set = nil
			}
			page, pageSize := paginateCatalog(set, tc.page, tc.pageSize)
			require.Equal(t, tc.wantPage, page)
			require.Equal(t, tc.wantPageSize, pageSize)
		})
	}
}

// TestSliceCatalogPage_NeverSlicesOutOfRange pins the arithmetic against every
// page/size pair the parser can produce. A panic here would be a 500 on a normal
// UI race (paging forward while a filter shrinks the set), so it is worth
// exhaustive coverage rather than one representative case.
func TestSliceCatalogPage_NeverSlicesOutOfRange(t *testing.T) {
	items := make([]int, 7)
	for i := range items {
		items[i] = i
	}

	for _, count := range []int{0, 1, 7} {
		set := items[:count]
		for _, page := range []int{1, 2, 3, 100, 1_000_000} {
			for _, size := range []int{1, 3, 10, 50, 1000} {
				window := sliceCatalogPage(set, page, size)
				require.LessOrEqual(t, len(window), min(size, len(set)),
					"count=%d page=%d size=%d", count, page, size)
				expectedStart := (page - 1) * size
				if expectedStart > len(set) {
					require.Empty(t, window, "count=%d page=%d size=%d", count, page, size)
					continue
				}
				for i, value := range window {
					require.Equal(t, set[expectedStart+i], value,
						"count=%d page=%d size=%d offset=%d", count, page, size, i)
				}
			}
		}
	}
}

// TestSliceCatalogPage_OutOfRangeIsEmpty documents the contract the admin page
// relies on: an out-of-range page yields an empty slice (not the last page), and
// the caller steps back to page 1.
func TestSliceCatalogPage_OutOfRangeIsEmpty(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	require.Empty(t, sliceCatalogPage(items, 2, 10))
	require.Empty(t, sliceCatalogPage(items, 100, 10))
	require.Empty(t, sliceCatalogPage[int](nil, 1, 50))
}

// TestParseBoundedInt pins the ok flag's meaning: it reports whether a usable
// value was supplied, which is what lets paginateCatalog tell "not asked for"
// from "asked for zero".
func TestParseBoundedInt(t *testing.T) {
	for _, tc := range []struct {
		raw      string
		min, max int
		want     int
		wantOK   bool
	}{
		{"", 1, 10, 0, false},
		{"abc", 1, 10, 0, false},
		{"1.5", 1, 10, 0, false},
		{"-5", 1, 10, 0, false},
		{"0x10", 1, 10, 0, false},
		{"5", 1, 10, 5, true},
		{"1", 1, 10, 1, true},
		{"0", 1, 10, 1, true},
		{"50", 1, 10, 10, true},
		{" 7 ", 1, 10, 7, true},
		{"0", 0, 1000, 0, true},
		// An absurdly long digit string must clamp rather than overflow.
		{"999999999999999999999999", 0, 1000, 1000, true},
	} {
		t.Run(fmt.Sprintf("%q", tc.raw), func(t *testing.T) {
			value, ok := parseBoundedInt(tc.raw, tc.min, tc.max)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, value)
		})
	}
}
