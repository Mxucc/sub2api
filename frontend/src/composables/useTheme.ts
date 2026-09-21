import { ref, readonly } from 'vue'

/**
 * 浅色 / 深色主题（公开页与控制台共用）。
 *
 * 主题状态存在 html 的 `dark` 类上（Tailwind darkMode: 'class'），
 * 选择持久化在 localStorage 的 `theme` 键，与 src/main.ts 的首屏预置脚本一致。
 */
const STORAGE_KEY = 'theme'

function readInitial(): boolean {
  if (typeof document === 'undefined') return false
  if (document.documentElement.classList.contains('dark')) return true
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) return saved === 'dark'
  } catch {
    // localStorage 不可用时按浅色处理
  }
  return false
}

const isDark = ref(readInitial())

function apply(dark: boolean): void {
  isDark.value = dark
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', dark)
  }
  try {
    localStorage.setItem(STORAGE_KEY, dark ? 'dark' : 'light')
  } catch {
    // 忽略写入失败（隐私模式等）
  }
}

export function useTheme() {
  return {
    isDark: readonly(isDark),
    setTheme: apply,
    toggleTheme: () => apply(!isDark.value)
  }
}
