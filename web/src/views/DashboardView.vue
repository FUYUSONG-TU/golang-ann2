<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">
          {{ greeting }}，{{ displayName }}
          <el-tag v-if="auth.role" size="small" effect="plain" type="warning">{{ auth.role.name }}</el-tag>
        </h2>
        <p class="jj-page-desc">
          今天是 {{ todayText }}，这里是机匠学徒后台的整体运行情况。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
        <el-button v-if="canWriteUsers" type="primary" @click="$router.push('/users')">
          <el-icon><Plus /></el-icon>新建账号
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="announcement"
      class="announce"
      type="warning"
      :closable="false"
      show-icon
      :title="announcement"
    />

    <el-alert v-if="auth.user?.mustChangePwd" class="announce" type="error" show-icon :closable="false">
      <template #title>当前账号仍在使用初始密码，存在安全风险</template>
      <template #default>
        请前往「个人中心」修改密码，修改后该提示自动消失。
        <el-button link type="primary" size="small" @click="$router.push('/profile')">去修改</el-button>
      </template>
    </el-alert>

    <div class="jj-stats" v-loading="loading">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><User /></el-icon>账号总数</div>
        <div class="jj-stat-value">
          {{ cards.userTotal }}<span class="jj-stat-unit">个</span>
        </div>
        <div class="jj-stat-foot">
          启用 {{ cards.userActive }} · 今日新增 {{ cards.userToday }}
        </div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><Position /></el-icon>今日登录</div>
        <div class="jj-stat-value">
          {{ cards.loginToday }}<span class="jj-stat-unit">次</span>
        </div>
        <div class="jj-stat-foot">失败 {{ cards.loginFailToday }} 次</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Operation /></el-icon>今日操作</div>
        <div class="jj-stat-value">
          {{ cards.opToday }}<span class="jj-stat-unit">次</span>
        </div>
        <div class="jj-stat-foot">角色 {{ cards.roleTotal }} 个</div>
      </div>
      <div class="jj-stat" :class="dbOk ? '' : 'bad'">
        <div class="jj-stat-label"><el-icon><Coin /></el-icon>当前计数值</div>
        <div class="jj-stat-value">
          {{ cards.counterValue }}<span class="jj-stat-unit">次</span>
        </div>
        <div class="jj-stat-foot">
          记录 {{ cards.counterTotal }} 条 · 数据库 {{ dbOk ? latency + 'ms' : '异常' }}
        </div>
      </div>
    </div>

    <div class="grid-2">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">
            近 7 天登录趋势
            <span class="jj-card-title-sub">成功登录次数</span>
          </div>
          <span class="jj-sub">合计 {{ trendTotal }} 次</span>
        </div>
        <div class="jj-card-body">
          <div class="jj-trend">
            <div v-for="item in trend" :key="item.day" class="jj-trend-col">
              <div class="jj-trend-val">{{ item.total }}</div>
              <div
                class="jj-trend-bar"
                :class="{ zero: item.total === 0 }"
                :style="{ height: barHeight(item.total) }"
              />
              <div class="jj-trend-day">{{ item.day.slice(5) }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">系统状态</div>
          <el-tag :type="dbOk ? 'success' : 'danger'" size="small" effect="plain">
            {{ dbOk ? '运行正常' : '数据库异常' }}
          </el-tag>
        </div>
        <div class="jj-card-body">
          <dl class="jj-kv">
            <dt>后台版本</dt>
            <dd>v{{ version }}</dd>
            <dt>数据库延迟</dt>
            <dd>{{ dbOk ? latency + ' ms' : dbError }}</dd>
            <dt>已运行</dt>
            <dd>{{ uptimeText }}</dd>
            <dt>服务器时间</dt>
            <dd>{{ serverTimeText }}</dd>
            <dt>小程序端</dt>
            <dd>
              <el-tag size="small" type="info" effect="plain">待接入</el-tag>
              <el-button v-if="canViewMini" link type="primary" @click="$router.push('/mini')">
                查看骨架
              </el-button>
            </dd>
          </dl>
        </div>
      </div>
    </div>

    <div class="grid-2">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">最近登录</div>
          <el-button link type="primary" @click="$router.push('/logs')">全部日志</el-button>
        </div>
        <div class="jj-card-body">
          <div v-if="!recentLogins.length" class="jj-empty">暂无登录记录</div>
          <div v-else class="jj-list">
            <div v-for="item in recentLogins" :key="item.id" class="jj-list-item">
              <span class="jj-dot" :class="item.success ? 'ok' : 'bad'" />
              <span class="jj-strong">{{ item.username }}</span>
              <span class="jj-sub">{{ item.message }}</span>
              <span class="meta">{{ item.ip }} · {{ fmt(item.createdAt) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">最近操作</div>
          <span class="jj-sub">写操作自动留痕</span>
        </div>
        <div class="jj-card-body">
          <div v-if="!recentOps.length" class="jj-empty">暂无操作记录</div>
          <div v-else class="jj-list">
            <div v-for="item in recentOps" :key="item.id" class="jj-list-item">
              <span class="jj-dot" :class="item.success ? 'ok' : 'bad'" />
              <span class="jj-strong">{{ item.username || '系统' }}</span>
              <span class="jj-sub">{{ item.action }}</span>
              <span class="meta">{{ fmt(item.createdAt) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { auth, canView, canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canWriteUsers = computed(() => canWrite('users'))
const canViewMini = computed(() => canView('mp'))

const loading = ref(false)
const dbOk = ref(true)
const latency = ref(0)
const dbError = ref('')
const version = ref('3.0.0')
const uptime = ref(0)
const serverTime = ref('')
const announcement = ref('')
const trend = ref([])
const recentLogins = ref([])
const recentOps = ref([])

const cards = reactive({
  userTotal: 0,
  userActive: 0,
  userToday: 0,
  roleTotal: 0,
  loginToday: 0,
  loginFailToday: 0,
  opToday: 0,
  counterValue: 0,
  counterTotal: 0
})

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '管理员')
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 11) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const todayText = computed(() => {
  const d = new Date()
  const week = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月 ${d.getDate()} 日 星期${week}`
})

const trendTotal = computed(() => trend.value.reduce((s, i) => s + i.total, 0))
const maxTrend = computed(() => Math.max(1, ...trend.value.map((i) => i.total)))

const uptimeText = computed(() => {
  const s = uptime.value
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d) return `${d} 天 ${h} 小时`
  if (h) return `${h} 小时 ${m} 分`
  return `${m} 分钟`
})

const serverTimeText = computed(() => {
  if (!serverTime.value) return '-'
  const d = new Date(serverTime.value)
  return d.toLocaleString('zh-CN', { hour12: false })
})

function barHeight(total) {
  const ratio = total / maxTrend.value
  return Math.max(6, Math.round(ratio * 110)) + 'px'
}

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.dashboard()
    Object.assign(cards, data.cards || {})
    trend.value = data.trend || []
    recentLogins.value = data.recentLogins || []
    recentOps.value = data.recentOps || []
    dbOk.value = !!data.dbOk
    latency.value = data.dbLatencyMs || 0
    dbError.value = data.dbError || ''
    version.value = data.version || version.value
    uptime.value = data.uptimeSeconds || 0
    serverTime.value = data.serverTime || ''
    announcement.value = data.announcement || ''
  } catch (e) {
    dbOk.value = false
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.announce {
  margin-bottom: 14px;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.grid-2 .jj-card + .jj-card {
  margin-top: 0;
}

@media (max-width: 1000px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
