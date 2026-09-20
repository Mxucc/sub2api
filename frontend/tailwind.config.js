/**
 * DeepSeek 设计语言 Design Tokens
 *
 * 色值来源：从 deepseek.com 官方站点样式表 (--dsw-static-*) 与
 * DeepSeek App 语义别名 (--dsw-alias-*) 中提取的真实品牌色阶。
 *
 *  - primary : DeepSeek 品牌蓝（deepseek-50 … 900）
 *  - gray    : DeepSeek 冷中性色阶（neutral-bluish-*）
 *  - dark    : 深色模式层次色阶（bg-base → layer-1 → layer-2 → layer-3）
 *  - accent  : 保留旧类名，语义为冷中性色（兼容历史代码）
 *
 * 注意：全部类名与旧版本保持一一对应，仅替换色值，因此 200+ 页面
 * 无需改动即可整体换肤。
 */
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // DeepSeek 品牌蓝（--dsw-static-deepseek-*）
        // 500 = 品牌主色 #3964FE，400 = 深色模式品牌文字色 #679EFE
        primary: {
          50: '#edf3fe',
          100: '#e4edfd',
          200: '#d3e2ff',
          300: '#b7c8fe',
          400: '#679efe',
          500: '#3964fe',
          600: '#2f55e3',
          700: '#2444c0',
          800: '#34415b',
          900: '#283142',
          950: '#1c2230'
        },
        // 冷中性色阶（--dsw-static-neutral-bluish-* 对齐 Tailwind 明度梯度）
        gray: {
          50: '#f9fafb',
          100: '#f1f3f5',
          200: '#e5e8ee',
          300: '#cfd3d6',
          400: '#979da6',
          500: '#6b7076',
          600: '#61666b',
          700: '#43454a',
          800: '#2c2c2e',
          900: '#1e232c',
          950: '#151517'
        },
        // 深色模式层次（base / layer-1 / layer-2 / layer-3）
        dark: {
          50: '#f9fafb',
          100: '#f1f3f5',
          200: '#e1e5ee',
          300: '#cfd3d6',
          400: '#adb2b8',
          500: '#979da6',
          600: '#43454a',
          700: '#353638',
          800: '#2c2c2e',
          900: '#232324',
          950: '#151517'
        },
        // 兼容旧类名（原 Teal 辅助色），现为冷中性色
        accent: {
          50: '#f9fafb',
          100: '#f1f3f5',
          200: '#e1e5ee',
          300: '#cfd3d6',
          400: '#a5aab2',
          500: '#83888f',
          600: '#61666b',
          700: '#43454a',
          800: '#2c2c2e',
          900: '#1e232c',
          950: '#151517'
        },
        // 语义化品牌别名
        brand: {
          DEFAULT: '#3964fe',
          hover: '#2f55e3',
          active: '#2444c0',
          soft: '#edf3fe',
          deep: '#34415b',
          'dark-mode': '#5686fe',
          text: '#3964fe',
          'text-dark': '#679efe'
        },
        // DeepSeek 官方站点辅助色
        link: '#234792',
        'ink-bluish': '#152443'
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
      // DeepSeek 表面层级阴影：整体偏平，用极淡的投影 + 边框营造层次
      boxShadow: {
        xs: '0 1px 2px 0 rgba(21, 34, 60, 0.04)',
        glass: '0 8px 32px rgba(21, 34, 60, 0.08)',
        'glass-sm': '0 4px 16px rgba(21, 34, 60, 0.06)',
        glow: '0 0 0 3px rgba(57, 100, 254, 0.14)',
        'glow-lg': '0 0 0 6px rgba(57, 100, 254, 0.18)',
        card: '0 1px 2px 0 rgba(21, 34, 60, 0.04)',
        'card-hover': '0 8px 24px -6px rgba(21, 34, 60, 0.12)',
        popover: '0 12px 32px -8px rgba(21, 34, 60, 0.16), 0 2px 8px -2px rgba(21, 34, 60, 0.06)',
        dialog: '0 24px 60px -12px rgba(21, 34, 60, 0.22)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.08)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #3964fe 0%, #2444c0 100%)',
        'gradient-dark': 'linear-gradient(135deg, #232324 0%, #151517 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(57, 100, 254, 0.10) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(86, 134, 254, 0.08) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(103, 158, 254, 0.06) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.25s ease-out',
        'slide-up': 'slideUp 0.25s ease-out',
        'slide-down': 'slideDown 0.25s ease-out',
        'slide-in-right': 'slideInRight 0.25s ease-out',
        'scale-in': 'scaleIn 0.18s ease-out',
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
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.97)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 0 3px rgba(57, 100, 254, 0.10)' },
          '100%': { boxShadow: '0 0 0 5px rgba(57, 100, 254, 0.20)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      // DeepSeek 圆角体系：input 10px / media 12px / panel 16px / card 24px / pill 100px
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
