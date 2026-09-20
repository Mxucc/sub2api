# Authentication Views Visual Guide

This document describes the visual design and layout of the authentication views.

## Layout Structure

Both LoginView and RegisterView use the AuthLayout component, which provides:

```
┌─────────────────────────────────────────────┐
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │                                     │   │
│  │         Sub2API Logo                │   │
│  │  "Subscription to API Conversion"   │   │
│  │                                     │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │                                     │   │
│  │    [Form Content - White Card]     │   │
│  │                                     │   │
│  └─────────────────────────────────────┘   │
│                                             │
│         [Footer Links]                      │
│                                             │
└─────────────────────────────────────────────┘

Background: Gradient (Indigo → White → Purple)
Card: White with rounded corners and shadow
Max Width: 28rem (448px)
Centered: Both horizontally and vertically
```

## LoginView Visual Design

### Default State

```
┌─────────────────────────────────────────────┐
│                                             │
│         🔷 Sub2API                          │
│    Subscription to API Conversion Platform  │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │                                     │   │
│  │        Welcome Back                 │   │
│  │  Sign in to your account to continue│  │
│  │                                     │   │
│  │  Username                           │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ Enter your username          │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  Password                           │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ ••••••••••••••                 │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  ☐ Remember me                      │   │
│  │                                     │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │       Sign In                  │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  └─────────────────────────────────────┘   │
│                                             │
│    Don't have an account? Sign up          │
│                                             │
└─────────────────────────────────────────────┘
```

### Loading State

```
┌─────────────────────────────────────────────┐
│  ┌────────────────────────────────┐         │
│  │  Username                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ john_doe                 │  │         │
│  │  └──────────────────────────┘  │         │
│  │                                │         │
│  │  Password                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ ••••••••••••            │  │         │
│  │  └──────────────────────────┘  │         │
│  │                                │         │
│  │  ☑ Remember me                 │         │
│  │                                │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ ⟳ Signing in...         │  │ ← Spinner
│  │  └──────────────────────────┘  │         │
│  │      (Button disabled)         │         │
│  └────────────────────────────────┘         │
└─────────────────────────────────────────────┘
```

### Error State

```
┌─────────────────────────────────────────────┐
│  ┌────────────────────────────────┐         │
│  │  Username                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ jo                       │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Username must be at least 3 │ ← Red text
│  │     characters                 │         │
│  │                                │         │
│  │  Password                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │                          │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Password is required        │ ← Red text
│  │                                │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ ⚠ Invalid username or    │  │ ← Error banner
│  │  │   password. Please try   │  │
│  │  │   again.                 │  │
│  │  └──────────────────────────┘  │         │
│  │                                │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │       Sign In            │  │         │
│  │  └──────────────────────────┘  │         │
│  └────────────────────────────────┘         │
└─────────────────────────────────────────────┘
```

## RegisterView Visual Design

### Default State

```
┌─────────────────────────────────────────────┐
│                                             │
│         🔷 Sub2API                          │
│    Subscription to API Conversion Platform  │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │                                     │   │
│  │        Create Account               │   │
│  │     Sign up to start using Sub2API  │   │
│  │                                     │   │
│  │  Username                           │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ Choose a username            │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  Email                              │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ your.email@example.com       │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  Password                           │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ Create a strong password     │ │   │
│  │  └────────────────────────────────┘ │   │
│  │  At least 8 characters with letters │  │
│  │  and numbers                        │   │
│  │                                     │   │
│  │  Confirm Password                   │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │ Confirm your password        │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  ┌────────────────────────────────┐ │   │
│  │  │     Create Account             │ │   │
│  │  └────────────────────────────────┘ │   │
│  │                                     │   │
│  │  By signing up, you agree to our   │   │
│  │  Terms of Service and Privacy Policy│  │
│  │                                     │   │
│  └─────────────────────────────────────┘   │
│                                             │
│   Already have an account? Sign in         │
│                                             │
└─────────────────────────────────────────────┘
```

### Validation Errors

```
┌─────────────────────────────────────────────┐
│  ┌────────────────────────────────┐         │
│  │  Username                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ jane@smith               │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Username can only contain   │ ← Red text
│  │     letters, numbers, _, and - │         │
│  │                                │         │
│  │  Email                         │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ invalid-email            │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Please enter a valid email  │ ← Red text
│  │     address                    │         │
│  │                                │         │
│  │  Password                      │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ short                    │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Password must be at least 8 │ ← Red text
│  │     characters with letters    │         │
│  │     and numbers                │         │
│  │                                │         │
│  │  Confirm Password              │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ different                │  │ ← Red border
│  │  └──────────────────────────┘  │         │
│  │  ⚠ Passwords do not match      │ ← Red text
│  │                                │         │
│  └────────────────────────────────┘         │
└─────────────────────────────────────────────┘
```

### Loading State

```
┌─────────────────────────────────────────────┐
│  ┌────────────────────────────────┐         │
│  │  Username: jane_smith          │         │
│  │  Email: jane@example.com       │         │
│  │  Password: ••••••••••••        │         │
│  │  Confirm: ••••••••••••         │         │
│  │                                │         │
│  │  ┌──────────────────────────┐  │         │
│  │  │ ⟳ Creating account...   │  │ ← Spinner
│  │  └──────────────────────────┘  │         │
│  │      (All inputs disabled)     │         │
│  └────────────────────────────────┘         │
└─────────────────────────────────────────────┘
```

## Color Palette

> 本节的权威来源是 `frontend/tailwind.config.js` 与 `docs/UI_DESIGN.md`（企业级控制台规范）。
> 下面列出登录/注册等认证页实际使用的令牌。

### Brand（品牌蓝，交互色）

- **primary-500**: `#0052D9` - 主按钮、链接、选中态
- **primary-400**: `#366EF4` - hover
- **primary-600**: `#003CAB` - active
- **primary-50**: `#F2F3FF` - 品牌浅底（选中背景、提示条）

### Neutral（TDesign 灰阶）

- **gray-900**: `#242424` - 标题与主正文
- **gray-700**: `#4B4B4B` - 表单标签
- **gray-600**: `#5E5E5E` - 次级正文
- **gray-500**: `#777777` - 说明文字、placeholder
- **gray-200**: `#E7E7E7` - 默认描边
- **gray-50**: `#F7F8FA` - 页面底色
- **White**: `#FFFFFF` - 卡片底色

### Semantic（状态色）

- **成功/正常/验证通过**: `sky` 系（天蓝）— 文字 `sky-700`、浅底 `sky-50`、
  深色模式文字 `sky-400` / 浅底 `sky-500/10`；实底按钮 `sky-700`
- **警告**: `amber` 系 — 文字 `amber-700`、浅底 `amber-50`
- **错误**: `red` 系 — 文字 `red-700`、浅底 `red-50`、描边 `red-200`

> 注意：本项目的状态色**不使用绿色**（emerald/green 已全部替换为天蓝 sky）。

### Shape

- 全站直角：圆角令牌统一为 `0`（唯一例外是加载指示器 `.spinner`）
- 控件高度 32px、字号 13px；卡片 1px 描边 + `shadow-xs`

## Typography

### Font Family

- **Default**: System font stack (`ui-sans-serif, system-ui, -apple-system, ...`)

### Font Sizes

- **Headings (h2)**: `1.5rem` (24px), `font-bold`
- **Body**: `0.875rem` (14px), `font-normal`
- **Labels**: `0.875rem` (14px), `font-medium`
- **Helper text**: `0.75rem` (12px), `font-normal`
- **Error text**: `0.875rem` (14px), `font-normal`

### Line Heights

- **Headings**: `1.5`
- **Body**: `1.5`
- **Helper text**: `1.25`

## Spacing

### Card Spacing

- **Padding**: `2rem` (32px) all sides
- **Gap between sections**: `1.5rem` (24px)
- **Gap between fields**: `1rem` (16px)

### Input Spacing

- **Padding**: `0.5rem 1rem` (8px 16px)
- **Label margin-bottom**: `0.25rem` (4px)
- **Error text margin-top**: `0.25rem` (4px)

### Button Spacing

- **Padding**: `0.5rem 1rem` (8px 16px)
- **Margin-top**: `1rem` (16px)

## Interactive States

### Input States

**Default:**

```css
border: 1px solid #D1D5DB (gray-300)
focus: 2px ring #6366F1 (indigo-500)
```

**Error:**

```css
border: 1px solid #EF4444 (red-500)
focus: 2px ring #EF4444 (red-500)
```

**Disabled:**

```css
background: #F3F4F6 (gray-100)
cursor: not-allowed
opacity: 0.6
```

### Button States

**Default:**

```css
background: #4F46E5 (indigo-600)
text: #FFFFFF (white)
shadow: shadow-sm
```

**Hover:**

```css
background: #4338CA (indigo-700)
transition: colors 150ms
```

**Focus:**

```css
outline: none
ring: 2px offset-2 #6366F1 (indigo-500)
```

**Disabled:**

```css
opacity: 0.5
cursor: not-allowed
```

**Loading:**

```css
opacity: 0.5
cursor: not-allowed
+ spinning icon
```

### Link States

**Default:**

```css
color: #4F46E5 (indigo-600)
font-weight: 500 (medium)
```

**Hover:**

```css
color: #6366F1 (indigo-500)
transition: colors 150ms
```

## Responsive Design

### Breakpoints

**Mobile (< 640px):**

```
- Full width container
- Padding: 1rem (16px)
- Smaller text sizes
```

**Tablet (640px - 768px):**

```
- Max width: 28rem (448px)
- Centered layout
- Standard spacing
```

**Desktop (> 768px):**

```
- Max width: 28rem (448px)
- Centered layout
- Standard spacing
```

### Mobile Optimizations

1. Touch-friendly tap targets (44px minimum)
2. Proper keyboard handling on mobile
3. Prevent zoom on input focus
4. Responsive font sizes
5. Full-width inputs
6. Adequate spacing for thumbs

## Animations

### Transitions

- Color changes: `150ms ease-in-out`
- Opacity changes: `150ms ease-in-out`
- Transform: `150ms ease-in-out`

### Loading Spinner

```css
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
animation: spin 1s linear infinite;
```

### Toast Animations

- Enter: Slide in from right + fade in
- Exit: Slide out to right + fade out
- Duration: 300ms

## Accessibility Features

### Visual Indicators

- Clear focus states (2px ring)
- Error states (red border + red text)
- Loading states (spinner + text)
- Success states (green toast)

### Color Contrast

- Text on white: > 7:1 (AAA)
- Labels on white: > 4.5:1 (AA)
- Buttons: > 4.5:1 (AA)
- Error text: > 4.5:1 (AA)

### Interactive Elements

- Minimum size: 44x44px (mobile)
- Clear hover states
- Distinct disabled states
- Keyboard accessible

### Screen Reader Support

- Proper labels on all inputs
- ARIA attributes where needed
- Error announcements
- Loading state announcements

## Icons

### Loading Spinner

```svg
<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
</svg>
```

### Error Icon

```svg
<svg class="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
</svg>
```

## Browser Compatibility

### Supported Browsers

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile Safari: iOS 14+
- Chrome Mobile: Latest 2 versions

### CSS Features Used

- Flexbox (full support)
- CSS Grid (full support)
- CSS Transitions (full support)
- CSS Custom Properties (full support)
- Gradient backgrounds (full support)

### JavaScript Features Used

- ES2015+ syntax
- Async/await
- Optional chaining
- Nullish coalescing
- Modules

## Print Styles

(Not applicable for authentication pages - users shouldn't print login forms)

## Dark Mode Considerations

**Future Enhancement:**

- Dark mode toggle in user preferences
- System preference detection
- Persistent dark mode setting
- Adjusted color palette for dark backgrounds

```css
/* Example dark mode colors (not implemented yet) */
dark:bg-gray-900
dark:text-white
dark:border-gray-700
```

## Performance Metrics

### Target Metrics

- First Contentful Paint (FCP): < 1s
- Largest Contentful Paint (LCP): < 2.5s
- Time to Interactive (TTI): < 3s
- Cumulative Layout Shift (CLS): < 0.1
- First Input Delay (FID): < 100ms

### Optimization Strategies

- Lazy load non-critical resources
- Minimize initial bundle size
- Use efficient animations (transform, opacity)
- Optimize images (logo, icons)
- Preconnect to API domain
- Cache static assets

## Component Size

### Bundle Impact

- LoginView.vue: ~4 KB (minified)
- RegisterView.vue: ~6 KB (minified)
- AuthLayout.vue: ~1 KB (minified)
- Total: ~11 KB (excluding dependencies)

### Dependencies

- Vue 3: ~40 KB (runtime)
- Vue Router: ~15 KB
- Pinia: ~10 KB
- Total framework overhead: ~65 KB (gzipped)

## Testing Checklist

### Visual Regression Tests

- [ ] Default state (login)
- [ ] Default state (register)
- [ ] Loading state
- [ ] Error state (validation)
- [ ] Error state (API)
- [ ] Success state
- [ ] Mobile view
- [ ] Tablet view
- [ ] Desktop view
- [ ] Focus states
- [ ] Hover states

### Cross-browser Tests

- [ ] Chrome (Windows, Mac, Linux)
- [ ] Firefox (Windows, Mac, Linux)
- [ ] Safari (Mac, iOS)
- [ ] Edge (Windows)
- [ ] Chrome Mobile (Android)
- [ ] Safari Mobile (iOS)

### Accessibility Tests

- [ ] Keyboard navigation
- [ ] Screen reader (NVDA)
- [ ] Screen reader (JAWS)
- [ ] Screen reader (VoiceOver)
- [ ] Color contrast
- [ ] Focus indicators
- [ ] Error announcements

## Design Assets

### Figma/Sketch Files

(Not applicable - designed directly in code with Tailwind)

### Design Tokens

- Defined in Tailwind config
- Consistent with design system
- Reusable across all components

### Iconography

- SVG icons inline
- Heroicons (outline and solid)
- Consistent stroke width
- Accessible with proper ARIA labels

---

**Note:** This visual guide is for reference and documentation purposes. The actual implementation is in the Vue components using TailwindCSS classes.
