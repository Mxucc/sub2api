/**
 * 设计令牌：企业级控制台（TDesign / 腾讯云风格）
 *
 * 视觉基调：
 *  - 品牌色：TDesign 蓝 #0052D9（企业控制台标准色）；700 档已提亮为 #00359A（原 #002A8A 作正文强调过深）
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
    // ===== 全站圆角 5px（rounded-none 仍为 0）=====
    // 全站圆角：统一 5px（参考站指标卡/分段控件用 6–8px，这里取更克制的 5px）
    // rounded-none 仍为 0；rounded-full 一并收敛到 5px（头像/圆点变成小圆角矩形）。
    // 唯一例外：加载指示器 .spinner 在 style.css 里显式保留 50% 圆。
    borderRadius: {
      none: '0px',
      sm: '5px',
      DEFAULT: '5px',
      md: '5px',
      lg: '5px',
      xl: '5px',
      '2xl': '5px',
      '3xl': '5px',
      '4xl': '5px',
      full: '5px'
    },
    extend: {
      colors: {
        // 品牌色：TDesign 蓝（500 = #0052d9 主色 / 400 = hover / 600 = active；700 = 强调文字，已提亮）
        primary: {
          50: '#f2f3ff',
          100: '#d9e1ff',
          200: '#b5c7ff',
          300: '#8aa4ff',
          400: '#366ef4',
          500: '#0052d9',
          600: '#003cab',
          700: '#00359a',
          800: '#00206b',
          900: '#001a4d',
          950: '#001230'
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
          DEFAULT: '#0052d9',
          hover: '#366ef4',
          active: '#003cab',
          soft: '#f2f3ff',
          deep: '#00359a',
          'dark-mode': '#366ef4',
          text: '#0052d9',
          'text-dark': '#8aa4ff'
        },
        // TDesign 语义文本色（企业控制台常见分层）
        'text-1': '#1f1f1f',
        'text-2': '#5e5e5e',
        'text-3': '#8b8b8b',
        'text-4': '#c5c5c5',
        link: '#0052d9'
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
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
        display: [
          'Georgia',
          '"Noto Serif SC"',
          '"Source Han Serif SC"',
          '"Songti SC"',
          'SimSun',
          'serif'
        ],
      },
      // 企业控制台阴影：极淡、中性，用于区分层级而不是装饰
      boxShadow: {
        xs: '0 1px 2px 0 rgba(0, 0, 0, 0.04)',
        glass: '0 2px 8px rgba(0, 0, 0, 0.06)',
        'glass-sm': '0 1px 4px rgba(0, 0, 0, 0.05)',
        glow: '0 0 0 3px rgba(0, 82, 217, 0.12)',
        'glow-lg': '0 0 0 5px rgba(0, 82, 217, 0.16)',
        card: '0 1px 2px 0 rgba(0, 0, 0, 0.04)',
        'card-hover': '0 2px 8px -2px rgba(0, 0, 0, 0.10)',
        popover: '0 4px 12px -2px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.06)',
        dialog: '0 8px 24px -4px rgba(0, 0, 0, 0.16), 0 2px 8px rgba(0, 0, 0, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.08)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #0052d9 0%, #003cab 100%)',
        'gradient-dark': 'linear-gradient(135deg, #1f1f1f 0%, #181818 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(0, 82, 217, 0.06) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(54, 110, 244, 0.05) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(138, 164, 255, 0.04) 0px, transparent 50%)'
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
          '0%': { boxShadow: '0 0 0 3px rgba(0, 82, 217, 0.10)' },
          '100%': { boxShadow: '0 0 0 4px rgba(0, 82, 217, 0.18)' }
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
