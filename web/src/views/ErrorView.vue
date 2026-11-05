<template>
  <div class="jj-page">
    <div class="jj-error">
      <div class="jj-error-code">{{ code }}</div>
      <div class="jj-error-msg">{{ title }}</div>
      <p class="jj-hint" style="max-width: 420px">{{ desc }}</p>
      <div style="margin-top: 12px; display: flex; gap: 10px">
        <el-button type="primary" @click="goHome">返回首页</el-button>
        <el-button @click="goBack">返回上一页</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const code = computed(() => route.meta.code || 404)
const title = computed(() => (code.value === 403 ? '没有访问权限' : '页面不存在'))
const desc = computed(() =>
  code.value === 403
    ? '当前账号未被授予该模块的查看权限。如需访问，请联系超级管理员在「角色权限」中调整。'
    : '链接可能已失效，或该页面尚未创建。'
)

function goHome() {
  router.replace('/')
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.replace('/')
}
</script>
