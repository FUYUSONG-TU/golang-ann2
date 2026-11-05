<template>
  <el-container class="jj-layout">
    <el-aside :width="collapsed ? '64px' : '228px'" class="jj-aside">
      <div class="jj-brand">
        <div class="jj-logo">{{ logoText }}</div>
        <div v-show="!collapsed" class="jj-brand-text">
          <div class="jj-brand-name">{{ siteName }}</div>
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
        <template v-for="group in visibleGroups" :key="group.title">
          <div v-show="!collapsed" class="jj-menu-group">{{ group.title }}</div>
          <el-menu-item v-for="item in group.items" :key="item.path" :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </template>
      </el-menu>

      <div v-show="!collapsed" class="jj-aside-foot">
        <span class="jj-dot" :class="online ? 'ok' : 'bad'" />
        <span>{{ online ? '服务正常' : '连接异常' }}</span>
        <span class="ver">v{{ version }}</span>
      </div>
    </el-aside>

    <el-container class="jj-main-wrap">
      <el-header class="jj-header" height="56px">
        <div class="jj-header-left">
          <el-icon class="jj-collapse" @click="collapsed = !collapsed">
            <component :is="collapsed ? 'Expand' : 'Fold'" />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">{{ siteName }}</el-breadcrumb-item>
            <el-breadcrumb-item v-if="groupTitle">{{ groupTitle }}</el-breadcrumb-item>
            <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="jj-header-right">
          <el-tooltip content="刷新当前页" placement="bottom">
            <el-button text circle @click="reload">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </el-tooltip>

          <el-divider direction="vertical" />

          <el-dropdown trigger="click" @command="onCommand">
            <div class="jj-user">
              <el-avatar :size="28" class="jj-avatar">{{ avatarText }}</el-avatar>
              <div class="jj-user-meta">
                <span class="jj-user-name">{{ displayName }}</span>
                <span class="jj-user-role">{{ roleName }}</span>
              </div>
              <el-icon class="jj-user-arrow"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>个人中心
                </el-dropdown-item>
                <el-dropdown-item command="password">
                  <el-icon><Lock /></el-icon>修改密码
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main ref="mainRef" class="jj-main">
        <router-view v-slot="{ Component }">
          <transition name="jj-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>

    <el-dialog v-model="pwdVisible" title="修改密码" width="420px" :close-on-click-modal="false">
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="88px">
        <el-form-item label="原密码" prop="oldPassword">
          <el-input v-model="pwdForm.oldPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input v-model="pwdForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdSaving" @click="submitPassword">保存</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup>
import { computed, onMounted, provide, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { menuGroups } from '../menu'
import { api } from '../api'
import { auth, canView, clearSession, patchUser } from '../store/auth'

const route = useRoute()
const router = useRouter()

const mainRef = ref(null)
const collapsed = ref(localStorage.getItem('jj_sidebar_collapsed') === '1')
const online = ref(true)
const reloadTick = ref(0)
const version = ref('3.0.0')
const siteName = ref('机匠学徒')
const logoText = ref('匠')

provide('jjReloadTick', reloadTick)
provide('jjOnline', online)

const visibleGroups = computed(() =>
  menuGroups
    .map((g) => ({ ...g, items: g.items.filter((i) => canView(i.module)) }))
    .filter((g) => g.items.length)
)

const activeMenu = computed(() => route.meta.activeMenu || (route.path === '/' ? '/' : route.path))
const pageTitle = computed(() => route.meta.title || '数据看板')
const groupTitle = computed(() => {
  const target = activeMenu.value
  const hit = menuGroups.find((g) => g.items.some((i) => i.path === target))
  return hit ? hit.title : ''
})

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '管理员')
const roleName = computed(() => auth.role?.name || '未分配角色')
const avatarText = computed(() => (displayName.value || '管').slice(0, 1))

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdFormRef = ref(null)
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwdRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule, value, cb) => {
        if (value !== pwdForm.newPassword) cb(new Error('两次输入的密码不一致'))
        else cb()
      },
      trigger: 'blur'
    }
  ]
}

// 折叠状态记住，刷新后保持一致
watch(collapsed, (v) => {
  localStorage.setItem('jj_sidebar_collapsed', v ? '1' : '0')
})

// 切换页面时回到顶部，避免上一页的滚动位置残留
watch(
  () => route.path,
  () => {
    if (mainRef.value) mainRef.value.scrollTop = 0
  }
)

function reload() {
  reloadTick.value += 1
}

function onCommand(cmd) {
  if (cmd === 'profile') return router.push('/profile')
  if (cmd === 'password') return openPassword()
  if (cmd === 'logout') return doLogout()
}

function openPassword() {
  pwdForm.oldPassword = ''
  pwdForm.newPassword = ''
  pwdForm.confirmPassword = ''
  pwdVisible.value = true
}

async function submitPassword() {
  if (!pwdFormRef.value) return
  await pwdFormRef.value.validate(async (valid) => {
    if (!valid) return
    pwdSaving.value = true
    try {
      await api.changePassword(pwdForm.oldPassword, pwdForm.newPassword)
      ElMessage.success('密码已更新，请重新登录')
      pwdVisible.value = false
      clearSession()
      router.replace('/login')
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      pwdSaving.value = false
    }
  })
}

async function doLogout() {
  try {
    await ElMessageBox.confirm('确认退出当前账号？', '退出登录', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.logout()
  } catch {
    /* 退出失败也要清本地状态 */
  }
  clearSession()
  router.replace('/login')
}

onMounted(async () => {
  if (auth.user) patchUser(auth.user)
  try {
    const info = await api.profile()
    online.value = info.dbOk
    version.value = info.version || version.value
    if (info.siteName) siteName.value = info.siteName
  } catch {
    online.value = false
  }
})
</script>

<style scoped>
.jj-layout {
  height: 100vh;
  overflow: hidden;
}

/* ---------- 侧边栏 ---------- */
.jj-aside {
  background: linear-gradient(180deg, #1e2735 0%, #18202c 55%, #141b25 100%);
  display: flex;
  flex-direction: column;
  transition: width 0.24s var(--jj-ease);
  overflow: hidden;
  position: relative;
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.05);
}

.jj-brand {
  height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.14);
}

.jj-logo {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: linear-gradient(135deg, #f4a06a, #e8722c 62%, #cf5f1f);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(232, 114, 44, 0.4), inset 0 1px 0 rgba(255, 255, 255, 0.3);
  transition: transform 0.3s var(--jj-ease);
}

.jj-brand:hover .jj-logo {
  transform: rotate(-4deg) scale(1.04);
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
  white-space: nowrap;
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
  padding: 4px 0 12px;
  overflow-y: auto;
  background: transparent !important;
}

.jj-menu::-webkit-scrollbar {
  width: 0;
}

.jj-menu-group {
  padding: 16px 18px 6px;
  font-size: 10px;
  letter-spacing: 1.5px;
  color: #56637a;
  text-transform: uppercase;
  position: relative;
}

.jj-menu :deep(.el-menu-item) {
  height: 40px;
  line-height: 40px;
  margin: 1px 8px;
  border-radius: 8px;
  font-size: 13.5px;
  position: relative;
  color: #a8b4c6;
  transition: background 0.2s var(--jj-ease), color 0.2s var(--jj-ease),
    padding 0.2s var(--jj-ease);
}

.jj-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.055) !important;
  color: #e6edf6 !important;
}

.jj-menu :deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, #ec7b35, #e8722c 70%, #de6a26) !important;
  color: #fff !important;
  box-shadow: 0 6px 16px -8px rgba(232, 114, 44, 0.85),
    inset 0 1px 0 rgba(255, 255, 255, 0.18);
}

.jj-menu :deep(.el-menu-item.is-active::before) {
  content: '';
  position: absolute;
  left: 7px;
  top: 50%;
  width: 3px;
  height: 15px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.92);
  transform: translateY(-50%);
  box-shadow: 0 0 8px rgba(255, 255, 255, 0.5);
}

.jj-menu :deep(.el-menu-item .el-icon) {
  transition: transform 0.24s var(--jj-ease);
}

.jj-menu :deep(.el-menu-item:hover .el-icon) {
  transform: scale(1.08);
}

.jj-aside-foot {
  padding: 12px 18px;
  font-size: 11.5px;
  color: #6d7a8d;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  align-items: center;
  gap: 7px;
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.12);
}

.jj-aside-foot .ver {
  margin-left: auto;
  color: #4d5a6c;
  font-family: 'JetBrains Mono', Consolas, monospace;
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
  background: rgba(255, 255, 255, 0.86);
  backdrop-filter: saturate(180%) blur(12px);
  -webkit-backdrop-filter: saturate(180%) blur(12px);
  border-bottom: 1px solid var(--jj-line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  position: relative;
  z-index: 10;
  transition: box-shadow 0.3s var(--jj-ease);
}

.jj-header-left {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.jj-collapse {
  font-size: 18px;
  cursor: pointer;
  color: #5c6675;
  padding: 5px;
  border-radius: 7px;
  transition: color 0.2s ease, background 0.2s ease, transform 0.2s ease;
}

.jj-collapse:hover {
  color: var(--jj-orange);
  background: #fdf3ec;
  transform: scale(1.04);
}

.jj-header :deep(.el-breadcrumb__inner) {
  font-size: 12.5px;
  color: #93a0b2;
  font-weight: 400;
}

.jj-header :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
  color: var(--jj-text);
  font-weight: 600;
}

.jj-header :deep(.el-breadcrumb__separator) {
  color: #cfd6e0;
  margin: 0 7px;
}

.jj-header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.jj-user {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 10px 4px 4px;
  border-radius: 22px;
  border: 1px solid transparent;
  transition: background 0.22s var(--jj-ease), border-color 0.22s var(--jj-ease);
}

.jj-user:hover {
  background: #f7f8fa;
  border-color: #e9ecf1;
}

.jj-avatar {
  background: linear-gradient(135deg, #f4a06a, #e8722c);
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 2px 6px -1px rgba(232, 114, 44, 0.5);
}

.jj-user-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.jj-user-name {
  font-size: 13px;
  color: #3d4653;
  font-weight: 500;
}

.jj-user-role {
  font-size: 11px;
  color: #a5adba;
}

.jj-user-arrow {
  font-size: 12px;
  color: #9aa4b2;
  transition: transform 0.22s ease;
}

.jj-user:hover .jj-user-arrow {
  transform: translateY(1px);
}

/* ---------- 内容区 ---------- */
.jj-main {
  background:
    radial-gradient(1200px 400px at 80% -10%, rgba(232, 114, 44, 0.045), transparent 70%),
    var(--jj-bg);
  padding: 0;
  overflow-y: auto;
  scroll-behavior: smooth;
}
</style>
