import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it } from 'vitest'

import PublicNavTabs from '../PublicNavTabs.vue'
import { useAppStore, useAuthStore } from '@/stores'

const headerSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue'),
  'utf8'
)
const publicLayoutSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../PublicLayout.vue'),
  'utf8'
)

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', redirect: '/home' },
      { path: '/home', component: { template: '<div />' } },
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/admin/dashboard', component: { template: '<div />' } },
      { path: '/admin/users', component: { template: '<div />' } },
      { path: '/model-plaza', component: { template: '<div />' } },
      { path: '/about', component: { template: '<div />' } }
    ]
  })
}

function makeI18n() {
  return createI18n({
    legacy: false,
    locale: 'zh',
    messages: {
      zh: {
        home: {
          nav: {
            label: '站点导航',
            home: '首页',
            console: '控制台',
            plaza: '模型广场',
            about: '关于'
          }
        }
      }
    }
  })
}

async function mountAt(path: string, props: Record<string, unknown> = {}) {
  const router = makeRouter()
  await router.push(path)
  await router.isReady()
  const wrapper = mount(PublicNavTabs, {
    props,
    global: { plugins: [router, makeI18n()] }
  })
  await router.isReady()
  return wrapper
}

function setPlaza(enabled: boolean, requireAuth = false) {
  const appStore = useAppStore() as unknown as { cachedPublicSettings: unknown }
  appStore.cachedPublicSettings = {
    model_plaza_enabled: enabled,
    model_plaza_require_auth: requireAuth
  }
}

/** 登录态与管理员角色都由 store 的 user/token 推导（isAuthenticated / isAdmin 是 computed） */
function setUser(role: 'admin' | 'user' | null) {
  const authStore = useAuthStore() as unknown as { user: unknown; token: string | null }
  if (role === null) {
    authStore.user = null
    authStore.token = null
    return
  }
  authStore.token = 'test-token'
  authStore.user = { id: 1, username: 'tester', role } as unknown
}

describe('PublicNavTabs', () => {
  // 测试构建里 vue-i18n 不包含 message compiler，t() 直接回显 key —— 因此断言用 key
  const KEYS = ['home.nav.home', 'home.nav.console', 'home.nav.plaza', 'home.nav.about']

  beforeEach(() => {
    setActivePinia(createPinia())
    setPlaza(false)
    setUser(null)
  })

  // 回归：点进 /dashboard 后头部不再丢掉 首页 / 关于 等标签。
  it('keeps the same public tabs inside the console routes', async () => {
    setPlaza(true)
    const wrapper = await mountAt('/admin/users')
    expect(wrapper.findAll('a').map((a) => a.text())).toEqual(KEYS)
  })

  it('marks the console tab active on console routes and 首页 elsewhere', async () => {
    const inConsole = await mountAt('/dashboard')
    expect(inConsole.findAll('a')[1].classes()).toContain('text-primary-600')

    const onHome = await mountAt('/home')
    expect(onHome.findAll('a')[0].classes()).toContain('text-primary-600')
  })

  it('hides the model plaza tab when the site disables it', async () => {
    const wrapper = await mountAt('/home')
    expect(wrapper.findAll('a').map((a) => a.text())).toEqual([
      'home.nav.home',
      'home.nav.console',
      'home.nav.about'
    ])
  })

  it('opens the embedded plaza form inside the console', async () => {
    setPlaza(true)
    const wrapper = await mountAt('/dashboard', { embeddedPlaza: true })
    const plaza = wrapper.findAll('a').find((a) => a.text() === 'home.nav.plaza')
    expect(plaza?.attributes('href')).toBe('/model-plaza?embedded=1')

    const standalone = await mountAt('/home')
    expect(
      standalone.findAll('a').find((a) => a.text() === 'home.nav.plaza')?.attributes('href')
    ).toBe('/model-plaza')
  })

  it('links the console tab to the admin dashboard for admins', async () => {
    setUser('admin')
    const wrapper = await mountAt('/home')
    expect(wrapper.findAll('a')[1].attributes('href')).toBe('/admin/dashboard')
  })

  it('links the console tab to the user dashboard for regular users', async () => {
    setUser('user')
    const wrapper = await mountAt('/home')
    expect(wrapper.findAll('a')[1].attributes('href')).toBe('/dashboard')
  })
})

describe('public tabs are shared by every shell', () => {
  // 单一来源：公开页外壳与控制台外壳都用同一个组件
  it('uses PublicNavTabs in both PublicLayout and AppHeader', () => {
    expect(publicLayoutSource).toContain('<PublicNavTabs')
    expect(headerSource).toContain('<PublicNavTabs')
    expect(publicLayoutSource).not.toContain('internalLinks')
  })

  it('renders the tabs in the console mobile drawer too', () => {
    const sidebarSource = readFileSync(
      resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue'),
      'utf8'
    )
    expect(sidebarSource).toContain('<PublicNavTabs')
    expect(sidebarSource).toContain('vertical')
  })
})
