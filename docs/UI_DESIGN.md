# UI 设计规范 · 企业级控制台（TDesign / 腾讯云风格）

本文是前端视觉的唯一准绳。新增页面/组件请对照本文，不要自行发明间距、字号、色值。

---

## 1. 设计基调

| 维度 | 规范 |
| --- | --- |
| 定位 | **企业级控制台**（腾讯云 / TDesign 控制台），信息密度高、层级克制、可长时间阅读 |
| 形状 | **全站直角**：所有容器/控件圆角 = 0（`theme.borderRadius` 统一归零） |
| 层次 | 靠 **1px 描边** + 极淡中性投影（`shadow-xs` / `shadow-card` / `shadow-popover` / `shadow-dialog`），不用重阴影、不用彩色光晕 |
| 主色 | **品牌蓝 #0052D9**（hue 221°）；hover `#366EF4`、active `#003CAB`、浅底 `#F2F3FF` |
| 彩色使用 | 只有「品牌交互」和「状态语义」用彩色；其余一律中性灰 |
| 图标 | Lucide 线性图标，1.5px 描边，圆头端点（线圆润、面直角）；尺寸 12/16/20/24/32 |
| 动效 | 120–150ms，明确属性（`transition-colors` 等），位移 ≤ 4px |

---

## 2. 设计令牌

### 2.1 品牌色（primary）

| Token | 值 | 用途 |
| --- | --- | --- |
| `primary-50` | #F2F3FF | 选中行/品牌浅底 |
| `primary-100` | #D9E1FF | 焦点环、描边 |
| `primary-200` | #B5C7FF | 品牌描边（hover） |
| `primary-300` | #8AA4FF | 深色模式品牌文字 |
| `primary-400` | #366EF4 | hover |
| `primary-500` | #0052D9 | **主色**：主按钮、链接、选中 |
| `primary-600` | #003CAB | active/pressed |
| `primary-700` | #00359A | 深色强调（已从 #002A8A 提亮，避免浅底上接近黑蓝） |

### 2.2 中性色（gray）

| Token | 值 | 用途 |
| --- | --- | --- |
| `gray-50` | #F7F8FA | 页面底色、表头底色 |
| `gray-100` | #F2F3F5 | 次级面（hover、禁用、代码块） |
| `gray-200` | #E7E7E7 | **默认描边** |
| `gray-300` | #DCDCDC | 强描边、分隔线 |
| `gray-400` | #C5C5C5 | 装饰性图标、禁用态 |
| `gray-500` | #777777 | 提示文本、placeholder |
| `gray-600` | #5E5E5E | 次级正文 |
| `gray-700` | #4B4B4B | 表头文字 |
| `gray-900` | #242424 | 主标题、主正文 |

### 2.3 深色模式（dark）

| Token | 值 | 用途 |
| --- | --- | --- |
| `dark-950` | #181818 | 页面底色 |
| `dark-900` | #1F1F1F | 容器（卡片/侧边栏） |
| `dark-800` | #262626 | 浮层（菜单/弹窗/输入框） |
| `dark-700` | #2E2E2E | 描边 |
| `dark-600` | #3D3D3D | 强描边 |
| `dark-400` | #A6A6A6 | 次级文字 |

> 深色模式不使用纯黑，也不使用品牌色大面积铺底。

### 2.4 语义色（状态色）

全站只用三种状态色 + 一种品牌色；**不使用绿色**（历史遗留的 emerald/green 已全部替换为品牌蓝 primary）。

| 语义 | 文字 | 浅底 | 描边 / 实底 | 用途 |
| --- | --- | --- | --- | --- |
| 成功 / 正常 / 健康 | `primary-700`（深色 `primary-400`） | `primary-50` / `dark:primary-500/10` | `primary-200` / 实底 `primary-700`（白字） | 运行正常、验证通过、支付成功、健康指标 |
| 警告 | `amber-700`（深色 `amber-400`） | `amber-50` / `dark:amber-500/10` | `amber-200` / 实底 `amber-500` | 限流、配额告警、降级 |
| 危险 / 失败 | `red-700`（深色 `red-400`） | `red-50` / `dark:red-500/10` | `red-200` / 实底 `red-500` | 失败、错误、删除 |
| 品牌 / 交互 | `primary-600`（深色 `primary-300`） | `primary-50` / `dark:primary-500/10` | `primary-200` / 实底 `primary-500` | 链接、按钮、选中态、信息提示 |

品牌蓝色阶（primary）：`50 #F2F3FF` / `100 #D9E1FF` / `200 #B5C7FF` / `300 #8AA4FF` / `400 #366EF4` /
`500 #0052D9` / `600 #003CAB` / `700 #00359A`。

> 数据可视化里的分类色板同样不含绿色：原本的 green/emerald/teal/lime 系列位已改为
> `primary` 与 `cyan` 的蓝色调（保持同一张图表内颜色不重复）。

---

## 3. 密度与尺寸（企业控制台的核心）

| 元素 | 尺寸 |
| --- | --- |
| 控件高度 | 默认 **32px**（`h-control`）；小 24–28px（表格内/工具条）；大 40px（登录页主按钮） |
| 控件字号 | **13px**（`text-control`），辅助文字 12px（`text-caption`），最小 11px（`text-2xs`） |
| 正文 | 14px；表格单元格 13px；页面标题 18px `font-medium`；卡片标题 14px `font-medium` |
| 行高 | 表格行 ≈ 40px（`py-2.5` + 13px）；列表行 32–36px |
| 图标尺寸 | 控件内 16px；表格内 14px；功能入口 20px |
| 间距 | 页面内容 `gap-4`；卡片内 `p-4`；工具条 `py-3 px-4`；表单纵向 `gap-4`，label 与控件 `gap-1.5` |
| 页面留白 | 内容区 `px-4 py-4`（`lg:px-6 lg:py-5`），卡片之间 `gap-4` |

**顶栏 56px（h-14）/ 侧边栏 240px（折叠 64px）/ 侧边栏项 36px / 面包屑条 36px（h-9）** —— 控制台标准节奏。

---

## 4. 组件规范

### 按钮（`.btn`）
- 高度 32px，`px-3.5`，13px `font-medium`，直角。
- 变体：`.btn-primary`（实色品牌蓝）、`.btn-secondary`（白底 1px 描边）、`.btn-ghost`（无底）、`.btn-soft`（品牌浅底）、`.btn-danger`、`.btn-danger-soft`、`.btn-success`、`.btn-warning`。
- 禁用态 `opacity-45` + `cursor-not-allowed`；焦点环 `ring-2 ring-primary-500/40`。
- **禁止**渐变填充、彩色阴影、>1px 的描边。

### 输入（`.input`）
- 32px、13px、白底（深色 `dark-800`）、1px `gray-200` 描边、直角。
- hover 描边 `gray-300`；focus 描边 `primary-500` + `ring-2 ring-primary-500/20`。
- 错误态 `.input-error`（红描边）；辅助文字 `.input-hint` / `.input-error-text`（12px）。
- `textarea.input` 自动恢复为多行（`h-auto py-2`），不会被困在 32px。

### 页面头部（`PageHeader` / `.page-header-bar`）
结构固定为：**面包屑（可选）→ 标题 + 描述 → 右侧操作**，底部 1px 分隔。
- **面包屑由 `AppLayout` 的全局面包屑条统一提供**（`.crumb-strip`，来自路由 `meta.breadcrumbs`），
  页头默认不再重复渲染；`PageHeader` 仅在显式传入 `breadcrumbs` 时内联展示（用于抽屉/二级工作台）。
- 面包屑 12px `gray-500`，分隔符 `/`，最后一项为当前页（不可点）；`meta.breadcrumbs` 里写 i18n key，
  由布局层用 `te()` + `t()` 解析（写已翻译文本也能正常显示）。
- 标题 18px `font-medium`；描述 12–13px `gray-500`。
- 同一页面只出现一次页面头部，不要在内容区重复标题。

### 表格（企业控制台的骨架）
- 表头：`bg-gray-50 dark:bg-dark-800`，13px `font-medium text-gray-700 dark:text-dark-400`，**不用大写、不加字距**（中文界面无意义）。
- 单元格：13px，`px-3 py-2.5`，下分隔 `gray-200/70`（深色 `dark-700`）。
- 行 hover `bg-gray-50 dark:bg-dark-800`；选中行 `bg-primary-50 dark:bg-primary-500/10`。
- 数值列 `tabular-nums` 右对齐；操作列固定在右侧，用 `.icon-btn-sm`。
- 纸面化：表格外层包一层 `.data-card`（1px 描边 + 直角）。

### 卡片 / 区块
- `.card`：1px `gray-200` 描边 + `shadow-xs`，无圆角。
- `.card-header`：14px `font-medium` 标题 + 底部 1px 分隔；右侧放操作。
- `.data-card`：表格/列表容器（`.card` + `overflow-hidden`）。
- `.filter-bar`：筛选行（`px-4 py-3` + 底部分隔）；`.toolbar`：操作行（`flex justify-between py-3`）。

### 页面头部（`PageHeader` / `.page-header-bar`）
结构固定为：**面包屑 → 标题 + 描述 → 右侧操作**，底部 1px 分隔。
- 面包屑 12px `gray-500`，分隔符 `/`，最后一项为当前页（不可点）。
- 标题 18px `font-medium`；描述 12–13px `gray-500`。
- 同一页面只出现一次页面头部，不要在内容区重复标题。

### 徽章 / 标签
- `.badge`：状态语义标签，20–22px 高，12px，直角或胶囊（当前为直角）。
- `.tag`：紧凑契约标签（1px 描边 + 12px）；用于平台、分组、计费模式等。
- `.status-dot`：8px 方形状态点 + 12px 文字，适合表格内状态列。

### 弹窗 / 抽屉
- 头部 `px-4 py-3`（14px `font-medium` 标题），正文 `px-4 py-4`，底部 `px-4 py-3`（右侧按钮 32px）。
- 遮罩 `bg-black/45`；面板 `shadow-dialog` + 1px 描边；出现动效 150ms。

### 下拉 / 菜单
- 面板：1px `gray-200`（深色 `dark-600`）+ `shadow-popover`，`p-1`。
- 选项：32px 高、13px、hover `bg-gray-100 dark:bg-dark-700`、选中 `bg-primary-50 text-primary-500 dark:bg-primary-500/10 dark:text-primary-300`。

### 空态 / 骨架 / 加载
- 空态：图标 ≤ 40px（`gray-300` / `dark-600`），标题 14px，描述 12–13px `gray-500`，最多一个主按钮。
- 骨架：`.skeleton-shimmer`；加载指示器保留圆形（唯一圆角例外）。

---

## 5. 页面结构的标准骨架（应用外壳）

外壳由 `AppLayout` 提供，是**通栏顶栏 + 其下模块侧栏 + 内容区**的双区结构：

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ AppHeader  fixed inset-x-0 top-0 h-14 z-40  白底 / dark-900 + 底部 1px 描边 │
│ [☰移动端] [32px logo][产品名 15px][版本 tag] │ [当前页面 14px]* …… 右侧操作区 │
├──────────┬─────────────────────────────────────────────────────────────────┤
│ Sidebar  │ .crumb-strip  h-9（面包屑，仅当 route.meta.breadcrumbs 存在）    │
│ fixed    ├─────────────────────────────────────────────────────────────────┤
│ top-14   │ px-5 py-4 lg:px-6 lg:py-5                                       │
│ bottom-0 │   .page-header-bar   标题 + 描述 + 右侧操作（PageHeader）         │
│ 240px    │   .data-card → .filter-bar（筛选）→ 表格 → .card-footer（分页）   │
│ ⇄ 64px   │   或 .card / .surface 组成的任意内容                             │
└──────────┴─────────────────────────────────────────────────────────────────┘
* 顶栏的「当前页面」标题只在没有面包屑的路由上显示，避免与内容区页头重复。
```

要点：
- 顶栏通栏固定（`z-40`），侧栏与内容区都在其下（顶栏 56px）：侧栏 `fixed left-0 top-14 bottom-0`，
  内容区 `pt-14` + `lg:pl-60`（折叠 `lg:pl-16`）。
- 品牌（logo + 产品名 + 版本）只在顶栏出现一次；侧栏模块条只显示当前模块名（h-12）。
- 页面标题的归属：**有 `meta.breadcrumbs` 的页面**用 `.crumb-strip` + `PageHeader`；
  其余页面由顶栏显示标题。同一页面不会出现两处标题。
- 页面内若有自绘的吸顶工具栏，必须用 `sticky top-14`（顶栏高度），否则会滑到顶栏下面。
- 列表页统一用 `TablePageLayout`：`header`（PageHeader）→ `filters`（筛选 + 操作）→ `table` → `pagination`。


---

## 6. 禁止事项

1. 不要用渐变填充按钮/卡片背景、不要彩色阴影或光晕。
2. 不要使用 >1px 的装饰描边、不要 `rounded-*`（全站直角，令牌已归零）。
3. 不要在中文界面使用 `uppercase` / `tracking-wider`。
4. 不要用彩色表示「无含义的强调」；彩色只用于品牌交互与状态语义。
5. 不要出现 12px 以下的正文；不要出现两层以上嵌套卡片。
6. 不要新增未在本文登记的间距/字号/色值——需要时先补规范。
7. **不要使用绿色系**（`emerald` / `green` / `lime` / `teal` 及其 hex/rgb 写法）：
   状态「正常/成功/健康」统一用品牌蓝 `primary` 系；图表分类色板同样不含绿色。
8. 不要使用蓝色族以外的强调色（`blue` / `indigo` / `violet` / `purple` / `fuchsia` 的类名或 hex）：
   品牌与交互一律 `primary`（#0052D9）；紫色/靛蓝只允许作为平台身份色与图表数据系列出现。

---

## 7. 相关实现

| 文件 | 说明 |
| --- | --- |
| `frontend/tailwind.config.js` | 令牌唯一来源（色阶、阴影、控件高度、字号、5px 圆角、`font-display` 衬线字族） |
| `frontend/src/style.css` | 组件层（`.btn` / `.input` / `.table` / `.card` / `.data-card` / `.page-header-bar` / `.breadcrumb` / `.tag` …） |
| `frontend/src/components/layout/AppHeader.vue` | 通栏顶栏（56px）：品牌 + 当前页面标题 + 全局操作 + 版本/更新入口 |
| `frontend/src/components/layout/AppSidebar.vue` | 模块导航侧栏（240px ⇄ 64px），选中态品牌浅底 + 2px 左侧竖条 |
| `frontend/src/components/layout/AppLayout.vue` | 外壳：顶栏 + 侧栏 + 面包屑条 + 内容区 |
| `frontend/src/components/layout/PageHeader.vue` | 标准页面头部（标题 + 描述 + 操作槽；面包屑由外壳统一提供） |
| `frontend/src/components/icons/Icon.vue` + `iconMap.ts` | 图标（Lucide 映射，`name` 类型即契约） |
| `design-preview.html` | 静态设计预览（不参与构建），改令牌后重新生成 `design-preview.css` |
| `docs/TDESIGN_MIGRATION.md` | 若要真正替换组件框架（TDesign）的迁移方案与影响评估 |

## 附录 B · 仪表盘 / editorial 层（5px 圆角之后新增）

语言参考 kedaya.ai 的 signal 皮肤，拆解与取舍见 `docs/UI_REFERENCE_SIGNAL.md`。
实现在 `frontend/src/style.css` 末尾的 `@layer components` 中，与既有 `.card` / `.filter-bar` / `.data-card` 体系并存：

| 类 | 用途 |
| --- | --- |
| `.page-header-index` / `.page-header-copy` / `.page-header-meta` | 页头编号 / 说明文案（≤68ch，行高舒展）/ 元信息；`.page-title` 已改为衬线 22px |
| `.section-heading` + `.section-heading-title` + `.section-heading-meta` | 区块标题（衬线 15px + 底部发丝线 + 右侧元信息） |
| `.metric-hero-row` / `.metric-strip` | 仪表盘首行三格（`lg:grid-cols-[1.05fr_1.3fr_1.15fr]`：余额焦点卡 / 累计 Token / 快捷操作）与紧凑指标条（1/2/4 列）；对照 new-api「用量概览」的排版 |
| `.metric-label` / `.metric-value` / `.metric-unit` / `.metric-note` / `.metric-accent` / `.metric-foot` / `.metric-delta-up` / `.metric-delta-down` | 指标卡内部元素；数值 22px（焦点卡 24px）、标签与说明 11px、页脚 12px 约 30px 高；`.metric-foot` 是**满出血页脚**（发丝线 + 左标签 / 右数值） |
| `.chart-frame` / `.chart-hair` / `.chart-canvas` | 图表去卡片化：顶部发丝线 + 工具栏发丝线 + 260px（≤639px 时 220px）画布 |
| `.stat-pairs` / `.stat-pair` / `.stat-pair-label` / `.stat-pair-value` | 卡片页脚里的键值对（输入 / 输出 / 缓存、实际 / 标准），`auto-fit minmax(72px,1fr)` 自动分列 |
| `.stat-line` / `.stat-line-item` | 行内指标（性能 RPM/TPM、平均响应），不占卡片位 |
| `.action-grid` / `.action-item` / `.action-item-label` | 快捷操作两列动作格（图标 + 文案 + 箭头，发丝线分行） |
| `.collapse-head` / `.collapse-head-title` | 区块折叠头（按平台拆分 / 最近使用），标题左、元信息与箭头右 |
| `.platform-stat-row` | 平台卡片内的三行键值（今日消费 / 请求 / Token） |
| `.chart-canvas-wide` | 通栏趋势图画布高度（320px，≤639px 时 220px） |
| `.chip` | 轻量小徽标（比 `.badge` 更克制） |
| `.rise` / `.stagger` | 入场动效（0.46s `cubic-bezier(.16,1,.3,1)`，45/90/135ms 错峰，`prefers-reduced-motion` 下关闭） |
| `.decor-grid` | 极淡蓝图网格装饰层（60×60，`mask-image` 渐隐；可选） |

仪表盘排版顺序（用户 `/dashboard`，对照 new-api「用量概览」）：
**首行三格**（余额焦点卡 / 近期累计 Token / 快捷操作）→ **紧凑指标条**（今日 Token · 消费 · 请求 · 密钥）→
**行内指标**（性能 RPM/TPM · 平均响应）→ **图表**（工具栏：日期范围左、分段与刷新右 → 通栏趋势图 → 模型分布台账）→
**按平台拆分**（折叠头，**默认折叠**，需要时手动展开）→ **最近使用**（折叠头 + 台账）。

管理员仪表盘（`views/admin/DashboardView.vue`）用同一套排班：
首行三格（今日 Token 焦点卡 / 累计 Token / 快捷操作）→ 紧凑指标条（今日请求 · 今日消费 · API 密钥 · 账号）→
行内指标（性能 RPM/TPM · 平均响应 · 活跃用户 · 用户）→ 图表工具栏（日期范围左 / 粒度 + 刷新右）→
通栏 Token 趋势 → 模型分布（含用户消费排行）→ 用户用量趋势 Top 12。

四条沿用原则：

1. **投影几乎不可见**（`0 2px 4px rgba(32,36,38,.03)`），层与层之间优先用 1px 发丝线。
2. **一页一个焦点**：`.metric-card-hero` 只用于最关键的数字（余额），用品牌浅底 + 顶线而不是深色反色板。
3. **数字一律 `tabular-nums`**，标签用 11px 级别的小字，正文 13px。
4. **卡片紧凑**：内边距 13px、数值 22px（焦点卡 24px）、页脚约 30px，网格 `lg` 起 3 列 / `xl` 4 列 —— 一屏要能看到关键信息。
5. **全站 5px 圆角**（`rounded-none` 与 `.spinner` 除外）。

已落地的页面：用户 `/dashboard`（`views/user/DashboardView.vue` + `components/user/dashboard/*`）、
管理员仪表盘（`views/admin/DashboardView.vue`）；其余内页通过 `PageHeader` 与全局令牌自动继承同一套语言。

## 附录 C · 公开页外壳（PublicLayout）与公开路由

对外页面统一走 `frontend/src/components/layout/PublicLayout.vue`（参考 new-api 的一体化头部，用户 2026-09 口径）：

- **头部 56px 吸顶**：站点 Logo + 站名 + 导航 **首页 / 控制台 / 模型广场 / 关于** + 语言切换 + 主题切换 + 「登录 / 控制台」按钮；窄屏折叠为菜单。
- **底部**：版权 + 用量查询 + 文档 + 版本号。
- **Props**：`wide`（内容容器 7xl，模型广场用）、`showFooter`、`showNav`。
- 导航里的「模型广场」受站点开关 `model_plaza_enabled` 与 `model_plaza_require_auth` 约束，关闭时不渲染该标签；`/key-usage` 移到底部（不占导航位）。

| 路由 | 页面 | 外壳 |
| --- | --- | --- |
| `/home`（`/` 重定向到这里） | `views/HomeView.vue` | `PublicLayout`；管理员自定义首页内容（iframe/HTML）模式仍为整页，不套外壳 |
| `/model-plaza` | `views/ModelPlazaView.vue` | 独立形态用 `PublicLayout wide`；`?embedded=1`（后台内嵌）走 `AppLayout` |
| `/about` | `views/AboutView.vue` | `PublicLayout` |
| `/login`、`/register`、`/forgot-password`、`/reset-password` 及各类 OAuth 回调 | `views/auth/*` | `AuthLayout` → 内部包 `PublicLayout`（卡片居中，头部/底部由外壳提供） |
| 控制台（`/dashboard`、`/admin/*` 等） | 各业务页 | 不使用本外壳：控制台是 `AppLayout`（通栏顶栏 + 模块侧栏 + 面包屑条） |
