package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
)

// 覆盖补丁文件（pricing.override_file）里两个元数据键：条目是否被管理端手动微调过、
// 最后一次写入时间。价格解析器按已知字段名解 JSON，未知键被忽略，因此这两个键既能被
// 管理端「模型价格」页面展示，又不会影响计费解析链。
//
// 缺省语义：条目里没有 _tuned 键时视为 true——覆盖文件本来就允许手工编辑，没有元数据
// 的条目就是人写的，刷新时不能悄悄覆盖掉它。_tuned=false 只由本文件的刷新动作写入，
// 表示「这条是从目录同步来的，可以随目录更新」。
const (
	overrideTunedKey     = "_tuned"
	overrideUpdatedAtKey = "_updated_at"
	overrideExprKey      = "billing_expr"
	overrideFilePerm     = 0o644
)

// OverrideDefaultsKey 是覆盖文件里存放全局默认价的保留键。它不是一个模型名，
// 因此不会参与目录条目的合并，也不会出现在每模型微调列表里。
const OverrideDefaultsKey = "__defaults__"

// overridePriceFields 是允许手动微调的字段白名单。除此之外的字段名一律拒绝：
// 静默忽略会让管理员以为改生效了。
//
// 白名单刻意只收单价字段（加 billing_expr）。上下文窗口、provider、mode 之类不进覆盖
// 文件：覆盖文件是补丁而不是目录条目的副本，整条复制会立刻和目录失去同步。
//
// 每个键都必须是价格解析器真正读取的 JSON 字段名（见 LiteLLMRawEntry）。写进白名单
// 但没人读的键等于一个静默无效的开关，所以新增键之前先确认解析器认得它。
var overridePriceFields = map[string]struct{}{
	"input_cost_per_token":                      {},
	"output_cost_per_token":                     {},
	"input_cost_per_token_priority":             {},
	"output_cost_per_token_priority":            {},
	"cache_read_input_token_cost":               {},
	"cache_read_input_token_cost_priority":      {},
	"cache_creation_input_token_cost":           {},
	"cache_creation_input_token_cost_priority":  {},
	"cache_creation_input_token_cost_above_1hr": {},
	// 图片输入价在目录里叫 input_cost_per_image_token；它同时喂
	// ModelPricing.ImageInputPricePerToken，多模态图文不同价的模型靠它调价。
	"input_cost_per_image_token":  {},
	"output_cost_per_image_token": {},
	// output_cost_per_image 是按张计费（图片生成）的基准单价：解析器读它会填
	// ModelPricing.OutputCostPerImage，也就是 getDefaultImagePrice 的按张价基准。
	"output_cost_per_image": {},
	// billing_expr 是唯一的字符串字段（见 validateOverrideField）。它在白名单里表示
	// 「覆盖文件可以承载这个键」，从而让价格数据接管该模型的价与计费规则；它**不会**
	// 被「获取最新价格」从目录抄进覆盖文件（见 catalogOverridePatch 的理由）。
	overrideExprKey: {},
}

// overrideGlobalPriceFields 是 __defaults__ 段允许微调的全局默认价白名单。
//
// 它和 overridePriceFields 刻意分成两份，因为两者的消费路径完全不同：
// overridePriceFields 里的键会被写进 LiteLLM 目录条目、由价格解析器亲自读取；
// 而这里的键（按张 / 按秒 / 按次 / 搜索 / 音频）不会被解析器读取，而是由
// BillingService 各自的计费函数在「没有分组/渠道显式定价」时回落到全局默认值。
// 把合成键塞进 overridePriceFields 只会得到一个静默无效的开关——写进白名单但没人读。
//
// 键名与分组/渠道定价的 JSON 字段名保持一致，方便前后端与运维对照。
var overrideGlobalPriceFields = map[string]struct{}{
	"image_price_1k":                    {},
	"image_price_2k":                    {},
	"image_price_4k":                    {},
	"video_price_480p":                  {},
	"video_price_720p":                  {},
	"video_price_1080p":                 {},
	"web_search_price_per_call":         {},
	"search_price_per_1k":               {},
	"audio_realtime_price_per_min":      {},
	"audio_tts_price_per_million_chars": {},
	"audio_stt_price_per_hour":          {},
	"per_request_price":                 {},
}

// OverrideEntry 是覆盖文件里的一条记录：模型名 + 元数据 + 价格补丁。
// Fields 里值为 null 表示删除该字段（与覆盖文件的浅合并语义一致）。
type OverrideEntry struct {
	Model     string         `json:"model"`
	Tuned     bool           `json:"tuned"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Fields    map[string]any `json:"fields"`
}

// OverrideRefreshResult 汇总一次「按最新目录刷新覆盖文件」的结果。
type OverrideRefreshResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Kept    int `json:"kept"`
	Total   int `json:"total"`
}

// OverrideValidationError 表示调用方提供的模型名/字段不合法，HTTP 层应回 400。
type OverrideValidationError struct {
	message string
}

func (e *OverrideValidationError) Error() string { return e.message }

// IsOverrideValidationError 报告 err 是否由请求内容引起（可安全回显给管理员）。
func IsOverrideValidationError(err error) bool {
	var target *OverrideValidationError
	return errors.As(err, &target)
}

func newOverrideValidationError(format string, args ...any) error {
	return &OverrideValidationError{message: fmt.Sprintf(format, args...)}
}

// OverrideFilePath 返回覆盖补丁文件路径（未配置时为空串）。
func (s *PricingService) OverrideFilePath() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.Pricing.OverrideFile)
}

// ListOverrideEntries 读取覆盖文件。文件不存在（或未配置）返回空切片而不是错误：
// 「还没有任何微调」是合法状态，管理端页面要能正常打开。
// 条目按模型名排序，让页面在两次刷新之间保持稳定。
func (s *PricingService) ListOverrideEntries() ([]OverrideEntry, error) {
	entries, err := readOverrideEntries(s.OverrideFilePath())
	if err != nil {
		return nil, err
	}
	names := sortedOverrideKeys(entries)
	out := make([]OverrideEntry, 0, len(names))
	for _, name := range names {
		// __defaults__ 不是模型：把它当模型显示会让管理端多出一条无法计费的
		// 幽灵行。它的读写走专门的默认价接口。
		if name == OverrideDefaultsKey {
			continue
		}
		entry, _ := decodeOverrideEntry(name, entries[name])
		out = append(out, entry)
	}
	return out, nil
}

// SetOverrideEntry 写入/更新一条手动微调（_tuned=true）。字段为空或含有白名单外的
// 字段名时返回错误且不改动文件。
//
// model 为 OverrideDefaultsKey 时写入的是一段全局默认价，字段按 overrideGlobalPriceFields
// 校验；其余行为（_tuned / _updated_at、null 表示撤销）完全一致。
func (s *PricingService) SetOverrideEntry(model string, fields map[string]any) (OverrideEntry, error) {
	path, err := s.requireOverrideFilePath()
	if err != nil {
		return OverrideEntry{}, err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return OverrideEntry{}, newOverrideValidationError("model must not be empty")
	}
	if model == OverrideDefaultsKey {
		if err := validateOverrideDefaultsFields(fields); err != nil {
			return OverrideEntry{}, err
		}
	} else if err := validateOverrideFields(fields); err != nil {
		return OverrideEntry{}, err
	}

	// A null value means "stop tuning this field", so it is dropped from the patch
	// rather than written through. That distinction matters: writing null through
	// makes the merge delete the key from the price-table entry, and a missing
	// input or output rate is read as 0 — clearing the box would silently make the
	// model free instead of restoring the table's price.
	tuned := make(map[string]any, len(fields))
	for name, value := range fields {
		if value == nil {
			continue
		}
		tuned[name] = value
	}

	updatedAt := overrideTimestamp()
	patch, err := encodeOverridePatch(tuned, true, updatedAt)
	if err != nil {
		return OverrideEntry{}, err
	}

	entries, err := readOverrideEntries(path)
	if err != nil {
		return OverrideEntry{}, err
	}
	entries[model] = patch
	if err := writeOverrideFileAtomic(path, entries); err != nil {
		return OverrideEntry{}, err
	}
	s.reloadOverridesAfterWrite()

	// The response reports what the entry actually pins, so a client can tell a
	// cleared field from an untouched one.
	return OverrideEntry{
		Model:     model,
		Tuned:     true,
		UpdatedAt: updatedAt,
		Fields:    cloneOverrideFields(tuned),
	}, nil
}

// DeleteOverrideEntry 删除一条微调。条目本来就不存在时返回 (false, nil)：删除是幂等
// 动作，管理端重复点击不该报错，也不该顺手建出一个空文件。
func (s *PricingService) DeleteOverrideEntry(model string) (bool, error) {
	path, err := s.requireOverrideFilePath()
	if err != nil {
		return false, err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false, newOverrideValidationError("model must not be empty")
	}

	entries, err := readOverrideEntries(path)
	if err != nil {
		return false, err
	}
	if _, ok := entries[model]; !ok {
		return false, nil
	}
	delete(entries, model)
	if err := writeOverrideFileAtomic(path, entries); err != nil {
		return false, err
	}
	s.reloadOverridesAfterWrite()
	return true, nil
}

// DefaultOverridePrice 读取 __defaults__ 里的一个全局默认价。ok=false 表示未微调。
//
// 这个接口供计费路径每请求调用，因此不能每次读盘：缓存按覆盖文件的「大小 + mtime」
// 指纹失效，只有文件真的变了才重读。读取或解析失败时保留上一次成功的缓存（回退到
// 历史值），绝不让一次磁盘抖动把计费变成报错或 panic。
func (s *PricingService) DefaultOverridePrice(field string) (float64, bool) {
	if s == nil || s.cfg == nil {
		return 0, false
	}
	prices := s.defaultOverridePricesSnapshot()
	value, ok := prices[field]
	return value, ok
}

// ModelOverrideField 报告某个模型在覆盖文件里对 field 做了手动微调（只认白名单内的数值键）。
// ok=false 表示该字段没有被单独微调过。
//
// 它和 DefaultOverridePrice 共用同一份缓存与同一套指纹/失效逻辑：计费路径可能在同一请求里
// 既要问全局默认价又要问每模型微调，第二次读盘（以及第二个 mtime 判断）纯属浪费 —— 更糟的是
// 两次读取之间文件若被替换，两个答案会来自不同版本的文件。
//
// 只认白名单内的数值键：白名单外的键、非数值、null、以及 __defaults__ 自身都不算命中，
// 与「写进覆盖文件的价格必须真的会被解析器读到」这条原则一致。
func (s *PricingService) ModelOverrideField(model, field string) (float64, bool) {
	if s == nil || s.cfg == nil {
		return 0, false
	}
	model = strings.TrimSpace(model)
	if model == "" || model == OverrideDefaultsKey {
		return 0, false
	}
	fields := s.modelOverrideFieldsSnapshot()[model]
	value, ok := fields[field]
	return value, ok
}

// DefaultOverridePrices 返回 __defaults__ 的一份副本（管理端 / 调试用）。
func (s *PricingService) DefaultOverridePrices() map[string]float64 {
	out := map[string]float64{}
	if s == nil || s.cfg == nil {
		return out
	}
	for name, value := range s.defaultOverridePricesSnapshot() {
		out[name] = value
	}
	return out
}

// defaultOverridePricesSnapshot 返回最近的 __defaults__ 解析结果快照。
//
// 刷新在独立互斥锁下进行（不复用 s.mu：计费路径与目录重载没有锁序关系，混用会引入
// 死锁风险）。快照里的 map 一旦发布就不再原地修改，因此调用方在锁外读取是安全的。
func (s *PricingService) defaultOverridePricesSnapshot() map[string]float64 {
	s.defaultsMu.Lock()
	defer s.defaultsMu.Unlock()
	s.refreshDefaultOverrideCacheLocked()
	return s.defaultsCache
}

// modelOverrideFieldsSnapshot 返回最近的「每模型微调字段」解析结果快照，与
// defaultOverridePricesSnapshot 共用同一份缓存（同一次刷新、同一个指纹）。
func (s *PricingService) modelOverrideFieldsSnapshot() map[string]map[string]float64 {
	s.defaultsMu.Lock()
	defer s.defaultsMu.Unlock()
	s.refreshDefaultOverrideCacheLocked()
	return s.defaultsModelCache
}

// refreshDefaultOverrideCacheLocked 在指纹变化时重读覆盖文件。调用方必须持有 defaultsMu。
//
// 一次读取同时构建两份快照（__defaults__ 全局默认价、每模型微调字段），因此计费路径
// 无论问哪一边都只付一次磁盘 IO，且两份答案必然来自同一版本的文件。
//
// 只保留白名单内的数值键：非数值或白名单外的键被忽略（打一次 WARN），与读取路径
// 「宽容但明确」的取向一致——手写文件里的一个坏键不该让计费路径拿到脏值。
func (s *PricingService) refreshDefaultOverrideCacheLocked() {
	path := strings.TrimSpace(s.cfg.Pricing.OverrideFile)
	if path == "" {
		s.defaultsCacheKey = ""
		s.defaultsCache = nil
		s.defaultsModelCache = nil
		return
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		// 文件被删掉等于没有微调：立刻清空，而不是回退到上一次的缓存值。
		s.defaultsCache = map[string]float64{}
		s.defaultsModelCache = map[string]map[string]float64{}
		s.defaultsCacheKey = "missing"
		return
	}
	if err != nil {
		// 其它 stat 失败（权限等）：沿用上一次成功的缓存。
		return
	}
	key := fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
	if s.defaultsCache != nil && key == s.defaultsCacheKey {
		return
	}

	entries, err := readOverrideEntries(path)
	if err != nil {
		s.warnDefaultOverrideOnce("read failed: %v", err)
		return
	}

	s.defaultsCache = overrideDefaultsPriceSnapshot(s, entries[OverrideDefaultsKey])
	s.defaultsModelCache = overrideModelFieldSnapshots(s, entries)
	s.defaultsCacheKey = key
}

// overrideDefaultsPriceSnapshot 解析 __defaults__ 段为「字段 → 价格」。
func overrideDefaultsPriceSnapshot(s *PricingService, raw json.RawMessage) map[string]float64 {
	prices := map[string]float64{}
	if len(bytes.TrimSpace(raw)) == 0 {
		return prices
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		s.warnDefaultOverrideOnce("%s is not a JSON object", OverrideDefaultsKey)
		return prices
	}
	for name, value := range fields {
		if name == overrideTunedKey || name == overrideUpdatedAtKey {
			continue
		}
		if _, allowed := overrideGlobalPriceFields[name]; !allowed {
			s.warnDefaultOverrideOnce("%s ignored: not an adjustable default price field", name)
			continue
		}
		number, ok := overrideSnapshotNumber(value)
		if !ok {
			s.warnDefaultOverrideOnce("%s ignored: not a non-negative finite number", name)
			continue
		}
		prices[name] = number
	}
	return prices
}

// overrideModelFieldSnapshots 解析每个模型条目的白名单价格字段，跳过 __defaults__
// （它不是模型）与只带元数据的条目。非数值 / 负数 / null / 白名单外的键一律不收录。
func overrideModelFieldSnapshots(s *PricingService, entries map[string]json.RawMessage) map[string]map[string]float64 {
	models := make(map[string]map[string]float64, len(entries))
	for name, raw := range entries {
		if name == OverrideDefaultsKey {
			continue
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			s.warnDefaultOverrideOnce("model %q is not a JSON object", name)
			continue
		}
		priced := make(map[string]float64, len(fields))
		for field, value := range fields {
			if field == overrideTunedKey || field == overrideUpdatedAtKey {
				continue
			}
			if _, allowed := overridePriceFields[field]; !allowed {
				continue
			}
			number, ok := overrideSnapshotNumber(value)
			if !ok {
				continue
			}
			priced[field] = number
		}
		if len(priced) > 0 {
			models[name] = priced
		}
	}
	return models
}

// overrideSnapshotNumber 把缓存快照里的取值收成非负有限数字。字符串（"1e-7"）与 null
// 都不接受：前者是拼错的输入，后者表示「撤销该字段的微调」。
func overrideSnapshotNumber(value any) (float64, bool) {
	number, ok := overrideFieldNumber(value)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return 0, false
	}
	return number, true
}

// warnDefaultOverrideOnce 只打一次 WARN，避免坏键在每请求的计费路径上刷屏。
func (s *PricingService) warnDefaultOverrideOnce(format string, args ...any) {
	if s.defaultsWarned {
		return
	}
	s.defaultsWarned = true
	logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override cache: "+format, args...)
}

// RefreshOverrides 用最新下载的目录刷新覆盖文件。
//
// overwriteTuned=false（默认，管理端「获取最新价格」按钮）：只重写 _tuned=false 的条目
// （上一轮从目录同步来的），并补上目录里还没有条目的模型；管理员手调过的条目原样保留。
// overwriteTuned=true：连手调过的条目也按最新目录重写，等于「全部回到目录价」。
//
// 覆盖文件里那些目录根本没有的模型（例如只由内置兜底表定价的模型）在两种模式下都原样
// 保留——它们不在目录里不代表它们是垃圾，删掉就是数据丢失。
func (s *PricingService) RefreshOverrides(overwriteTuned bool) (OverrideRefreshResult, error) {
	var result OverrideRefreshResult
	path, err := s.requireOverrideFilePath()
	if err != nil {
		return result, err
	}

	catalogPath := s.getPricingFilePath()
	body, err := os.ReadFile(catalogPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, fmt.Errorf("pricing catalog %s is missing; sync the catalog first", catalogPath)
		}
		return result, fmt.Errorf("read pricing catalog %s: %w", catalogPath, err)
	}
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(body, &catalog); err != nil {
		return result, fmt.Errorf("parse pricing catalog %s: %w", catalogPath, err)
	}

	existing, err := readOverrideEntries(path)
	if err != nil {
		return result, err
	}
	// __defaults__ 不是模型条目：它不进目录补丁比较，只在最后原样放回结果里。
	// 保护是双重的——已经被过滤一次，这里再显式取出来，将来若有人改动过滤也丢不了它。
	defaultsRaw, hasDefaults := existing[OverrideDefaultsKey]
	delete(existing, OverrideDefaultsKey)

	// 只从目录条目里挑白名单字段。带上上下文窗口等无关字段会让覆盖文件在目录更新时
	// 变成一份过期的影子副本。
	patches := make(map[string]map[string]any, len(catalog))
	for name, raw := range catalog {
		if name == OverrideDefaultsKey {
			continue
		}
		if patch := catalogOverridePatch(raw); len(patch) > 0 {
			patches[name] = patch
		}
	}

	updatedAt := overrideTimestamp()
	names := make([]string, 0, len(patches)+len(existing))
	seen := make(map[string]struct{}, len(patches)+len(existing))
	for name := range patches {
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range existing {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)

	next := make(map[string]json.RawMessage, len(names)+1)
	if hasDefaults {
		next[OverrideDefaultsKey] = defaultsRaw
	}
	for _, name := range names {
		if name == OverrideDefaultsKey {
			continue
		}
		patch, inCatalog := patches[name]
		raw, present := existing[name]
		entry, _ := decodeOverrideEntry(name, raw)

		switch {
		case inCatalog && present && entry.Tuned && !overwriteTuned:
			next[name] = raw
			result.Kept++
		case inCatalog:
			encoded, encodeErr := encodeOverridePatch(patch, false, updatedAt)
			if encodeErr != nil {
				return OverrideRefreshResult{}, encodeErr
			}
			next[name] = encoded
			if present {
				result.Updated++
			} else {
				result.Added++
			}
		case present:
			// 目录里没有这个模型（拼错、仅兜底表、或目录暂时缺条目）：原样保留。
			next[name] = raw
			result.Kept++
		}
	}

	if err := writeOverrideFileAtomic(path, next); err != nil {
		return OverrideRefreshResult{}, err
	}
	s.reloadOverridesAfterWrite()

	result.Total = len(next)
	return result, nil
}

// ExportOverrideFile 返回覆盖文件的原始字节。文件不存在时返回空对象，让「先导出再
// 导入」在还没有任何微调时也能工作。
func (s *PricingService) ExportOverrideFile() ([]byte, error) {
	path, err := s.requireOverrideFilePath()
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []byte("{}\n"), nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return []byte("{}\n"), nil
	}
	return body, nil
}

// ImportOverrideEntries 用上传的 JSON 覆盖文件内容。mode="replace" 整体替换，
// mode="merge" 逐条合并（同模型以导入内容为准）。任一条目不合法就整体拒绝、不写文件：
// 导入一半的补丁文件比拒绝导入更难排查。
func (s *PricingService) ImportOverrideEntries(body []byte, mode string) (int, error) {
	path, err := s.requireOverrideFilePath()
	if err != nil {
		return 0, err
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "merge" {
		return 0, newOverrideValidationError("mode must be %q or %q, got %q", "replace", "merge", mode)
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return 0, newOverrideValidationError("override file must be a JSON object")
	}
	var imported map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &imported); err != nil {
		return 0, newOverrideValidationError("override file must be a JSON object: %v", err)
	}

	updatedAt := overrideTimestamp()
	names := sortedOverrideKeys(imported)
	tuned := make(map[string]json.RawMessage, len(names))
	// 返回值语义是「导入的模型条目数」，__defaults__ 是全局默认价段而非模型，不计入。
	modelCount := 0
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			return 0, newOverrideValidationError("override file contains an empty model name")
		}
		entry, ok := decodeOverrideEntry(name, imported[rawName])
		if !ok {
			return 0, newOverrideValidationError("entry %q: override entry must be a JSON object", name)
		}
		if name == OverrideDefaultsKey {
			if err := validateOverrideDefaultsFields(entry.Fields); err != nil {
				return 0, newOverrideValidationError("%s: %v", OverrideDefaultsKey, err)
			}
		} else {
			if err := validateOverrideFields(entry.Fields); err != nil {
				return 0, newOverrideValidationError("model %q: %v", name, err)
			}
			modelCount++
		}
		encoded, encodeErr := encodeOverridePatch(entry.Fields, true, updatedAt)
		if encodeErr != nil {
			return 0, encodeErr
		}
		tuned[name] = encoded
	}

	next := make(map[string]json.RawMessage, len(tuned))
	if mode == "merge" {
		existing, readErr := readOverrideEntries(path)
		if readErr != nil {
			return 0, readErr
		}
		for name, raw := range existing {
			next[name] = raw
		}
	}
	for name, raw := range tuned {
		next[name] = raw
	}

	if err := writeOverrideFileAtomic(path, next); err != nil {
		return 0, err
	}
	s.reloadOverridesAfterWrite()
	return modelCount, nil
}

// ReloadCustomPricingLayers 让覆盖层立刻生效，不必等下一次哈希检查定时器。
func (s *PricingService) ReloadCustomPricingLayers() error {
	if s == nil || s.cfg == nil {
		return errors.New("pricing service is not configured")
	}
	return s.reloadCustomPricingLayers()
}

// reloadOverridesAfterWrite 在每次写入覆盖文件后调用一次。失败只告警：目录文件还没
// 下载、并发编辑导致指纹漂移等情况都不该让一次已经落盘的写入回报错误。
func (s *PricingService) reloadOverridesAfterWrite() {
	if err := s.reloadCustomPricingLayers(); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override file written but reload failed: %v", err)
	}
}

// requireOverrideFilePath 返回配置的覆盖文件路径；未配置时返回错误——此时无法持久化
// 任何微调，必须让调用方知道，而不是静默丢弃。
func (s *PricingService) requireOverrideFilePath() (string, error) {
	path := s.OverrideFilePath()
	if path == "" {
		return "", errors.New("pricing.override_file is not configured")
	}
	return path, nil
}

// readOverrideEntries 读取覆盖文件为原始条目表。路径为空或文件不存在都视为空表。
func readOverrideEntries(path string) (map[string]json.RawMessage, error) {
	entries := make(map[string]json.RawMessage)
	if strings.TrimSpace(path) == "" {
		return entries, nil
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return entries, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return entries, nil
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("parse override file %s: %w", path, err)
	}
	if decoded == nil {
		return entries, nil
	}
	return decoded, nil
}

// writeOverrideFileAtomic 先写同目录临时文件再 rename：热重载按内容哈希每 10 分钟读一次
// 该文件，原地改写会让它读到半截 JSON。权限保持 0644，方便运维直接查看/编辑。
func writeOverrideFileAtomic(path string, entries map[string]json.RawMessage) error {
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".model_pricing_overrides-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, overrideFilePerm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// decodeOverrideEntry 宽容地把一条原始记录拆成元数据 + 价格补丁。
//
// 宽容是刻意的：这是读取路径，一条手写坏掉的条目不该让整个管理页面 500。解析不出对象
// 时返回 ok=false，调用方仍会拿到一个「已被手动修改、无字段」的占位条目。
func decodeOverrideEntry(model string, raw json.RawMessage) (OverrideEntry, bool) {
	entry := OverrideEntry{Model: model, Tuned: true, Fields: map[string]any{}}
	if len(raw) == 0 {
		return entry, false
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return entry, false
	}
	for key, value := range fields {
		switch key {
		case overrideTunedKey:
			if tuned, ok := value.(bool); ok {
				entry.Tuned = tuned
			}
		case overrideUpdatedAtKey:
			if text, ok := value.(string); ok {
				entry.UpdatedAt = text
			}
		default:
			entry.Fields[key] = value
		}
	}
	return entry, true
}

// encodeOverridePatch 把价格补丁与元数据合并成写入文件的原始条目。
func encodeOverridePatch(fields map[string]any, tuned bool, updatedAt string) (json.RawMessage, error) {
	patch := make(map[string]any, len(fields)+2)
	for name, value := range fields {
		patch[name] = value
	}
	patch[overrideTunedKey] = tuned
	patch[overrideUpdatedAtKey] = updatedAt
	return json.Marshal(patch)
}

// catalogOverridePatch 从目录条目里挑出白名单字段作为补丁。
//
// 值为 null 的字段直接跳过：null 在目录里表示「没有这个价格」，把它搬进覆盖文件只会
// 得到一条什么都没 pin 住的空条目。
//
// billing_expr 刻意不在搬运之列。单价字段抄一份是安全的（覆盖文件只负责固定价格数字，
// 抄错了改文件即可），而表达式一旦被抄进来就固化成快照：价格仓库或内置表后来改了规则，
// 这份旧表达式仍会把请求按旧规则计费，而且没有任何地方会提示它已经过期。
func catalogOverridePatch(raw json.RawMessage) map[string]any {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil
	}
	patch := make(map[string]any, len(overridePriceFields))
	names := make([]string, 0, len(overridePriceFields))
	for name := range overridePriceFields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		// 表达式不抄：见函数注释。白名单里有它，只是因为覆盖文件允许承载它。
		if name == overrideExprKey {
			continue
		}
		value, ok := fields[name]
		if !ok || value == nil {
			continue
		}
		// 目录里出现负数或非数字时不带进覆盖文件：覆盖文件应当只承载合法值，
		// 否则管理页面会显示一条无法再次保存的条目。
		if err := validateOverrideField(name, value); err != nil {
			continue
		}
		patch[name] = value
	}
	return patch
}

// overridePatchIsMetadataOnly 报告补丁除 _tuned/_updated_at 外没有其它键。
//
// 这类条目本身不含价格：既可能来自「只标了已微调」的管理端操作，也可能来自字段全被
// 删除的条目。mergeOverrideOnlyModels 的解析必然丢弃它们，但那不是「改价没生效」，
// 不该触发哨兵 WARN。
func overridePatchIsMetadataOnly(patch json.RawMessage) bool {
	var fields map[string]any
	if err := json.Unmarshal(patch, &fields); err != nil || fields == nil {
		return false
	}
	for key := range fields {
		if key != overrideTunedKey && key != overrideUpdatedAtKey {
			return false
		}
	}
	return true
}

// validateOverrideFields 校验整条补丁：至少一个字段，且每个字段名/取值合法。
func validateOverrideFields(fields map[string]any) error {
	if len(fields) == 0 {
		return newOverrideValidationError("fields must contain at least one price field")
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := validateOverrideField(name, fields[name]); err != nil {
			return err
		}
	}
	return nil
}

// validateOverrideField 校验单个字段：白名单内的单价字段必须是非负有限数字或 null，
// billing_expr 必须是字符串（**允许空串**），非空时要能通过 billingexpr 的编译校验，
// 其它字段名一律拒绝。
//
// 空串是合法取值，而且语义明确：`"billing_expr": ""` 表示「这个模型的价由目录/覆盖层
// 定义，代码内置规则不得接管」。它不是「没有表达式」，所以不能靠删键表达——删键等于
// 回到内置规则（DeepSeek 的官方峰谷表达式就是内置规则），这正是它要防的事。
func validateOverrideField(name string, value any) error {
	if name == overrideExprKey {
		text, ok := value.(string)
		if !ok {
			// null 与数字都不接受：null 在浅合并语义里是「删掉这个键」，而删键会让内置
			// 规则重新接管；要表达「用目录单价」应写空串。
			return newOverrideValidationError("field %q must be a string: use \"\" to declare the price from the price data", name)
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return nil
		}
		if err := billingexpr.Validate(trimmed); err != nil {
			// 编译错误的原文照抄给管理员：这是他们照着改表达式的唯一线索。
			return newOverrideValidationError("field %q is not a valid billing expression: %v", name, err)
		}
		return nil
	}

	if _, ok := overridePriceFields[name]; !ok {
		return newOverrideValidationError("unknown field %q: not an adjustable price field", name)
	}
	return validateOverrideNumberValue(name, value)
}

// validateOverrideDefaultsFields 校验 __defaults__ 段：至少一个字段，且每个字段都在
// overrideGlobalPriceFields 白名单内、取值合法。数值校验复用 validateOverrideField 用的
// 同一份校验逻辑，避免两份白名单出现不同的取值规则。
func validateOverrideDefaultsFields(fields map[string]any) error {
	if len(fields) == 0 {
		return newOverrideValidationError("fields must contain at least one price field")
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, ok := overrideGlobalPriceFields[name]; !ok {
			return newOverrideValidationError("unknown field %q: not an adjustable default price field", name)
		}
		if err := validateOverrideNumberValue(name, fields[name]); err != nil {
			return err
		}
	}
	return nil
}

// validateOverrideNumberValue 校验一个数字型价格字段的取值：非负有限数字或 null。
// overridePriceFields 与 overrideGlobalPriceFields 共用它，保证取值规则只有一份。
func validateOverrideNumberValue(name string, value any) error {
	if value == nil {
		return nil
	}
	number, ok := overrideFieldNumber(value)
	if !ok {
		return newOverrideValidationError("field %q must be a non-negative number or null", name)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return newOverrideValidationError("field %q must be a non-negative finite number or null", name)
	}
	return nil
}

// overrideFieldNumber 只接受数字类型。字符串（"1e-7"）刻意不接受：JSON 里数字和字符串
// 是两种类型，放行字符串会让前端把拼错的输入当成价格存下去。
func overrideFieldNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

func cloneOverrideFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for name, value := range fields {
		out[name] = value
	}
	return out
}

func sortedOverrideKeys(entries map[string]json.RawMessage) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func overrideTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
