# 声明式计费表达式（billing expressions）

一个模型的计费有时不能用「每 token 单价」表达：DeepSeek 按北京时间的峰谷段计价、
部分模型按上下文长度分档、图片模型混算 token 与张数。这些以前散落在 Go 代码的常量里，
改价必须改代码重新发布。

现在这些规则可以写成一个**计费表达式**字符串。表达式是该模型在默认价卡层的**完整
计费依据**：它算出金额，不再叠乘价格表里的 per-token 单价。语法与语义对齐
[new-api](https://github.com/QuantumNous/new-api) 的 `pkg/billingexpr`，表达式可以在
两个项目之间直接搬。

实现位置：

| 层 | 文件 |
|---|---|
| 表达式引擎（编译/求值） | `backend/pkg/billingexpr/compile.go`、`run.go`、`types.go` |
| 表达式展示解析 | `backend/pkg/billingexpr/describe.go` |
| 有效表达式解析（谁生效） | `backend/internal/service/model_billing_expression.go` |
| 接入计费主路径 | `backend/internal/service/model_billing_expression_cost.go` |
| 内置表达式 | `backend/internal/service/model_billing_expression.go` 的 `builtinModelBillingExpr` |
| 管理端价格总览 | `GET /api/v1/admin/channels/pricing/catalog` |

## 表达式写在哪：价格表里

在价格表条目上新增可选字段 `billing_expr`：

```json
{
  "deepseek-flash": {
    "input_cost_per_token": 1.5e-07,
    "output_cost_per_token": 6e-07,
    "cache_read_input_token_cost": 3e-09,
    "billing_expr": "v1:(...) ? tier(\"peak\", p * 0.30 + cr * 0.006 + c * 1.20) : tier(\"off_peak\", p * 0.15 + cr * 0.003 + c * 0.60)"
  }
}
```

放在价格表里的理由：表达式与价格同源、进 git 版本管理、随现有的哈希检查（默认 10 分钟）
热重载，**不需要 migration、不需要重启、不需要新的管理接口**。

### 谁可以定义表达式（优先级）

表达式有三个来源，从低到高：

```
内置 Go 表（builtinModelBillingExpr）
  <  价格仓库（远端目录 config.json / pricing.fallback_file）的 billing_expr
  <  pricing.override_file 的 billing_expr（覆盖文件是盖在目录条目上的补丁）
  <  分组 / 渠道显式定价（不用表达式）
```

要点：

- **显式声明即接管**：只要目录或覆盖文件里**出现** `billing_expr` 键，该模型的价与计费
  规则就由价格数据说了算，代码内置规则不再接管。对 DeepSeek 也一样：写了表达式，
  内置的官方峰谷表达式不再生效，`deepseek-*` 也不会再被强制改写成官方单价。
- **分组/渠道显式定价永远优先**。运营者在分组或渠道上配了价，表达式就不参与那次计费。
  这条是硬保证：渠道侧的自定义售价必须生效。
- 内置表（`builtinModelBillingExpr`）仍是 DeepSeek 的默认来源：目录里没有这个键时，
  `deepseek-*` 一律走内置的官方峰谷表达式。

### `"billing_expr": ""`：显式声明「按价格数据里的单价计费」

空串是**合法且有意义**的取值，它不等于「没有表达式」：

```json
{
  "deepseek-v3-2-251201": {
    "litellm_provider": "volcengine",
    "input_cost_per_token": 2e-07,
    "output_cost_per_token": 8e-07,
    "billing_expr": ""
  }
}
```

语义是「**这个模型的价由目录/覆盖层定义，别用内置规则接管**」：

- 不再被套内置表达式（例如 DeepSeek 官方峰谷表达式）；
- 不再被强制改写成内置官方单价（`deepseek-*` 的强制价豁免）；
- 计费完全按该条目的 per-token 单价走。

典型用途是**同名模型走别的上游**：`deepseek-v3-2-251201` 这类 `deepseek-` 前缀的名字由
volcengine 提供，却会被内置规则按 DeepSeek 官方峰谷价计费；写上 `"billing_expr": ""`
就回到价格数据里的单价。

注意：**删掉这个键不等于空表达式** —— 删键会回到内置规则兜底（又套上官方峰谷表达式），
要表达「用单价」必须显式写空串。

### DeepSeek：`deepseek-flash` 与 `deepseek-v4-pro` 各用各的峰谷表达式

官方现行口径（2026-09-22 核实，<https://api-docs.deepseek.com/zh-cn/quick_start/pricing>）
只有两个模型：`deepseek-flash`（DeepSeek-V4.1-Flash）与 `deepseek-v4-pro`
（DeepSeek-V4-Pro-0813）。两者各有一份「高峰 / 低谷」价卡——工作日北京时间
09:00-12:00、14:00-18:00 为高峰，其余时段与周末为低谷，低谷恰为高峰的一半。内置表按
模型名归档，谁都不看时刻：

| 模型名 | 内置表达式 | `billing_expr_source` |
|---|---|---|
| `deepseek-v4-pro`（含 `deepseek-v4-pro-0813` 这类版本化名称） | `deepseekProBillingExpr`：高峰 `p * 1.32 + cr * 0.044 + c * 3.96`，低谷为其一半 | `builtin` |
| `deepseek-flash` | `deepseekFlashBillingExpr`：高峰 `p * 0.30 + cr * 0.006 + c * 1.20`，低谷为其一半 | `builtin` |
| 旧名 `deepseek-v4-flash` / `deepseek-v4-flash-vision-exp`，以及 `deepseek-chat` / `deepseek-reasoner` 等其余 `deepseek-` 前缀 | 同 Flash 表达式 | `builtin` |

旧名 `deepseek-v4-flash` / `deepseek-v4-flash-vision-exp` 仍可调用，但对应模型已下线，
请求由 DeepSeek-V4.1-Flash 提供服务，因此按 Flash 价计费；`deepseek-v4-pro` 已恢复为现行
模型，不再被路由到 V4.1-Flash，一律按 Pro 价计费（曾经存在的「pro 自 2026-09-14 起按
Flash 计费」硬规则已从代码中整条删除，历史成本也不存在重算路径）。

表达式解析里没有任何「按时刻改写」的硬规则：`resolveBillingExprWithSource` 只看
「价格数据有没有声明过 `billing_expr`」这一件事（见下方两节），所以计费与展示在任意时刻
都一致，历史时点与当前时点走同一条链。

同一条归档规则也决定**基准单价**：`deepseek-*` 的展示单价被强制归一到对应档位的**低谷价**
（pro $0.66 / $1.98 / $0.022 每 MTok，flash $0.15 / $0.60 / $0.003 每 MTok），高峰 2×
由表达式给出，所以单价列不随时段翻转。价格数据显式声明 `billing_expr`（含空串）时，
单价不再被改写。

## 表达式语言

基于 [`expr-lang/expr`](https://github.com/expr-lang/expr)。支持算术 `+ - * / %`、比较
`== != < <= > >=`、逻辑 `&& || !`、三元 `? :`、括号、一元负号。

### 变量

所有变量都是**原始 token 数**（不预先除以 1e6）。表达式系数是**真实 USD / 百万 token**，
表达式的求值结果也是「USD / 百万 token」，扣费金额 = 结果 ÷ 1e6。

| 变量 | 含义 |
|---|---|
| `p` | 输入 token（**自动排除**表达式中单独计价的子类） |
| `c` | 输出 token（同上） |
| `len` | 完整输入上下文长度，**不受排除影响**，用于分档条件 |
| `cr` | 缓存命中（读取）token |
| `cc` | 缓存写入 token（5 分钟 TTL / 通用） |
| `cc1h` | 缓存写入 token（1 小时 TTL） |
| `img` | 图片输入 token |
| `img_cr` | 图片缓存命中 token |
| `ai` / `ao` | 音频输入 / 输出 token（当前账本未落这两项，恒为 0） |

**`p` 与 `len` 的区别很重要。** `p` 是「没有被单独定价的输入」：表达式引用了 `cr`，
缓存命中的 token 就从 `p` 里扣掉、按 `cr` 的价单算；没引用 `cr`，它们就留在 `p` 里按
输入价计费。这样可以写 `p * 3 + c * 15` 这种"不关心缓存"的表达式而不丢计费。
`len` 永远是完整上下文，所以**分档条件要用 `len` 而不是 `p`**——否则缓存命中越多、
`p` 越小，一个很长的请求会被误判成低档。

**反过来的坑：不想收费的类别要显式写 0。** 不在表达式里出现的子类会留在 `p` 里按输入价
计费，而不是免费。DeepSeek 的缓存写入就是免费项（价卡 `cache_creation_input_token_cost = 0`，
官方也是自动缓存、写入不收费），所以内置表达式里写了 `+ cc * 0 + cc1h * 0`：这个 0 系数把
缓存写入 token 从 `p` 里摘出来、按 0 计价。漏掉它，缓存写入就会开始按输入价收费——
金额上只差一点，但它是最容易静默上线的一类偏差。

### 函数

| 函数 | 说明 |
|---|---|
| `tier(name, value)` | 记录命中的档名，返回 `value`。用于给分支命名，便于对账与展示 |
| `hour(tz)` / `minute(tz)` / `weekday(tz)` / `month(tz)` / `day(tz)` | 计费时刻在 `tz` 时区的分量。`weekday` 0=周日…6=周六。`tz` 是 IANA 时区名 |
| `max` / `min` / `abs` / `ceil` / `floor` | 数学函数 |

时间函数读取的是**请求的计费时刻**（`CostInput.PricingAt`），不是墙上时钟。所以历史
补账会按当时的时段复算出当时的价格；测试也能用固定时点钉住结果。

### 版本前缀

`v1:` 可选前缀，不写等同于 `v1`。保留用于将来的文法演进。

### 示例

```
# 平坦定价
tier("base", p * 2.5 + c * 15 + cr * 0.25)

# 按上下文长度分档
len <= 200000
  ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)
  : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12)

# DeepSeek 峰谷价（工作日北京时间 09:00-12:00、14:00-18:00 为高峰，价格翻倍；
# 缓存写入免费，用 0 系数显式摘出）
v1:(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5
     && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12)
      || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18)))
  ? tier("peak", p * 0.30 + cr * 0.006 + c * 1.20 + cc * 0 + cc1h * 0)
  : tier("off_peak", p * 0.15 + cr * 0.003 + c * 0.60 + cc * 0 + cc1h * 0)
```

## 展示解析

`billingexpr.ParseTiers(expr)` 把表达式解析成「每个档 + 它的条件 + 它的各项价格」，
用于管理端价格总览页与模型广场。它**不求值**，所以没有任何流量的模型也能显示价格。

判定的形态是规范的「条件分档 + 每档一个 tier()」：

```
cond1 ? tier("a", p * x + c * y) : cond2 ? tier("b", ...) : tier("c", ...)
```

识别不了时返回 `Recognized=false`、`Tiers` 为空；**调用方必须回退展示表达式原文，不要
渲染一个半截的表格**——那会显示一个模型可能永远不会收的价。

时段条件会被额外渲染成人读文案（`weekday(tz) >= 1 && weekday(tz) <= 5` +
`hour(tz) >= 9 && hour(tz) < 12` → `周一至周五 09:00-12:00`），展示时优先用它。

## 节假日口径

DeepSeek 官方的峰谷时段是「北京时间周一至周五 09:00-12:00、14:00-18:00 为高峰；其余时段
（含周末与中国法定节假日）为低谷」。当前实现按**自然周末**判定：

```
weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5   // 周一至周五才可能高峰
```

也就是**周六、周日全天低谷**，而**法定节假日没有被建模** —— 落在工作日的假日会按高峰计价。
这是表达式语言当前的能力边界，不是配置问题。

将来要支持时的扩展方式：加一个 `holiday("Asia/Shanghai")` 函数（读取一张可配置的日期表），
时段条件写成 `!holiday(tz) && weekday(tz) >= 1 && ...`。在此之前不要试图用只按小时判定的
表达式绕过 —— 那样连周末也会算高峰。

## 已知缺口

- **未实现的语法**：按次固定价 `fixed()`、请求探针 `param()` / `header()` / `has()`、
  `|||` 请求规则、任务用量 `u()`。这些 new-api 有，我们按需再加。
- **表达式求值的分解**：逐项拆分 `CostBreakdown` 时，对「各项线性求和」形态是精确的；
  非线性的表达式（例如分档条件里直接比 `p`）会退回「总额记在输入项」并打一条 WARN。
