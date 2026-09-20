/**
 * 设计令牌：企业级控制台（TDesign / 腾讯云风格）
 *
 * 视觉基调：
 *  - 品牌色：青蓝 #08699A（hue 200°，白字对比度 6.0；比浓蓝 #0052D9 清爽，不是浅天蓝）
 *  - 中性色：TDesign 灰阶（#f7f8fa / #f2f3f5 / #e7e7e7 …），冷而不蓝
 *  - 深色：TDesign 深色层次（页面 #181818 → 容器 #1f1f1f → 浮层 #262626）
 *  - 形状：全站直角（borderRadius 全部 0）
 *  - 层次：靠 1px 描边 + 极淡投影，不用重阴影/彩色光晕
 *
 * 注意：类名与旧版本一一对应（primary/gray/dark/accent），因此 300+ 页面
 * 无需改动即可整体换肤；仅调整色值与圆角。
 */
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    // ===== 全站直角（sharp / square）=====
    // 覆盖 Tailwind 默认圆角刻度，使 rounded / rounded-sm … rounded-3xl /
    // rounded-full 全部解析为 0，因此 800+ 处既有类名无需改动即可整体变直角。
    // 唯一例外：加载指示器 .spinner 在 style.css 里显式保留 50% 圆（否则转圈动画视觉上不可读）。
    borderRadius: {
      none: '0px',
      sm: '0px',
      DEFAULT: '0px',
      md: '0px',
      lg: '0px',
      xl: '0px',
      '2xl': '0px',
      '3xl': '0px',
      '4xl': '0px',
      full: '0px'
    },
    extend: {
      colors: {
        // 品牌色：青蓝（500 = #08699a 主色 / 400 = hover / 600 = active；明度阶统一）
        primary: {
          50: '#f4fafd',
          100: '#e5f4fc',
          200: '#c2e8fb',
          300: '#5bc1f5',
          400: '#0a80bb',
          500: '#08699a',
          600: '#07567d',
          700: '#064462',
          800: '#06364e',
          900: '#052b3e',
          950: '#05212f'
        },
        // 中性色：TDesign 灰阶（50/100 用于页面与表头底色，500 起为文字）
        gray: {
          50: '#f7f8fa',
          100: '#f2f3f5',
          200: '#e7e7e7',
          300: '#dcdcdc',
          400: '#c5c5c5',
          500: '#777777',
          600: '#5e5e5e',
          700: '#4b4b4b',
          800: '#383838',
          900: '#242424',
          950: '#181818'
        },
        // 深色模式层次：950 页面 / 900 容器 / 800 浮层 / 700 分隔 / 600 强描边
        dark: {
          50: '#f7f8fa',
          100: '#f2f3f5',
          200: '#e7e7e7',
          300: '#dcdcdc',
          400: '#a6a6a6',
          500: '#8b8b8b',
          600: '#3d3d3d',
          700: '#2e2e2e',
          800: '#262626',
          900: '#1f1f1f',
          950: '#181818'
        },
        // 兼容旧类名（原辅助色），语义为冷中性色
        accent: {
          50: '#f7f8fa',
          100: '#f2f3f5',
          200: '#e7e7e7',
          300: '#dcdcdc',
          400: '#c5c5c5',
          500: '#777777',
          600: '#5e5e5e',
          700: '#4b4b4b',
          800: '#383838',
          900: '#242424',
          950: '#181818'
        },
        // 语义化品牌别名
        brand: {
          DEFAULT: '#08699a',
          hover: '#0a80bb',
          active: '#07567d',
          soft: '#f4fafd',
          deep: '#064462',
          'dark-mode': '#0a80bb',
          text: '#08699a',
          'text-dark': '#5bc1f5'
        },
        // TDesign 语义文本色（企业控制台常见分层）
        'text-1': '#1f1f1f',
        'text-2': '#5e5e5e',
        'text-3': '#8b8b8b',
        'text-4': '#c5c5c5',
        link: '#08699a'
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      // 企业控制台阴影：极淡、中性，用于区分层级而不是装饰
      boxShadow: {
        xs: '0 1px 2px 0 rgba(0, 0, 0, 0.04)',
        glass: '0 2px 8px rgba(0, 0, 0, 0.06)',
        'glass-sm': '0 1px 4px rgba(0, 0, 0, 0.05)',
        glow: '0 0 0 3px rgba(8, 105, 154, 0.12)',
        'glow-lg': '0 0 0 5px rgba(8, 105, 154, 0.16)',
        card: '0 1px 2px 0 rgba(0, 0, 0, 0.04)',
        'card-hover': '0 2px 8px -2px rgba(0, 0, 0, 0.10)',
        popover: '0 4px 12px -2px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.06)',
        dialog: '0 8px 24px -4px rgba(0, 0, 0, 0.16), 0 2px 8px rgba(0, 0, 0, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.08)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #08699a 0%, #07567d 100%)',
        'gradient-dark': 'linear-gradient(135deg, #1f1f1f 0%, #181818 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(8, 105, 154, 0.06) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(10, 128, 187, 0.05) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(91, 193, 245, 0.04) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-up': 'slideUp 0.2s ease-out',
        'slide-down': 'slideDown 0.2s ease-out',
        'slide-in-right': 'slideInRight 0.2s ease-out',
        'scale-in': 'scaleIn 0.15s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(16px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.98)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 0 3px rgba(8, 105, 154, 0.10)' },
          '100%': { boxShadow: '0 0 0 4px rgba(8, 105, 154, 0.18)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      // 企业控制台控件高度节奏：sm 24 / DEFAULT 28 / md 32 / lg 40
      height: {
        control: '32px',
        'control-sm': '24px',
        'control-lg': '40px'
      },
      fontSize: {
        // 控制台高频字号
        '2xs': ['11px', { lineHeight: '16px' }],
        caption: ['12px', { lineHeight: '18px' }],
        control: ['13px', { lineHeight: '20px' }]
      }
      // 注：圆角已在上方 theme.borderRadius 统一归零（全站直角），此处不再新增圆角令牌
    }
  },
  plugins: []
}
