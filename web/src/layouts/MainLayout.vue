<template>
  <el-container class="jj-layout">
    <el-aside :width="collapsed ? '64px' : '220px'" class="jj-aside">
      <div class="jj-brand">
        <div class="jj-logo">匠</div>
        <div v-show="!collapsed" class="jj-brand-text">
          <div class="jj-brand-name">机匠学徒</div>
          <div class="jj-brand-sub">ADMIN CONSOLE</div>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="collapsed"
        :collapse-transition="false"
        router
        class="jj-menu"
        background-color="#1d2531"
        text-color="#b3bccb"
        active-text-color="#ffffff"
      >
        <el-menu-item v-for="m in menus" :key="m.path" :index="m.path">
          <el-icon><component :is="m.icon" /></el-icon>
          <template #title>{{ m.title }}</template>
        </el-menu-item>
      </el-menu>

      <div v-show="!collapsed" class="jj-aside-foot">
        <span class="jj-dot" :class="onLine ? 'ok' : 'bad'" />
        {{ onLine ? '服务在线' : '连接异常' }}
      </div>
    </el-aside>

    <el-container class="jj-main-wrap">
      <el-header class="jj-header" height="56px">
        <div class="jj-header-left">
          <el-icon class="jj-collapse" @click="collapsed = !collapsed">
            <component :is="collapsed ? 'Expand' : 'Fold'" />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>机匠学徒</el-breadcrumb-item>
            <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="jj-header-right">
          <el-tag v-if="auth.authRequired" size="small" type="success" effect="plain">
            访问控制已启用
          </el-tag>
          <el-tag v-else size="small" type="warning" effect="plain">未设密钥</el-tag>

          <el-tooltip content="刷新数据" placement="bottom">
            <el-button circle text @click="reload">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </el-tooltip>

          <el-dropdown trigger="click" @command="onCommand">
            <div class="jj-user">
              <el-avatar :size="28" class="jj-avatar">匠</el-avatar>
              <span class="jj-user-name">管理员</span>
              <el-icon class="jj-user-arrow"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="about">系统信息</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="jj-main">
        <router-view v-slot="{ Component }">
          <transition name="jj-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, provide, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth, setToken } from '../store/auth'

const route = useRoute()
const router = useRouter()

const collapsed = ref(false)
const onLine = ref(true)
const reloadTick = ref(0)

provide('jjReloadTick', reloadTick)
provide('jjOnline', onLine)

const menus = [
  { path: '/', title: '仪表盘', icon: 'Odometer' },
  { path: '/counters', title: '计数管理', icon: 'Tickets' },
  { path: '/about', title: '系统信息', icon: 'InfoFilled' }
]

const activeMenu = computed(() => (route.path === '/' ? '/' : route.path))
const pageTitle = computed(() => route.meta.title || '仪表盘')

function reload() {
  reloadTick.value += 1
}

function onCommand(cmd) {
  if (cmd === 'about') {
    router.push('/about')
    return
  }
  if (cmd === 'logout') {
    setToken('')
    router.replace('/login')
  }
}
</script>

<style scoped>
.jj-layout {
  height: 100vh;
  overflow: hidden;
}

/* ---------- 侧边栏 ---------- */
.jj-aside {
  background: var(--jj-ink);
  display: flex;
  flex-direction: column;
  transition: width 0.22s ease;
  overflow: hidden;
}

.jj-brand {
  height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.jj-logo {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  background: linear-gradient(135deg, #f08a4b, #e8722c);
  color: #fff;
  font-size: 17px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(232, 114, 44, 0.35);
}

.jj-brand-text {
  min-width: 0;
}

.jj-brand-name {
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 1px;
  line-height: 1.2;
}

.jj-brand-sub {
  color: #6d7a8d;
  font-size: 9px;
  letter-spacing: 1.6px;
  margin-top: 2px;
}

.jj-menu {
  border-right: none;
  flex: 1;
  padding: 8px 0;
  overflow-y: auto;
}

.jj-menu :deep(.el-menu-item) {
  height: 44px;
  line-height: 44px;
  margin: 2px 8px;
  border-radius: 6px;
  font-size: 14px;
}

.jj-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.06) !important;
}

.jj-menu :deep(.el-menu-item.is-active) {
  background: var(--jj-orange) !important;
  color: #fff !important;
  box-shadow: 0 2px 10px rgba(232, 114, 44, 0.35);
}

.jj-aside-foot {
  padding: 12px 18px;
  font-size: 12px;
  color: #6d7a8d;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  align-items: center;
  gap: 7px;
  flex-shrink: 0;
}

.jj-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.jj-dot.ok {
  background: #4cc38a;
  box-shadow: 0 0 0 3px rgba(76, 195, 138, 0.15);
}

.jj-dot.bad {
  background: #f56c6c;
  box-shadow: 0 0 0 3px rgba(245, 108, 108, 0.15);
}

/* ---------- 顶栏 ---------- */
.jj-main-wrap {
  min-width: 0;
}

.jj-header {
  background: #fff;
  border-bottom: 1px solid var(--jj-line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}

.jj-header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.jj-collapse {
  font-size: 18px;
  cursor: pointer;
  color: #5c6675;
  transition: color 0.2s;
}

.jj-collapse:hover {
  color: var(--jj-orange);
}

.jj-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.jj-user {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px 4px 4px;
  border-radius: 20px;
  transition: background 0.2s;
}

.jj-user:hover {
  background: #f5f6f8;
}

.jj-avatar {
  background: linear-gradient(135deg, #f08a4b, #e8722c);
  font-size: 13px;
  font-weight: 600;
}

.jj-user-name {
  font-size: 13px;
  color: #3d4653;
}

.jj-user-arrow {
  font-size: 12px;
  color: #9aa4b2;
}

/* ---------- 内容区 ---------- */
.jj-main {
  background: var(--jj-bg);
  padding: 0;
  overflow-y: auto;
}

.jj-fade-enter-active,
.jj-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.jj-fade-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.jj-fade-leave-to {
  opacity: 0;
}
</style>
