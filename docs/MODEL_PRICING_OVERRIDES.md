# 模型价格微调（override 文件）

管理员可以在后台「模型价格」（`/admin/model-pricing`）页面直接微调某个模型的单价。微调
保存在**覆盖文件**里，全局生效，不需要重启、不需要发版。

## 生效优先级

从高到低：

```
分组显式定价  >  渠道显式定价  >  手动微调（覆盖文件，含 __defaults__）  >  默认价卡（价格表 / 硬编码默认值）
```

也就是说：

- **微调只影响没有在分组/渠道上显式定价的模型。** 如果某个渠道为这个模型单独配了价，
  渠道价照旧生效，微调不会盖掉它。这是刻意的：渠道侧的自定义售价必须优先。
- 微调同样不会盖掉分组价 —— 分组价是最高的。

- **微调优先于硬编码默认值**：按次 / 按张 / 按秒 / 搜索 / 音频这些非 token 价格也一样。
  没有分组/渠道显式价时，先套用手动微调，再用内置的默认价卡（含 grok 之类的模型硬编码默认价）。

这条规则不需要在代码里特殊处理，它只是覆盖文件在解析链里的位置决定的
（`ModelPricingResolver` 的 `分组 → 渠道 → 目录 → 兜底` 顺序，而覆盖文件是并入「目录」这一层的）。

## 覆盖文件

默认路径 `./data/model_pricing_overrides.json`（配置项 `pricing.override_file`），容器里即
`/app/data/model_pricing_overrides.json`。它是一份**补丁**，不是价格表的副本：

```json
{
  "deepseek-flash": {
    "_tuned": true,
    "_updated_at": "2026-09-21T08:22:15Z",
    "input_cost_per_token": 1.5e-07,
    "output_cost_per_token": 6e-07
  },
  "gpt-5.5": {
    "_tuned": true,
    "_updated_at": "2026-09-21T08:22:15Z",
    "cache_read_input_token_cost": null
  }
}
```

- 键是模型名，必须与价格表的键一致。
- 值按**字段名浅合并**到价格表条目上。
- `_tuned` / `_updated_at` 是给「获取最新价格」用的元数据，价格解析器会忽略它们。
- 模型名不在价格表里也能写：该条目会作为独立条目并入。

### 关于 `null`：手改文件与后台界面语义不同

**手改文件**时，`null` 表示「把这个字段从价格表条目里删掉」（就是上面那条浅合并语义）。
这个能力有用：有时上游目录带了一个我们不想被它决定的字段，就得去掉。

**后台界面传 `null`**（清空某个输入框）时，含义是「不再微调这个字段」—— 后端会把这个键从
补丁里摘掉，于是价格表的值重新生效。**两者刻意不同，因为后果差很多**：真把
`input_cost_per_token` 从条目里删掉，解析时会读成 `0`，等于清空一个输入框就把模型变成免费。
所以界面上「恢复价卡价」是独立动作，等于删掉整条微调。

**可微调的字段（白名单）**只收单价字段，外加唯一的字符串字段 `billing_expr`；写别的键
会被后端拒绝并报出是哪个键：

| 字段 | 含义 |
|---|---|
| `input_cost_per_token` | 输入（USD / token） |
| `output_cost_per_token` | 输出 |
| `cache_read_input_token_cost` | 缓存命中（读） |
| `cache_creation_input_token_cost` | 缓存写入（5m / 通用） |
| `cache_creation_input_token_cost_above_1hr` | 缓存写入（1h） |
| `input_cost_per_token_priority` / `output_cost_per_token_priority` | priority 档输入 / 输出 |
| `cache_read_input_token_cost_priority` / `cache_creation_input_token_cost_priority` | priority 档缓存读写 |
| `input_cost_per_image_token` / `output_cost_per_image_token` | 图片输入 / 输出 |
| `billing_expr` | 声明式计费表达式（**字符串**，见下与 `MODEL_BILLING_EXPRESSIONS.md`） |
| `output_cost_per_image` | 按张价（图片生成）的基准单价（USD / 张） |

`billing_expr` 的规则与其它字段不同，因为它是**表达式而不是数字**：

- 取值必须是字符串。非空时要能通过 `pkg/billingexpr` 的编译校验；通不过会被拒绝，并把
  编译器返回的错误原文一起回给管理员（照着自己改表达式）。
- **允许空串**：`"billing_expr": ""` 表示「这个模型的价由本层/目录定义，代码内置规则不得
  接管」，例如让 volcengine 提供的 `deepseek-*` 不再被套 DeepSeek 官方峰谷价。它不是
  「没有表达式」，所以不能用删键表达——删键会退回内置规则。
- `null` **不接受**（在浅合并语义里 null 是「把这个键从条目里删掉」，与用途正好相反）。
  管理端清空该字段请用空串。
- 它**不会**被「获取最新价格」抄进覆盖文件，而其它单价字段会：把表达式抄一份等于固化一份
  会与代码/价格仓库漂移的快照（目录后来改了规则，覆盖文件里的旧表达式仍按旧规则计费）。
  要在覆盖文件里声明表达式，只能显式写入（后台微调、导入或手改文件）。

界面上按 **USD / 百万 token** 输入，写进文件时除以 1e6；`0.003` 这类小数字要照原样保留。


## 全局默认价：`__defaults__`

按次 / 按张 / 按秒 / 搜索 / 音频这类**非 token 价格**没有对应的模型条目，改不了单个模型，
因此覆盖文件里用保留键 `__defaults__` 存放**全局默认价**（不分模型）。它不是模型名：不会参与
目录条目的合并，也不会出现在「每模型微调」列表里。

```json
{
  "__defaults__": {
    "_tuned": true,
    "_updated_at": "2026-09-21T08:22:15Z",
    "web_search_price_per_call": 0.005,
    "image_price_1k": 0.02
  }
}
```

`__defaults__` 下可调字段（键名与分组/渠道定价的字段名一致）：

| 字段 | 含义 | 单位 |
|---|---|---|
| `image_price_1k` / `image_price_2k` / `image_price_4k` | 图片生成按张价 | USD / 张 |
| `video_price_480p` / `video_price_720p` / `video_price_1080p` | 视频生成按秒价 | USD / 秒 |
| `web_search_price_per_call` | Codex 网页搜索 | USD / 次 |
| `search_price_per_1k` | 搜索工具调用 | USD / 1000 次 |
| `audio_realtime_price_per_min` | 实时音频 | USD / 分钟 |
| `audio_tts_price_per_million_chars` | TTS | USD / 百万字符 |
| `audio_stt_price_per_hour` | STT | USD / 小时 |
| `per_request_price` | 按次计费的兜底单价 | USD / 次 |

- 优先级：**分组显式定价 > 渠道显式定价 > 每模型微调 > `__defaults__` 全局默认价 > 目录原生值 / 硬编码默认价卡**。
- 校验与 `null` 语义和每模型微调一致（非数字 / 负数 / 白名单外的键会被拒绝并报出字段名）。
- **按张价可以直接对单个模型改 `output_cost_per_image`**：它比全局的 `image_price_*` 更具体，
  所以压过全局默认价（`2K × 1.5` / `4K × 2` 的倍数照旧生效）。反过来，没有单独微调的模型
  一律走全局 `image_price_*`，即使目录里已经有自己的按张价 —— 否则全局默认价对目录里
  已有按张价的模型完全失效。
- HTTP：`GET/PUT/DELETE /api/v1/admin/channels/pricing/overrides/defaults`，`PUT` body 形如
  `{"fields": {...}}`。

## 后台操作

- **微调**：行内「微调」按钮，填单价、保存。已微调的行会打「已微调」标记；「恢复价卡价」
  删除该条目。
- **获取最新价格**：重新拉一次价格表，并询问是否覆盖已微调的条目
  - 覆盖全部：所有条目按最新价卡重写（**会覆盖管理员的微调**）
  - 保留微调：只刷新未微调的条目，并补上价卡里新增的模型
- **导入 / 导出**：导出就是下载这份 JSON；导入可选「替换全部」或「合并」。

## 热重载与落盘

- 写入是**原子**的（临时文件 + rename），后台每 10 分钟按内容哈希检测；后台每次写操作后
  会立即触发一次重载，所以改完马上生效。
- 文件不存在或为空都合法（视为没有微调）。
- 覆盖文件进 git 可审计；也可以直接手改文件（改完等下一次哈希检查，或重启）。

## 相关代码

| 层 | 文件 |
|---|---|
| 覆盖层的读取与合并 | `backend/internal/service/pricing_service.go`（`applyPricingOverrides` / `mergeOverrideOnlyModels` / `reloadCustomPricingLayers`） |
| 微调的读写与校验 | `backend/internal/service/model_pricing_overrides.go` |
| HTTP 接口 | `backend/internal/handler/admin/model_pricing_override_handler.go` |
| 后台页面 | `frontend/src/views/admin/ModelPricingView.vue` |
