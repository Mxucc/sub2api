<template>
  <!--
    登录/注册等认证页：与首页 / 模型广场 / 关于共用同一个公开页外壳（PublicLayout）。
    外壳已提供品牌、导航、语言/主题切换与底部版权，这里只负责把表单卡片居中。
    公开 API 保持不变：默认插槽放表单，footer 插槽放页脚链接。
  -->
  <PublicLayout>
    <div class="flex min-h-[calc(100vh-4rem)] items-center justify-center px-4 py-10">
      <div class="w-full max-w-[400px]">
        <!-- 单卡片：顶部 4px 品牌条 + 表单 -->
        <div class="card overflow-hidden shadow-card">
          <div class="h-1 bg-primary-500 dark:bg-primary-400"></div>
          <div class="p-8">
            <slot />
          </div>
        </div>

        <!-- Footer Links -->
        <div class="mt-4 text-center text-[13px]">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </PublicLayout>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import PublicLayout from '@/components/layout/PublicLayout.vue'
import { useAppStore } from '@/stores'

const appStore = useAppStore()

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
