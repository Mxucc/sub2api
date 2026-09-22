//go:build unit

package service

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// overrideTestCatalogJSON 是一个小目录：三个可微调模型（其中 qwen-max 用来说明「目录
// 里还没有条目的模型会被补进来」）、一个带 billing_expr 的模型（deepseek-flash：它的
// 单价字段会被刷新搬运，表达式**不会**——见 TestRefreshOverrides_KeepsTunedAndRefreshesSynced）、
// 一个只带按张价（output_cost_per_image，已在白名单内）的图片模型。它还带一个非白名单
// 字段（context_window），用来证明刷新只搬运白名单字段。
const overrideTestCatalogJSON = `{
	"gpt-5.5": {"litellm_provider": "openai", "mode": "chat",
		"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
		"context_window": 400000},
	"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 2.8e-07, "output_cost_per_token": 4.2e-07,
		"billing_expr": "tier(\"base\", p * 2 + c * 4)"},
	"qwen-max": {"litellm_provider": "qwen", "mode": "chat",
		"input_cost_per_token": 1.2e-06, "output_cost_per_token": 6e-06},
	"image-only": {"litellm_provider": "openai", "mode": "image_generation",
		"output_cost_per_image": 0.04}
}`

// newOverrideTestService 造一个只带覆盖文件配置的价格服务。目录（DataDir）是空的
// TempDir：写入后的自动重载会因目录文件缺失而失败，那只告警、不影响写入语义。
func newOverrideTestService(t *testing.T) (*PricingService, string, string) {
	t.Helper()
	dir := t.TempDir()
	svc := &PricingService{cfg: &config.Config{}}
	svc.cfg.Pricing.DataDir = dir
	overridePath := filepath.Join(dir, "model_pricing_overrides.json")
	svc.cfg.Pricing.OverrideFile = overridePath
	return svc, dir, overridePath
}

// writeOverrideCatalog 写目录文件，让「按最新目录刷新」有输入。
func writeOverrideCatalog(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model_pricing.json"), []byte(body), 0o644))
}

// readOverrideFileRaw 把覆盖文件读成原始条目表（元数据键一起）。
func readOverrideFileRaw(t *testing.T, path string) map[string]map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var entries map[string]map[string]any
	require.NoError(t, json.Unmarshal(body, &entries), "覆盖文件必须是合法 JSON 对象")
	return entries
}

func fieldValue(t *testing.T, entry map[string]any, name string) float64 {
	t.Helper()
	value, ok := entry[name]
	require.Truef(t, ok, "entry must carry %s: %v", name, entry)
	number, ok := value.(float64)
	require.Truef(t, ok, "%s must be a number, got %T", name, value)
	return number
}

// TestOverrideSetEntry_WritesMetadataAndFields 固定写入形状：_tuned / _updated_at /
// 价格字段都要落盘，并且能原样读回。
func TestOverrideSetEntry_WritesMetadataAndFields(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	entry, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{"input_cost_per_token": 1.5e-7})
	require.NoError(t, err)
	require.Equal(t, "deepseek-flash", entry.Model)
	require.True(t, entry.Tuned)
	require.InDelta(t, 1.5e-7, entry.Fields["input_cost_per_token"], 1e-15)

	raw := readOverrideFileRaw(t, path)
	require.Len(t, raw, 1)
	stored := raw["deepseek-flash"]
	require.Equal(t, true, stored[overrideTunedKey])
	updatedAt, ok := stored[overrideUpdatedAtKey].(string)
	require.True(t, ok, "_updated_at must be a string")
	parsed, err := time.Parse(time.RFC3339, updatedAt)
	require.NoError(t, err, "_updated_at must be RFC3339")
	require.WithinDuration(t, time.Now().UTC(), parsed, 5*time.Minute)
	require.InDelta(t, 1.5e-7, fieldValue(t, stored, "input_cost_per_token"), 1e-15)

	entries, err := svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "deepseek-flash", entries[0].Model)
	require.True(t, entries[0].Tuned)
	require.Equal(t, updatedAt, entries[0].UpdatedAt)
	require.InDelta(t, 1.5e-7, entries[0].Fields["input_cost_per_token"], 1e-15)
}

// TestOverrideSetEntry_UnknownFieldRejectedWithoutTouchingFile：白名单外的字段必须
// 400 级报错并指出字段名，且不能改动文件——静默忽略会让管理员以为改生效了。
func TestOverrideSetEntry_UnknownFieldRejectedWithoutTouchingFile(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	_, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{"input_cost_per_token": 1e-7})
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	_, err = svc.SetOverrideEntry("deepseek-flash", map[string]any{
		"input_cost_per_token":               1e-7,
		"long_context_input_token_threshold": 0,
	})
	require.Error(t, err)
	require.True(t, IsOverrideValidationError(err), "未知字段是调用方的错误，应回 400")
	require.Contains(t, err.Error(), "long_context_input_token_threshold")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "被拒绝的写入不得改动文件")

	// 未知字段单独出现时同样必须拒绝，而不是当作空条目写进去。
	_, err = svc.SetOverrideEntry("deepseek-flash", map[string]any{"context_window": 1000})
	require.Error(t, err)
	require.Contains(t, err.Error(), "context_window")
}

// TestOverrideSetEntry_RejectsInvalidValues 覆盖负数、NaN、字符串、空补丁、空模型名。
func TestOverrideSetEntry_RejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{"negative", map[string]any{"input_cost_per_token": -1e-7}, "non-negative"},
		{"nan", map[string]any{"input_cost_per_token": math.NaN()}, "finite"},
		{"positive-inf", map[string]any{"output_cost_per_token": math.Inf(1)}, "finite"},
		{"string-number", map[string]any{"input_cost_per_token": "1e-7"}, "non-negative number"},
		{"bool", map[string]any{"cache_read_input_token_cost": true}, "non-negative number"},
		{"nested-object", map[string]any{"input_cost_per_token": map[string]any{"v": 1.0}}, "non-negative number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, path := newOverrideTestService(t)
			_, err := svc.SetOverrideEntry("deepseek-flash", tc.fields)
			require.Error(t, err)
			require.True(t, IsOverrideValidationError(err))
			require.Contains(t, err.Error(), tc.want)
			require.NoFileExists(t, path, "非法取值不得留下半成品文件")
		})
	}

	svc, _, path := newOverrideTestService(t)
	_, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one price field")

	_, err = svc.SetOverrideEntry("   ", map[string]any{"input_cost_per_token": 1e-7})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")
	require.NoFileExists(t, path)
}

// TestOverrideSetEntry_BillingExprMustValidate：billing_expr 允许改，但必须是合法
// 表达式（内置 DeepSeek 那种表达式被价表接管的前提）。
func TestOverrideSetEntry_BillingExprMustValidate(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	_, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{"billing_expr": "tier("})
	require.Error(t, err)
	require.Contains(t, err.Error(), "billing_expr")
	require.NoFileExists(t, path)

	_, err = svc.SetOverrideEntry("deepseek-flash", map[string]any{"billing_expr": 42.0})
	require.Error(t, err)
	require.Contains(t, err.Error(), "billing_expr")

	const expr = `len <= 272000 ? tier("standard", p * 10 + c * 50) : tier("long", p * 20 + c * 75)`
	_, err = svc.SetOverrideEntry("deepseek-flash", map[string]any{"billing_expr": expr})
	require.NoError(t, err)

	stored := readOverrideFileRaw(t, path)["deepseek-flash"]
	require.Equal(t, expr, stored[overrideExprKey])
}

// TestOverrideSetEntry_NullValueClearsTheTune：管理端传 null 表示「不再微调这个字段」，
// 不是「把这个字段从价格表里删掉」。
//
// 这个区分是安全性的关键：覆盖文件的浅合并会把 null 当成删除字段，而缺失的输入/输出
// 单价在解析时被读成 0 —— 那样清空一个输入框就会把模型变成免费，而不是回到价表价。
func TestOverrideSetEntry_NullValueClearsTheTune(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	// 先微调两个字段。
	_, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{
		"input_cost_per_token":  5e-06,
		"output_cost_per_token": 2e-05,
	})
	require.NoError(t, err)

	// 再把输出那一项撤掉（等价于界面里清空该输入框）。
	entry, err := svc.SetOverrideEntry("deepseek-flash", map[string]any{"input_cost_per_token": 5e-06})
	require.NoError(t, err)
	_, present := entry.Fields["output_cost_per_token"]
	require.False(t, present, "撤掉的字段不应出现在返回的条目里")
	require.InDelta(t, 5e-06, entry.Fields["input_cost_per_token"], 1e-21)

	stored := readOverrideFileRaw(t, path)["deepseek-flash"]
	_, rawPresent := stored["output_cost_per_token"]
	require.False(t, rawPresent, "撤掉的字段不应作为 null 写进文件")

	entries, err := svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	_, listedPresent := entries[0].Fields["output_cost_per_token"]
	require.False(t, listedPresent)

	// 端到端：撤掉的字段回到目录值，仍被微调的字段用微调值。
	catalog := `{"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 1e-07, "output_cost_per_token": 2e-06}}`
	data, err := svc.parsePricingData([]byte(catalog))
	require.NoError(t, err)
	require.InDelta(t, 5e-06, data["deepseek-flash"].InputCostPerToken, 1e-15)
	require.InDelta(t, 2e-06, data["deepseek-flash"].OutputCostPerToken, 1e-15,
		"撤掉的字段必须回到目录价，而不是变成 0")
}

// TestOverrideFile_NullStillRemovesFieldByHand 保留覆盖文件本身的破坏性语义：手写文件时
// null 依然表示「把这个字段从价格表条目里删掉」。
//
// 管理端接口刻意不用这条语义（见上一个用例），但手工维护文件的人仍然需要它：有时目录
// 带了一个我们不想让上游决定的字段，就得把它去掉。
func TestOverrideFile_NullStillRemovesFieldByHand(t *testing.T) {
	svc, _, path := newOverrideTestService(t)
	require.NoError(t, os.WriteFile(path, []byte(
		`{"deepseek-flash": {"output_cost_per_token": null}}`), 0o644))

	catalog := `{"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 1e-07, "output_cost_per_token": 2e-06}}`
	data, err := svc.parsePricingData([]byte(catalog))
	require.NoError(t, err)
	require.InDelta(t, 1e-7, data["deepseek-flash"].InputCostPerToken, 1e-15)
	require.Zero(t, data["deepseek-flash"].OutputCostPerToken,
		"手写 null 仍然删掉该字段")
}

// TestOverrideDeleteEntry：存在返回 true，不存在返回 false 且不报错、不建文件。
func TestOverrideDeleteEntry(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	removed, err := svc.DeleteOverrideEntry("deepseek-flash")
	require.NoError(t, err)
	require.False(t, removed)
	require.NoFileExists(t, path, "删除不存在的条目不该顺手建出文件")

	_, err = svc.SetOverrideEntry("deepseek-flash", map[string]any{"input_cost_per_token": 1e-7})
	require.NoError(t, err)
	_, err = svc.SetOverrideEntry("gpt-5.5", map[string]any{"output_cost_per_token": 1e-6})
	require.NoError(t, err)

	removed, err = svc.DeleteOverrideEntry("deepseek-flash")
	require.NoError(t, err)
	require.True(t, removed)

	entries, err := svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gpt-5.5", entries[0].Model)
	require.Len(t, readOverrideFileRaw(t, path), 1)
}

// TestOverrideWrite_FileIsValidJSONAndAtomic：写入后文件必须是干净 JSON、0644，
// 目录里不能留下临时文件（热重载会读到半截 JSON，所以必须写临时文件再 rename）。
func TestOverrideWrite_FileIsValidJSONAndAtomic(t *testing.T) {
	svc, dir, path := newOverrideTestService(t)

	models := []string{"gpt-5.5", "vendor/model.with.dots", `quote"model`}
	for index, model := range models {
		_, err := svc.SetOverrideEntry(model, map[string]any{
			"input_cost_per_token": float64(index+1) * 1e-7,
			"billing_expr":         `tier("base", p * 2 + c * 4)`,
		})
		require.NoError(t, err)
	}

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.Len(t, parsed, len(models))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, dirEntry := range dirEntries {
		require.False(t, strings.HasSuffix(dirEntry.Name(), ".tmp"),
			"写入完成后不得留下临时文件: %s", dirEntry.Name())
	}
}

// TestRefreshOverrides_KeepsTunedAndRefreshesSynced：默认刷新只更新 _tuned=false 的
// 条目、补上目录新模型，手调过的条目原样保留，并且只搬运白名单字段——唯一的例外是
// billing_expr：它在白名单里（覆盖文件允许承载），但刷新刻意不抄（见断言处的理由）。
func TestRefreshOverrides_KeepsTunedAndRefreshesSynced(t *testing.T) {
	svc, dir, path := newOverrideTestService(t)
	writeOverrideCatalog(t, dir, overrideTestCatalogJSON)

	require.NoError(t, os.WriteFile(path, []byte(`{
		"gpt-5.5": {"input_cost_per_token": 9e-09, "_tuned": true, "_updated_at": "2026-01-01T00:00:00Z"},
		"deepseek-flash": {"input_cost_per_token": 1e-09, "_tuned": false, "_updated_at": "2026-01-01T00:00:00Z"},
		"ghost-model": {"input_cost_per_token": 4e-09}
	}`), 0o644))

	result, err := svc.RefreshOverrides(false)
	require.NoError(t, err)
	require.Equal(t, 2, result.Added, "qwen-max 与 image-only 在目录里但覆盖文件里还没有条目，应被补进来")
	require.Equal(t, 1, result.Updated, "deepseek-flash 是上一轮同步来的（_tuned=false），应被刷新")
	require.Equal(t, 2, result.Kept, "手调过的 gpt-5.5 + 目录里没有的 ghost-model")
	require.Equal(t, 5, result.Total)

	raw := readOverrideFileRaw(t, path)
	require.Len(t, raw, 5)
	require.InDelta(t, 1.2e-6, fieldValue(t, raw["qwen-max"], "input_cost_per_token"), 1e-15)
	require.Equal(t, false, raw["qwen-max"][overrideTunedKey])

	require.InDelta(t, 9e-9, fieldValue(t, raw["gpt-5.5"], "input_cost_per_token"), 1e-18,
		"手调过的条目必须原样保留")
	require.Equal(t, true, raw["gpt-5.5"][overrideTunedKey])
	require.Equal(t, "2026-01-01T00:00:00Z", raw["gpt-5.5"][overrideUpdatedAtKey])

	require.InDelta(t, 2.8e-7, fieldValue(t, raw["deepseek-flash"], "input_cost_per_token"), 1e-15,
		"_tuned=false 的条目必须刷新成目录价")
	require.Equal(t, false, raw["deepseek-flash"][overrideTunedKey])
	// 表达式**不**随刷新抄进覆盖文件：抄一份等于固化一份会与代码/价格仓库漂移的快照
	// （目录后来改了规则，覆盖文件里的旧表达式仍按旧规则计费，且没有任何地方提示它过期）。
	// 覆盖文件要声明表达式，只能由管理员显式写入。
	_, exprCopied := raw["deepseek-flash"][overrideExprKey]
	require.False(t, exprCopied, "「获取最新价格」不得把 billing_expr 抄进覆盖文件")

	require.InDelta(t, 4e-9, fieldValue(t, raw["ghost-model"], "input_cost_per_token"), 1e-18,
		"目录里没有的条目不该被刷新动作丢掉")

	// output_cost_per_image 已在白名单内：图片模型应被刷新进来，且只带按张价。
	imageOnly, hasImageOnly := raw["image-only"]
	require.True(t, hasImageOnly, "output_cost_per_image 已在白名单内，图片模型应被刷新进来")
	require.InDelta(t, 0.04, fieldValue(t, imageOnly, "output_cost_per_image"), 1e-15)

	// 只写白名单字段：上下文窗口之类不得被复制进覆盖文件。
	for _, entry := range raw {
		_, hasContext := entry["context_window"]
		require.False(t, hasContext, "覆盖文件不该成为目录条目的影子副本: %v", entry)
	}
}

// TestRefreshOverrides_OverwriteTunedRewritesTunedEntries：overwrite_tuned=true 时
// 连手调过的条目也回到目录价。
func TestRefreshOverrides_OverwriteTunedRewritesTunedEntries(t *testing.T) {
	svc, dir, path := newOverrideTestService(t)
	writeOverrideCatalog(t, dir, overrideTestCatalogJSON)

	require.NoError(t, os.WriteFile(path, []byte(`{
		"gpt-5.5": {"input_cost_per_token": 9e-09, "_tuned": true}
	}`), 0o644))

	result, err := svc.RefreshOverrides(true)
	require.NoError(t, err)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, 3, result.Added, "deepseek-flash / qwen-max / image-only 都在目录里且文件里没有条目")
	require.Equal(t, 0, result.Kept)
	require.Equal(t, 4, result.Total)

	raw := readOverrideFileRaw(t, path)
	require.InDelta(t, 5e-6, fieldValue(t, raw["gpt-5.5"], "input_cost_per_token"), 1e-15,
		"overwrite_tuned=true 必须把手调价改回目录价")
	require.Equal(t, false, raw["gpt-5.5"][overrideTunedKey])
}

// TestRefreshOverrides_MissingCatalogIsAnError：目录还没下载时报明确错误，不 panic，
// 也不写坏覆盖文件。
func TestRefreshOverrides_MissingCatalogIsAnError(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	_, err := svc.RefreshOverrides(false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "model_pricing.json")
	require.NoFileExists(t, path)
}

// TestImportOverrideEntries_ReplaceAndMerge 覆盖两种导入模式与整体拒绝。
func TestImportOverrideEntries_ReplaceAndMerge(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	imported, err := svc.ImportOverrideEntries([]byte(`{
		"gpt-5.5": {"input_cost_per_token": 1e-06, "_tuned": false, "_updated_at": "2026-01-01T00:00:00Z"},
		"deepseek-flash": {"output_cost_per_token": 2e-06}
	}`), "replace")
	require.NoError(t, err)
	require.Equal(t, 2, imported)

	raw := readOverrideFileRaw(t, path)
	require.Len(t, raw, 2)
	require.InDelta(t, 1e-6, fieldValue(t, raw["gpt-5.5"], "input_cost_per_token"), 1e-18)
	require.Equal(t, true, raw["gpt-5.5"][overrideTunedKey], "导入的条目按手调对待")
	require.NotEmpty(t, raw["gpt-5.5"][overrideUpdatedAtKey])

	imported, err = svc.ImportOverrideEntries([]byte(`{"deepseek-flash": {"output_cost_per_token": 3e-06}}`), "merge")
	require.NoError(t, err)
	require.Equal(t, 1, imported)
	raw = readOverrideFileRaw(t, path)
	require.Len(t, raw, 2, "merge 必须保留未提及的条目")
	require.InDelta(t, 3e-6, fieldValue(t, raw["deepseek-flash"], "output_cost_per_token"), 1e-18)
	require.InDelta(t, 1e-6, fieldValue(t, raw["gpt-5.5"], "input_cost_per_token"), 1e-18)

	before, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = svc.ImportOverrideEntries([]byte(`{
		"ok-model": {"input_cost_per_token": 1e-07},
		"bad-model": {"context_window": 1000}
	}`), "replace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad-model", "错误里必须点出模型名")
	require.Contains(t, err.Error(), "context_window")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "整体拒绝时不得写一半")

	_, err = svc.ImportOverrideEntries([]byte(`[{"model": "x"}]`), "replace")
	require.Error(t, err)
	_, err = svc.ImportOverrideEntries([]byte(`{}`), "overwrite")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mode")
}

// TestExportOverrideFile_EmptyObjectWhenMissing：还没有任何微调时导出 `{}` 而不是错误，
// 「先导出再导入」因此可用；文件存在时导出原始字节。
func TestExportOverrideFile_EmptyObjectWhenMissing(t *testing.T) {
	svc, _, path := newOverrideTestService(t)

	body, err := svc.ExportOverrideFile()
	require.NoError(t, err)
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Empty(t, decoded)

	_, err = svc.SetOverrideEntry("gpt-5.5", map[string]any{"input_cost_per_token": 1e-6})
	require.NoError(t, err)
	body, err = svc.ExportOverrideFile()
	require.NoError(t, err)
	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, onDisk, body)
}

// TestOverridePatchIsMetadataOnly：只带元数据的条目不算「改价没生效」，不该打哨兵
// WARN（mergeOverrideOnlyModels 会跳过它们）。
func TestOverridePatchIsMetadataOnly(t *testing.T) {
	require.True(t, overridePatchIsMetadataOnly(json.RawMessage(`{"_tuned": true, "_updated_at": "2026-01-01T00:00:00Z"}`)))
	require.True(t, overridePatchIsMetadataOnly(json.RawMessage(`{}`)))
	require.False(t, overridePatchIsMetadataOnly(json.RawMessage(`{"_tuned": true, "input_cost_per_token": 1e-07}`)))
	require.False(t, overridePatchIsMetadataOnly(json.RawMessage(`[]`)))
}

// TestOverrideMetadataOnlyEntryIsSkippedByTheMerge：只带元数据的条目（管理端标了
// 「已微调」但价格字段全被删除）不能把未知模型带进目录，也不能影响它旁边的正常条目；
// mergeOverrideOnlyModels 会跳过它们，于是不会打那条「改价没生效」的 WARN。
func TestOverrideMetadataOnlyEntryIsSkippedByTheMerge(t *testing.T) {
	svc, _, path := newOverrideTestService(t)
	require.NoError(t, os.WriteFile(path, []byte(`{
		"ghost-model": {"_tuned": true, "_updated_at": "2026-01-01T00:00:00Z"},
		"gpt-5.5": {"input_cost_per_token": 1e-06, "_tuned": true}
	}`), 0o644))

	data, _, err := svc.buildPricingData([]byte(overrideTestCatalogJSON))
	require.NoError(t, err)
	_, ghost := data["ghost-model"]
	require.False(t, ghost, "只有元数据的条目不该被并入目录")
	require.InDelta(t, 1e-6, data["gpt-5.5"].InputCostPerToken, 1e-18)
	require.InDelta(t, 3e-5, data["gpt-5.5"].OutputCostPerToken, 1e-18, "未提及的字段保持目录值")

	// 但它仍要出现在管理端的列表里，否则管理员无法把它删掉。
	entries, err := svc.ListOverrideEntries()
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "ghost-model", entries[0].Model)
	require.True(t, entries[0].Tuned)
	require.Empty(t, entries[0].Fields)
}
