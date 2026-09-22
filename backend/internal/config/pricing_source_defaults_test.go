package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestPricingSourceDefaultsPointAtOurOwnCatalog guards this fork's decision to
// serve model pricing from our own repository instead of the upstream mirror.
// Reverting these defaults silently discards every price edit made in our
// catalog, because the remote catalog always outranks the bundled fallback file.
func TestPricingSourceDefaultsPointAtOurOwnCatalog(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()

	const wantPrefix = "https://raw.githubusercontent.com/Mxucc/model-price-repo/main/"

	for _, tc := range []struct {
		key    string
		suffix string
	}{
		{"pricing.remote_url", "model_prices_and_context_window.json"},
		{"pricing.hash_url", "model_prices_and_context_window.sha256"},
	} {
		got := viper.GetString(tc.key)
		if !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("%s = %q, want it to start with %q", tc.key, got, wantPrefix)
		}
		if !strings.HasSuffix(got, tc.suffix) {
			t.Errorf("%s = %q, want it to end with %q", tc.key, got, tc.suffix)
		}
	}
}
