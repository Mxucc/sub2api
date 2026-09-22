//go:build unit

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOverrideDefaults_WriteReadAndHiddenFromModelList pins the two promises of the
// __defaults__ section: its prices are readable through the dedicated getter, and it
// never leaks into the per-model tuning list (where it would show up as a phantom model).
func TestOverrideDefaults_WriteReadAndHiddenFromModelList(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	entry, err := svc.SetOverrideEntry(OverrideDefaultsKey, map[string]any{
		"per_request_price": 0.5,
		"image_price_2k":    0.25,
	})
	require.NoError(t, err)
	require.Equal(t, OverrideDefaultsKey, entry.Model)
	require.True(t, entry.Tuned)
	require.InDelta(t, 0.5, entry.Fields["per_request_price"], 1e-12)

	price, ok := svc.DefaultOverridePrice("per_request_price")
	require.True(t, ok)
	require.InDelta(t, 0.5, price, 1e-12)
	price, ok = svc.DefaultOverridePrice("image_price_2k")
	require.True(t, ok)
	require.InDelta(t, 0.25, price, 1e-12)

	// 未微调的键 ok=false，包括不属于 defaults 白名单的 token 键。
	_, ok = svc.DefaultOverridePrice("video_price_480p")
	require.False(t, ok)
	_, ok = svc.DefaultOverridePrice("input_cost_per_token")
	require.False(t, ok)

	all := svc.DefaultOverridePrices()
	require.Len(t, all, 2)

	// 每模型列表不含 __defaults__。
	entries, err := svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Empty(t, entries, "__defaults__ 不该作为模型出现在列表里")

	// 再加一个模型条目，列表里只出现这个模型，defaults 仍留在文件里。
	_, err = svc.SetOverrideEntry("gpt-5.5", map[string]any{"input_cost_per_token": 1e-6})
	require.NoError(t, err)
	entries, err = svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gpt-5.5", entries[0].Model)

	raw := readOverrideFileRaw(t, path)
	require.Len(t, raw, 2)
	require.Contains(t, raw, OverrideDefaultsKey)
}

// TestOverrideDefaults_WhitelistIsSeparateFromModelFields pins the split between the
// two whitelists: a global default key must not be writable per model, and a token key
// must not be writable in __defaults__. Whitelisting a key nobody reads is a silently
// dead switch, which is why the two lists are kept apart.
func TestOverrideDefaults_WhitelistIsSeparateFromModelFields(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	_, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{"image_price_1k": 0.02})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err))
	require.Contains(t, err.Error(), "image_price_1k")

	_, err = svc.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"input_cost_per_token": 1e-6})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err))
	require.Contains(t, err.Error(), "input_cost_per_token")

	_, err = svc.SetOverrideEntry(OverrideDefaultsKey, map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one price field")

	require.NoFileExists(t, path, "被拒绝的写入不得留下文件")
}

// TestOverrideDefaults_SurviveRefresh checks that "refresh from catalog" keeps the
// defaults segment verbatim, and that the catalog patch comparison never sees it as a
// model to add or update.
func TestOverrideDefaults_SurviveRefresh(t *testing.T) {
	svc, dir, path := newOverrideTestService(t)
	writeOverrideCatalog(t, dir, overrideTestCatalogJSON)

	_, err := svc.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"web_search_price_per_call": 0.005})
	require.NoError(t, err)
	_, err = svc.SetOverrideEntry("gpt-5.5", map[string]any{"input_cost_per_token": 9e-9})
	require.NoError(t, err)

	result, err := svc.RefreshOverrides(false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Added, 1, "目录里的新模型仍应被补进来")

	price, ok := svc.DefaultOverridePrice("web_search_price_per_call")
	require.True(t, ok, "刷新不得丢掉 __defaults__")
	require.InDelta(t, 0.005, price, 1e-15)

	raw := readOverrideFileRaw(t, path)
	stored, ok := raw[OverrideDefaultsKey]
	require.True(t, ok)
	require.InDelta(t, 0.005, fieldValue(t, stored, "web_search_price_per_call"), 1e-15)
	// 目录补丁比较不会给 defaults 写 _tuned=false 之类的目录同步元数据。
	require.NotEqual(t, false, stored[overrideTunedKey])
}

// TestOverrideDefaults_ImportAcceptsAndDoesNotCountIt covers import: the file may carry a
// __defaults__ segment, it is validated with the defaults whitelist, and it does not count
// toward the "imported model entries" return value.
func TestOverrideDefaults_ImportAcceptsAndDoesNotCountIt(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	imported, err := svc.ImportOverrideEntries([]byte(`{
		"__defaults__": {"audio_stt_price_per_hour": 0.2},
		"gpt-5.5": {"input_cost_per_token": 1e-06}
	}`), "replace")
	require.NoError(t, err)
	require.Equal(t, 1, imported, "__defaults__ 不计入模型条数")

	price, ok := svc.DefaultOverridePrice("audio_stt_price_per_hour")
	require.True(t, ok)
	require.InDelta(t, 0.2, price, 1e-12)

	raw := readOverrideFileRaw(t, path)
	require.Len(t, raw, 2)

	// merge 模式自然保留 defaults。
	imported, err = svc.ImportOverrideEntries([]byte(`{"deepseek-flash": {"input_cost_per_token": 2e-06}}`), "merge")
	require.NoError(t, err)
	require.Equal(t, 1, imported)
	_, ok = svc.DefaultOverridePrice("audio_stt_price_per_hour")
	require.True(t, ok, "merge 必须保留 __defaults__")

	// 非法 defaults（token 键）整体拒绝且不写文件。
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = svc.ImportOverrideEntries([]byte(`{"__defaults__": {"input_cost_per_token": 1e-06}}`), "replace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "input_cost_per_token")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

// TestOverrideDefaults_NonNumericAndUnknownKeysIgnored pins the lenient read path: a
// hand-edited file may carry junk, but the billing path must only ever see whitelisted
// non-negative numbers — and must not fail.
func TestOverrideDefaults_NonNumericAndUnknownKeysIgnored(t *testing.T) {
	svc, _, path := newOverrideTestService(t)
	require.NoError(t, os.WriteFile(path, []byte(`{
		"__defaults__": {
			"per_request_price": 0.05,
			"video_price_480p": "0.1",
			"mystery_key": 1,
			"image_price_4k": -1
		}
	}`), 0o644))

	price, ok := svc.DefaultOverridePrice("per_request_price")
	require.True(t, ok)
	require.InDelta(t, 0.05, price, 1e-12)

	_, ok = svc.DefaultOverridePrice("video_price_480p")
	require.False(t, ok, "字符串不是合法价格，应被忽略")
	_, ok = svc.DefaultOverridePrice("mystery_key")
	require.False(t, ok)
	_, ok = svc.DefaultOverridePrice("image_price_4k")
	require.False(t, ok, "负数应被忽略")

	all := svc.DefaultOverridePrices()
	require.Len(t, all, 1)
}

// TestOverrideDefaults_CacheRefreshesWhenFileChanges proves the size+mtime fingerprint:
// the billing path caches, but a changed file is picked up on the very next read.
func TestOverrideDefaults_CacheRefreshesWhenFileChanges(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	_, err := svc.SetOverrideEntry(OverrideDefaultsKey, map[string]any{"per_request_price": 0.05})
	require.NoError(t, err)
	first, ok := svc.DefaultOverridePrice("per_request_price")
	require.True(t, ok)
	require.InDelta(t, 0.05, first, 1e-12)

	require.NoError(t, os.WriteFile(path, []byte(`{"__defaults__": {"per_request_price": 0.123456}}`), 0o644))
	second, ok := svc.DefaultOverridePrice("per_request_price")
	require.True(t, ok)
	require.InDelta(t, 0.123456, second, 1e-12, "文件变化后必须拿到新值")

	// 删掉整个 defaults 段：读取回到 ok=false。
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o644))
	_, ok = svc.DefaultOverridePrice("per_request_price")
	require.False(t, ok)
	require.Empty(t, svc.DefaultOverridePrices())
}

// TestOverrideDefaults_NilServiceIsSafe guards the hot billing path against a nil service.
func TestOverrideDefaults_NilServiceIsSafe(t *testing.T) {
	var svc *PricingService
	_, ok := svc.DefaultOverridePrice("per_request_price")
	require.False(t, ok)
	require.Empty(t, svc.DefaultOverridePrices())
}
