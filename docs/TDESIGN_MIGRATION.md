# TDesign Vue Next 迁移方案与影响评估

> 结论先说：**不建议一次性全量替换**。当前前端是「自研组件 + Tailwind 设计系统（已换成 DeepSeek 视觉）」，
> 308 个 `.vue`、14.3 万行、298 个测试文件里有 78 个直接断言设计系统类名。
> 全量换 TDesign 等于重写所有页面表现层，收益（更规范的组件实现）远小于成本与回归风险。
> 推荐路线：**分 3 阶段、以「公共基础组件」为切口**，每阶段独立可发布、可回滚；详细见 §5。

---

## 1. 现状盘点（迁移影响面的量化）

| 指标 | 数值 |
| --- | --- |
| Vue 单文件组件 | 308 个 / 约 143,600 行 |
| 前端测试 | 298 个文件 / 2,233 个用例（其中 78 个文件断言设计系统类名） |
| `<Icon>` 使用 | 774 处（已在用 Lucide，54 种图标名） |
| `<Select>` | 201 处 |
| `<Toggle>` | 148 处 |
| `<BaseDialog>` / `<ConfirmDialog>` | 97 / 42 处 |
| `<Pagination>` / `<DataTable>` | 29 / 21 处 |
| 表单类名 `class="input…"` | 1,626 处 |
| 按钮类名 `btn btn-*` | 582 处 |
| 徽章 / 下拉 / 卡片 / 标签页 | `badge` 231、`dropdown` 185、`class="card` 216、`tabs` 65 |

关键事实：**表现层是「类名约定」而不是「组件调用」**。`input`、`btn`、`card`、`badge` 这些是
`src/style.css` 的 `@layer components` 类，被 1600+ 处模板直接写死。TDesign 组件则通过自身
CSS 变量与内部 DOM 结构决定样式 —— 两者不是「替换标签」这么简单，而是两套样式语言的并存。

---

## 2. 目标框架

| 项 | 内容 |
| --- | --- |
| 包 | `tdesign-vue-next`（Vue 3）、图标 `tdesign-icons-vue-next` |
| 主题 | 通过 CSS 变量覆盖成 DeepSeek 品牌色（`--td-brand-color: #3964FE` 等），**不要**直接吃默认蓝色 |
| 暗色 | TDesign 用 `theme-mode="dark"` 属性；我们目前是 `html.dark` 类 → 需要双向同步（见 §4.2） |
| 按需 | `unplugin-vue-components` + `TDesignResolver`，避免全量引入（全量 ~1.4MB min） |
| 语言 | TDesign 自带 zh-CN / en-US，与我们 `vue-i18n` 的 key 体系互不影响，但需同步语言切换 |

---

## 3. 组件映射表（自研 → TDesign）

| 自研（现） | 引用量 | TDesign | 迁移要点 |
| --- | --- | --- | --- |
| `common/Select.vue` | 201 | `TSelect` + `TOption` | 我们支持分组/搜索/清除/错误态，需逐项映射；`modelValue` → `v-model`，`change` → `change` |
| `common/Toggle.vue` | 148 | `TSwitch` | 我们的是 `role="switch"` 自绘，事件名一致度低，需要一层适配组件 |
| `common/BaseDialog.vue` | 97 | `TDialog` | 我们的 `v-model` + 插槽 `header/default/footer`；TDesign 默认 Teleport + 遮罩，需保持 `body` 滚动锁行为一致 |
| `common/ConfirmDialog.vue` | 42 | `DialogPlugin.confirm` 或 `TDialog` | 42 处调用方签名要统一（Promise 风格 vs 回调） |
| `common/DataTable.vue` | 21 | `TTable` + `TBaseTable` | 我们实现了列宽拖拽、粘性表头/列、行选择、排序、空态等；TDesign 有 `TBaseTable` 支持虚拟滚动，但列宽拖拽需自研扩展 |
| `common/Pagination.vue` | 29 | `TPagination` | 含跳页输入与页大小选择，行为要对齐（有专门测试） |
| `common/DateRangePicker.vue` | 4 | `TDateRangePicker` | 我们只有预设 + 两个原生日期输入，TDesign 提供完整日历面板（体验升级） |
| `common/SearchInput.vue` / `Input.vue` / `TextArea.vue` | 少 | `TInput` / `TTextarea` | 类名 `input` 被 1626 处直接使用，迁移面最大的是「类名」而不是组件 |
| `common/Toast.vue` | 1 | `MessagePlugin` / `NotifyPlugin` | 全局函数式调用，替换我们 `appStore.toast` 的地方约 130+ 处 |
| `common/StatCard.vue` | 少量 | `TCard` + 自定义 | 统计卡片很轻，收益低，建议保留 |
| `common/EmptyState.vue` | — | `TEmpty` | 可平滑替换 |
| `common/Skeleton.vue` | — | `TSkeleton` | 可平滑替换 |
| `common/ModelIcon.vue` / `PlatformTypeBadge.vue` / `GroupBadge.vue` | — | 保留自研 | 业务语义强（平台/分组色），TDesign 无对应物 |
| `icons/Icon.vue`（Lucide） | 774 | `tdesign-icons-vue-next` | 二选一：保留 Lucide（当前方案）或改用 TDesign 图标；两者的 54 个名字需要一张映射表 |
| `layout/*`（Sidebar/Header/Layout） | — | `TMenu` / `TLayout` | 侧边栏有折叠、分组、徽标、权限过滤，迁移收益低、风险高，建议**不迁** |

---

## 4. 风险与冲突清单

### 4.1 样式体系冲突（最高风险，⭐️⭐️⭐️⭐️⭐️）
- **两套样式语言**：Tailwind 原子类（`bg-primary-500`）与 TDesign CSS 变量（`--td-brand-color`）并存时段，
  任何一处覆盖不当都会出现「一半直角、一半圆角」「一半品牌蓝、一半 TDesign 蓝」。
- **preflight 冲突**：Tailwind 的 `@tailwind base` 与 TDesign 的基础样式对 `button/input` 的默认重置不同步，
  已发现过的同类问题：类名优先级、`:hover` 覆盖、`box-sizing`。
- **我们的设计系统是「面直角 + 线圆润」**，TDesign 默认圆角 3–6px，需要全局覆盖
  `--td-radius-*: 0` 才能与现有页面一致，否则会出现风格断层。

### 4.2 暗色模式（⭐️⭐️⭐️⭐️）
我们：`html.classList.toggle('dark')`（Tailwind `darkMode: 'class'`）。
TDesign：`<t-config-provider theme-mode="dark">`。
需要写一层桥接（`watch` 主题变化 → 同步两处），否则深色模式下 TDesign 组件会是亮色。

### 4.3 测试断言（⭐️⭐️⭐️⭐️）
78 个测试文件断言 `class="card"`、`badge-*`、`text-primary-600`、`.modal-*` 等。
组件替换会让这些断言失效（不是"功能坏了"，而是"断言对象没了"），必须逐个改写 —— 这部分工作量常常被低估。

### 4.4 行为差异（⭐️⭐️⭐️）
- `Select`：键盘导航（↑↓/Enter/Esc/输入过滤）、`Teleport` 定位、滚动容器内翻转 —— 我们是自定义实现，
  有专门测试；TDesign 行为细节不同（如 Esc 是否清空、空值如何呈现）。
- `Dialog`：`body` 滚动锁、多弹窗堆叠顺序、`Esc` 关闭策略。
- `Table`：列宽拖拽、粘性表头 + 粘性首列的组合我们已实现；TDesign 需扩展。

### 4.5 体积与依赖（⭐️⭐️）
按需引入后单组件 ~10–40KB；`TBaseTable` 虚拟滚动依赖 `@vueuse/core`（我们已有）。
全量引入则 +1.4MB，必须走 Resolver 按需。

### 4.6 其它（⭐️⭐️）
- 第三方集成（Stripe Elements、阿里云验证码、微信支付二维码）**保持自研封装**，只换外壳。
- i18n：TDesign 组件内部文案需与 `vue-i18n` 语言联动（`ConfigProvider :global-config`）。
- 图表（Chart.js）、上传（`file-saver`）与 TDesign 无冲突，不动。
- `driver.js` 新手引导依赖我们的类名与 `data-*`，迁移时要同步 `onboarding.css`。

---

## 5. 推荐路线（分阶段，每阶段独立可发布）

### Phase 0 — 准备（0.5–1 天，无用户可感知变化）
1. 安装 `tdesign-vue-next` + `unplugin-vue-components`（Resolver 按需）+ `tdesign-icons-vue-next`。
2. 新建 `src/tdesign-theme.css`：把 DeepSeek 令牌映射到 TDesign 变量
   （`--td-brand-color: #3964FE`、`--td-radius-*: 0`、`--td-bg-color-*` → 我们的 `dark-900/950`）。
3. 加主题桥接：`html.dark` ⇄ `theme-mode="dark"`；`ConfigProvider` 同步 `zh-CN`/`en-US`。
4. 约定并存目录：`src/components/td/`（TDesign 适配层），**新组件只加在这里**，页面按需切换。
5. 验收：`pnpm build` 体积增量 < 30KB（按需引入）、测试全绿。

### Phase 1 — 基础控件（3–5 天，收益最大）
替换顺序按「引用量 × 行为简单度」排序：
1. `Input` / `TextArea` → `TInput`/`TTextarea`（同时把 1626 处 `class="input"` 逐步换成 `TInput`）
2. `Toggle` → `TSwitch`（148 处）
3. `Select` → `TSelect`（201 处，行为最复杂，需逐页回归）
4. `ConfirmDialog` / `BaseDialog` → `TDialog`（139 处）
5. `Toast` → `MessagePlugin`（130+ 处调用点）

**策略**：不做「改标签」式替换，而是在 `src/components/td/` 写**薄适配组件**（同名同 props 同事件），
页面里 `<Select>` 改成 `<TdSelect>`（或换 import 路径）即可，便于灰度与回滚。

### Phase 2 — 列表与布局（3–5 天）
- `DataTable` → `TBaseTable`（21 个页面；先迁 1–2 个低风险页面验证列宽拖拽/粘性列方案）
- `Pagination` → `TPagination`（29 处）
- `EmptyState` / `Skeleton` / `DateRangePicker` 顺带替换
- **不动** `layout/*`（侧边栏/头部），继续用自研（Tailwind 类名），保持导航稳定。

### Phase 3 — 长尾与清理（2–4 天）
- 清理 `style.css` 里被取代的组件类（`input`、`btn`、`modal-*`、`dropdown` …），
  在只剩使用不到 5 处时统一删除，避免两套样式长期并存。
- 统一图标来源（Lucide 或 TDesign 二选一）。
- 更新 `docs/` 与 `design-preview.html`。

**总投入估算**：8–15 个工作日（含回归），其中约 40% 花在测试断言改写与逐页视觉回归，而不是写代码。

### 回滚策略
- 适配层目录 + 逐页替换 ⇒ 任意页面可单独回到自研组件（保留旧组件直到 Phase 3 结束）。
- 每阶段一个独立 PR + 一次 fork release（`v0.2.7-g<sha>`），可随时回滚镜像版本。

---

## 6. 验收标准（每阶段）
1. `pnpm run lint:check`、`pnpm run typecheck`、`pnpm exec vitest run` 全绿（298 文件 / 2,233 用例）。
2. 视觉回归：`design-preview.html` + 关键页面截图对比（明/暗两套）。
3. 体积：主包增量 < 30KB（Phase 0），后续每阶段 < 60KB。
4. 行为：Select 键盘导航、Dialog 滚动锁与堆叠、Table 列宽拖拽/粘性列、Pagination 跳页 —— 逐项手测通过。
5. 无「两套风格混用」的页面：同屏内 TDesign 组件与 Tailwind 类必须共用同一套令牌（品牌蓝 + 直角）。

---

## 7. 我的建议（如果你只想要「更方正、更耐看」）

TDesign 的价值在**组件实现规范**（表单校验、虚拟滚动、无障碍、文档），而不是视觉；
它的默认视觉语言与我们已经定好的「DeepSeek 品牌蓝 + 全站直角 + Lucide 线性图标」并不相同，
迁移时必须再重做一遍主题覆盖，否则会退回「默认 TDesign 脸」。

因此：
- 若目标是**视觉**：现在的自研组件 + 设计系统已经能做到，且零迁移成本（本次已完成的直角 + Lucide 就是例子）。
- 若目标是**表单/表格这类复杂交互的长期可维护性**：按 Phase 1（基础控件）先做一轮试点，
  只迁 3–5 个页面，用真实回归成本验证收益，再决定是否继续 Phase 2/3。

需要我开工时，按 Phase 0 → Phase 1 的顺序做，每步独立提交、独立发版，可随时停。
