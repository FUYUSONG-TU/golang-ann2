<template>
  <div class="dash">
    <transition name="notice">
      <div v-if="announcement" class="dash-notice">
        <el-icon><Bell /></el-icon>
        <span class="dash-notice-text">{{ announcement }}</span>
      </div>
    </transition>

    <transition name="notice">
      <div v-if="auth.user?.mustChangePwd" class="dash-notice danger">
        <el-icon><WarningFilled /></el-icon>
        <span class="dash-notice-text">当前账号仍在使用初始密码，存在安全风险，建议立即修改。</span>
        <el-button link type="primary" size="small" @click="$router.push('/profile')">去修改</el-button>
      </div>
    </transition>

    <!-- 驾驶舱 -->
    <section class="hud" :class="{ offline: !dbOk }">
      <div class="hud-grid" aria-hidden="true" />
      <div class="hud-glow" aria-hidden="true" />
      <div class="hud-scan" aria-hidden="true" />

      <header class="hud-head">
        <div class="hud-brand">
          <span class="hud-pulse" :class="dbOk ? 'ok' : 'bad'" />
          <span class="hud-brand-en">OPERATIONS CONSOLE</span>
          <span class="hud-brand-sep">/</span>
          <span class="hud-brand-cn">运营驾驶舱</span>
        </div>
        <div class="hud-tools">
          <span class="hud-chip" :class="dbOk ? 'ok' : 'bad'">
            {{ dbOk ? '系统运行正常' : '数据库连接异常' }}
          </span>
          <button class="hud-btn" :class="{ spinning: loading }" title="刷新数据" @click="load">
            <el-icon><Refresh /></el-icon>
          </button>
        </div>
      </header>

      <div class="hud-main">
        <div class="hud-greet">
          <h1>
            {{ greeting }}，{{ displayName }}
            <span v-if="auth.role" class="hud-role">{{ auth.role.name }}</span>
          </h1>
          <p>{{ todayText }} · 数控技能学习平台运行态势</p>
        </div>
        <div class="hud-clock">
          <div class="clock-time">{{ clockTime }}</div>
          <div class="clock-meta">
            <span class="clock-live">LIVE</span>
            <span>{{ clockDate }}</span>
            <span class="clock-sep">·</span>
            <span>UTC+8</span>
          </div>
        </div>
      </div>

      <div class="hud-metrics">
        <div
          v-for="(m, i) in metrics"
          :key="m.key"
          class="hud-metric"
          :style="{ animationDelay: 60 + i * 55 + 'ms' }"
        >
          <div class="hm-top">
            <span class="hm-label">{{ m.label }}</span>
            <el-icon class="hm-icon"><component :is="m.icon" /></el-icon>
          </div>
          <div class="hm-value">
            {{ m.value }}<span class="hm-unit">{{ m.unit }}</span>
          </div>
          <div class="hm-foot">{{ m.foot }}</div>
          <div class="hm-track">
            <div class="hm-fill" :class="m.tone" :style="{ width: m.ratio + '%' }" />
          </div>
        </div>
      </div>
    </section>

    <!-- 趋势 + 内容构成 -->
    <div class="dash-row row-a">
      <section class="jj-card panel">
        <div class="panel-head">
          <div class="panel-title">
            登录趋势
            <span class="panel-sub">近 7 天成功登录次数</span>
          </div>
          <div class="panel-figure">
            <b>{{ trendTotal }}</b>
            <span>次</span>
          </div>
        </div>
        <div ref="chartBox" class="chart-wrap">
          <svg
            class="chart"
            :viewBox="`0 0 ${chartW} ${chartH}`"
            preserveAspectRatio="none"
            @mousemove="onMove"
            @mouseleave="hoverIdx = -1"
          >
            <defs>
              <linearGradient id="areaFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#e8722c" stop-opacity="0.32" />
                <stop offset="55%" stop-color="#e8722c" stop-opacity="0.09" />
                <stop offset="100%" stop-color="#e8722c" stop-opacity="0" />
              </linearGradient>
            </defs>

            <g class="chart-grid">
              <template v-for="g in gridLines" :key="g.y">
                <line :x1="pad.l" :x2="chartW - pad.r" :y1="g.y" :y2="g.y" />
                <text :x="pad.l - 10" :y="g.y + 3.5">{{ g.label }}</text>
              </template>
            </g>

            <path class="chart-area" :d="areaPath" />
            <path class="chart-line" :d="linePath" path-length="1" />

            <line
              v-if="hoverPoint"
              class="chart-cross"
              :x1="hoverPoint.x"
              :x2="hoverPoint.x"
              :y1="pad.t"
              :y2="chartH - pad.b"
            />

            <g class="chart-dots">
              <circle
                v-for="(p, i) in points"
                :key="i"
                :cx="p.x"
                :cy="p.y"
                :r="hoverIdx === i ? 5 : 3.2"
                :class="{ on: hoverIdx === i }"
              />
            </g>

            <g class="chart-x">
              <text v-for="p in points" :key="p.day" :x="p.x" :y="chartH - 7">{{ p.label }}</text>
            </g>
          </svg>

          <div
            v-if="hoverPoint"
            class="chart-tip"
            :style="{ left: hoverPoint.x + 'px', top: hoverPoint.y + 'px' }"
          >
            <b>{{ hoverPoint.value }}</b> 次
            <span>{{ hoverPoint.day }}</span>
          </div>

          <div v-if="!trendTotal" class="chart-empty">近 7 天还没有登录记录</div>
        </div>
      </section>

      <section class="jj-card panel">
        <div class="panel-head">
          <div class="panel-title">
            内容构成
            <span class="panel-sub">课程状态占比</span>
          </div>
          <el-button link type="primary" @click="$router.push('/courses')">管理课程</el-button>
        </div>
        <div class="donut-body">
          <div class="donut" :style="donutStyle">
            <div class="donut-hole">
              <b>{{ publishRate }}<i>%</i></b>
              <span>发布率</span>
            </div>
          </div>
          <div class="donut-legend">
            <div v-for="s in contentSegments" :key="s.key" class="dl-item">
              <span class="dl-bar" :style="{ background: s.color }" />
              <span class="dl-name">{{ s.name }}</span>
              <span class="dl-val">{{ s.value }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- 健康度 / 环境 / 动态 -->
    <div class="dash-row row-b">
      <section class="jj-card panel">
        <div class="panel-head">
          <div class="panel-title">
            最近动态
            <span class="panel-sub">写操作自动留痕</span>
          </div>
          <div class="seg">
            <button
              v-for="t in tabs"
              :key="t.key"
              class="seg-item"
              :class="{ on: activeTab === t.key }"
              @click="activeTab = t.key"
            >
              {{ t.name }}
            </button>
          </div>
        </div>
        <div class="panel-body">
          <div v-if="!activeList.length" class="panel-empty">暂无记录</div>
          <ul v-else class="feed">
            <li v-for="item in activeList" :key="item.id" class="feed-item">
              <span class="feed-dot" :class="item.success ? 'ok' : 'bad'" />
              <div class="feed-main">
                <div class="feed-line">
                  <span class="feed-who">{{ item.username || '系统' }}</span>
                  <span class="feed-what">{{ item.action || item.message }}</span>
                </div>
                <div class="feed-meta">
                  {{ item.ip || '—' }} · {{ fmt(item.createdAt) }}
                </div>
              </div>
            </li>
          </ul>
        </div>
        <div v-if="activeTab === 'login'" class="panel-foot">
          <el-button link type="primary" @click="$router.push('/logs')">查看全部登录日志</el-button>
        </div>
      </section>

      <section class="jj-card panel">
        <div class="panel-head">
          <div class="panel-title">
            知识库健康度
            <span class="panel-sub">AI 可检索内容</span>
          </div>
          <el-button link type="primary" @click="$router.push('/kb')">去管理</el-button>
        </div>
        <div class="panel-body">
          <div class="stack">
            <div
              v-for="s in kbSegments"
              :key="s.key"
              class="stack-seg"
              :style="{ width: s.pct + '%', background: s.color }"
              :title="`${s.name} ${s.value} 份`"
            />
            <div v-if="!cards.kbTotal" class="stack-seg empty" style="width: 100%" />
          </div>

          <div class="kb-stats">
            <div class="kb-stat">
              <span class="kb-num">{{ cards.kbSearchable || 0 }}</span>
              <span class="kb-label">可被检索</span>
            </div>
            <div class="kb-stat">
              <span class="kb-num">{{ cards.kbChunks || 0 }}</span>
              <span class="kb-label">知识分块</span>
            </div>
            <div class="kb-stat" :class="{ bad: cards.kbFailed > 0 }">
              <span class="kb-num">{{ cards.kbFailed || 0 }}</span>
              <span class="kb-label">解析失败</span>
            </div>
          </div>

          <p class="kb-hint">
            {{
              cards.kbTotal
                ? '只有「已发布 + 解析成功」的资料才会被 AI 引用，其余内容不会出现在回答里。'
                : '还没有录入资料，AI 目前没有可引用的依据。'
            }}
          </p>
        </div>
      </section>

      <section class="jj-card panel">
        <div class="panel-head">
          <div class="panel-title">
            运行环境
            <span class="panel-sub">实时</span>
          </div>
          <span class="status-pill" :class="dbOk ? 'ok' : 'bad'">
            {{ dbOk ? '正常' : '异常' }}
          </span>
        </div>
        <div class="panel-body">
          <dl class="env">
            <dt>数据库延迟</dt>
            <dd class="mono" :class="{ danger: !dbOk }">{{ dbOk ? latency + ' ms' : '不可达' }}</dd>
            <dt>已运行</dt>
            <dd class="mono">{{ uptimeText }}</dd>
            <dt>服务器时间</dt>
            <dd class="mono">{{ serverTimeText }}</dd>
            <dt>后台版本</dt>
            <dd class="mono">v{{ version }}</dd>
            <dt>运营账号</dt>
            <dd>{{ cards.userTotal || 0 }} 个 · 角色 {{ cards.roleTotal || 0 }} 个</dd>
          </dl>
          <div class="env-foot">
            <el-tag size="small" type="info" effect="plain">小程序端待接入</el-tag>
            <el-button v-if="canViewMini" link type="primary" @click="$router.push('/mini')">
              查看骨架
            </el-button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { auth, canView } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
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
const composition = ref({})

const cards = reactive({
  userTotal: 0,
  userActive: 0,
  roleTotal: 0,
  loginToday: 0,
  loginFailToday: 0,
  opToday: 0,
  majorTotal: 0,
  courseTotal: 0,
  coursePublished: 0,
  pathTotal: 0,
  pathPublished: 0,
  kbTotal: 0,
  kbPublished: 0,
  kbFailed: 0,
  kbChunks: 0,
  kbSearchable: 0,
  learnerTotal: 0,
  learnerToday: 0
})

/* ---------- 时钟 ---------- */
const now = ref(new Date())
let clockTimer = 0
const clockTime = computed(() => {
  const d = now.value
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
})
const clockDate = computed(() => {
  const d = now.value
  const week = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(
    d.getDate()
  ).padStart(2, '0')} 周${week}`
})

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '管理员')
const greeting = computed(() => {
  const h = now.value.getHours()
  if (h < 6) return '夜深了'
  if (h < 11) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})
const todayText = computed(() => {
  const d = new Date()
  return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月 ${d.getDate()} 日 周${
    ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  }`
})

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
  if (!serverTime.value) return '—'
  return new Date(serverTime.value).toLocaleTimeString('zh-CN', { hour12: false })
})

/* ---------- 数字滚动 ---------- */
const shown = reactive({ course: 0, path: 0, learner: 0, kb: 0, login: 0, latency: 0 })
const reveal = ref(0)
let rafId = 0

function reduceMotion() {
  return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function playReveal() {
  cancelAnimationFrame(rafId)
  const targets = {
    course: cards.courseTotal || 0,
    path: cards.pathTotal || 0,
    learner: cards.learnerTotal || 0,
    kb: cards.kbTotal || 0,
    login: cards.loginToday || 0,
    latency: dbOk.value ? latency.value : 0
  }
  if (reduceMotion()) {
    Object.assign(shown, targets)
    reveal.value = 1
    return
  }
  const from = { ...shown }
  const start = performance.now()
  const dur = 850
  const step = (t) => {
    const p = Math.min(1, (t - start) / dur)
    const e = 1 - Math.pow(1 - p, 3)
    Object.keys(targets).forEach((k) => {
      shown[k] = Math.round(from[k] + (targets[k] - from[k]) * e)
    })
    reveal.value = e
    if (p < 1) rafId = requestAnimationFrame(step)
  }
  rafId = requestAnimationFrame(step)
}

/* ---------- 指标 ---------- */
const trendTotal = computed(() => trend.value.reduce((s, i) => s + (i.total || 0), 0))
const trendPeak = computed(() => Math.max(1, ...trend.value.map((i) => i.total)))

function ratio(a, b) {
  if (!b) return 0
  return Math.max(2, Math.min(100, Math.round((a / b) * 100)))
}

const metrics = computed(() => {
  const health = dbOk.value ? Math.max(6, Math.min(100, 100 - latency.value * 0.4)) : 0
  return [
    {
      key: 'course',
      label: '课程内容',
      icon: 'Reading',
      value: shown.course,
      unit: '门',
      foot: `已发布 ${cards.coursePublished || 0} · 专业 ${cards.majorTotal || 0}`,
      ratio: ratio(cards.coursePublished, cards.courseTotal),
      tone: ''
    },
    {
      key: 'path',
      label: '学习路线',
      icon: 'Guide',
      value: shown.path,
      unit: '条',
      foot: `已发布 ${cards.pathPublished || 0} 条`,
      ratio: ratio(cards.pathPublished, cards.pathTotal),
      tone: ''
    },
    {
      key: 'learner',
      label: '学员',
      icon: 'UserFilled',
      value: shown.learner,
      unit: '人',
      foot: `今日活跃 ${cards.learnerToday || 0} 人`,
      ratio: ratio(cards.learnerToday, cards.learnerTotal),
      tone: 'ok'
    },
    {
      key: 'kb',
      label: '知识库',
      icon: 'Files',
      value: shown.kb,
      unit: '份',
      foot: `已发布 ${cards.kbPublished || 0} · 分块 ${cards.kbChunks || 0}`,
      ratio: ratio(cards.kbPublished, cards.kbTotal),
      tone: cards.kbFailed > 0 ? 'warn' : ''
    },
    {
      key: 'login',
      label: '今日登录',
      icon: 'Position',
      value: shown.login,
      unit: '次',
      foot: `失败 ${cards.loginFailToday || 0} 次 · 今日操作 ${cards.opToday || 0}`,
      ratio: ratio(cards.loginToday, trendPeak.value),
      tone: 'ok'
    },
    {
      key: 'latency',
      label: '系统响应',
      icon: 'Odometer',
      value: dbOk.value ? shown.latency : 0,
      unit: dbOk.value ? 'ms' : '',
      foot: dbOk.value ? `v${version.value} · 已运行 ${uptimeText.value}` : '数据库不可达',
      ratio: health,
      tone: dbOk.value ? 'ok' : 'bad'
    }
  ]
})

/* ---------- 折线图 ---------- */
const chartBox = ref(null)
const chartW = ref(640)
const chartH = 232
const pad = { l: 38, r: 16, t: 18, b: 30 }
const hoverIdx = ref(-1)

const innerW = computed(() => Math.max(60, chartW.value - pad.l - pad.r))
const innerH = chartH - pad.t - pad.b

// 纵轴取整：把峰值向上取成 4 段的整齐刻度，避免出现 61 这种别扭的顶点
const chartScale = computed(() => {
  const peak = Math.max(1, ...trend.value.map((i) => i.total))
  if (peak <= 4) return { step: 1, max: 4 }
  const raw = peak / 4
  const mag = Math.pow(10, Math.floor(Math.log10(raw)))
  const nice = [1, 2, 2.5, 5, 10].find((n) => raw / mag <= n) || 10
  const step = nice * mag
  return { step, max: step * 4 }
})

const points = computed(() => {
  const list = trend.value
  if (!list.length) return []
  const step = list.length > 1 ? innerW.value / (list.length - 1) : 0
  return list.map((item, i) => ({
    x: pad.l + (list.length === 1 ? innerW.value / 2 : step * i),
    y: pad.t + innerH - (item.total / chartScale.value.max) * innerH,
    value: item.total,
    day: item.day,
    label: item.day.slice(5)
  }))
})

const gridLines = computed(() => {
  const { step, max } = chartScale.value
  return [4, 3, 2, 1, 0].map((i) => ({
    y: pad.t + innerH - ((i * step) / max) * innerH,
    label: i * step
  }))
})

function smooth(points) {
  if (!points.length) return ''
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`
  let d = `M ${points[0].x} ${points[0].y}`
  const t = 0.16
  for (let i = 0; i < points.length - 1; i++) {
    const p0 = points[i - 1] || points[i]
    const p1 = points[i]
    const p2 = points[i + 1]
    const p3 = points[i + 2] || p2
    // 控制点纵向钳制在两端点区间内，避免曲线在 0 附近过冲跌破基线
    const lo = Math.min(p1.y, p2.y)
    const hi = Math.max(p1.y, p2.y)
    const cy = (v) => Math.min(hi, Math.max(lo, v))
    d += ` C ${p1.x + (p2.x - p0.x) * t} ${cy(p1.y + (p2.y - p0.y) * t)}, ${
      p2.x - (p3.x - p1.x) * t
    } ${cy(p2.y - (p3.y - p1.y) * t)}, ${p2.x} ${p2.y}`
  }
  return d
}

const linePath = computed(() => smooth(points.value))
const areaPath = computed(() => {
  const list = points.value
  if (!list.length) return ''
  const bottom = pad.t + innerH
  return `${smooth(list)} L ${list[list.length - 1].x} ${bottom} L ${list[0].x} ${bottom} Z`
})

const hoverPoint = computed(() => {
  if (hoverIdx.value < 0) return null
  return points.value[hoverIdx.value] || null
})

function onMove(e) {
  const list = points.value
  if (!list.length) return
  const rect = e.currentTarget.getBoundingClientRect()
  const x = ((e.clientX - rect.left) / rect.width) * chartW.value
  let best = 0
  let dist = Infinity
  list.forEach((p, i) => {
    const d = Math.abs(p.x - x)
    if (d < dist) {
      dist = d
      best = i
    }
  })
  hoverIdx.value = best
}

let ro = null
function bindResize() {
  if (!chartBox.value) return
  ro = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const w = Math.round(entry.contentRect.width)
      if (w > 0) chartW.value = w
    }
  })
  ro.observe(chartBox.value)
  chartW.value = Math.round(chartBox.value.clientWidth) || chartW.value
}

/* ---------- 构成环图 ---------- */
const contentSegments = computed(() => {
  const c = composition.value.courses || {}
  return [
    { key: 'published', name: '已发布', value: c.published || 0, color: '#2f9e6e' },
    { key: 'pending', name: '审核中', value: c.pending || 0, color: '#d9a13b' },
    { key: 'draft', name: '草稿', value: c.draft || 0, color: '#8b97a8' },
    { key: 'offline', name: '已下架', value: c.offline || 0, color: '#c26a5f' }
  ]
})

const courseTotalForRate = computed(() =>
  contentSegments.value.reduce((s, i) => s + i.value, 0)
)
const publishRate = computed(() => {
  const total = courseTotalForRate.value
  if (!total) return 0
  return Math.round((contentSegments.value[0].value / total) * 100 * reveal.value)
})

const donutStyle = computed(() => {
  const list = contentSegments.value.filter((i) => i.value > 0)
  const total = list.reduce((s, i) => s + i.value, 0)
  if (!total) {
    return { background: 'conic-gradient(#e9edf3 0 100%)' }
  }
  let acc = 0
  const stops = list.map((s) => {
    const from = (acc / total) * 100 * reveal.value
    acc += s.value
    const to = (acc / total) * 100 * reveal.value
    return `${s.color} ${from.toFixed(2)}% ${to.toFixed(2)}%`
  })
  return {
    background: `conic-gradient(${stops.join(', ')}, #e9edf3 0 100%)`
  }
})

/* ---------- 知识库健康度 ---------- */
const kbSegments = computed(() => {
  const total = cards.kbTotal || 0
  const published = cards.kbPublished || 0
  const failed = cards.kbFailed || 0
  const rest = Math.max(0, total - published - failed)
  const norm = total || 1
  return [
    { key: 'published', name: '已发布', value: published, pct: (published / norm) * 100, color: '#2f9e6e' },
    { key: 'failed', name: '解析失败', value: failed, pct: (failed / norm) * 100, color: '#c26a5f' },
    { key: 'rest', name: '未发布', value: rest, pct: (rest / norm) * 100, color: '#d5dbe5' }
  ]
})

/* ---------- 动态列表 ---------- */
const activeTab = ref('login')
const tabs = [
  { key: 'login', name: '登录' },
  { key: 'op', name: '操作' }
]
const activeList = computed(() =>
  (activeTab.value === 'login' ? recentLogins.value : recentOps.value).slice(0, 6)
)

function fmt(v) {
  if (!v) return '—'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

/* ---------- 加载 ---------- */
async function load() {
  loading.value = true
  try {
    const data = await api.dashboard()
    Object.assign(cards, data.cards || {})
    composition.value = data.composition || {}
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
    playReveal()
  } catch (e) {
    dbOk.value = false
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

watch(reloadTick, load)

onMounted(() => {
  load()
  bindResize()
  clockTimer = setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onBeforeUnmount(() => {
  clearInterval(clockTimer)
  cancelAnimationFrame(rafId)
  if (ro) ro.disconnect()
})
</script>

<style scoped>
.dash {
  padding: 18px 22px 30px;
  animation: dash-in 0.42s cubic-bezier(0.22, 0.68, 0.24, 1) both;
}

@keyframes dash-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
}

/* ---------- 顶部提示 ---------- */
.dash-notice {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 9px 14px;
  margin-bottom: 12px;
  border-radius: 10px;
  background: linear-gradient(90deg, #fff6ed, #fffdfa);
  border: 1px solid #f6ddc6;
  color: #9a5a2c;
  font-size: 13px;
}

.dash-notice.danger {
  background: linear-gradient(90deg, #fdf3f3, #fffbfb);
  border-color: #f3d4d4;
  color: #a8443f;
}

.dash-notice-text {
  flex: 1;
  line-height: 1.5;
}

.notice-enter-active,
.notice-leave-active {
  transition: all 0.24s ease;
}

.notice-enter-from,
.notice-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

/* ---------- 驾驶舱 ---------- */
.hud {
  position: relative;
  border-radius: 16px;
  padding: 20px 24px 0;
  margin-bottom: 16px;
  overflow: hidden;
  color: #e9eff8;
  background: linear-gradient(135deg, #121a28 0%, #182334 46%, #0d1520 100%);
  box-shadow: 0 18px 40px -22px rgba(13, 21, 32, 0.65), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  animation: hud-in 0.55s cubic-bezier(0.22, 0.68, 0.24, 1) both;
}

@keyframes hud-in {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.995);
  }
}

.hud-grid {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(rgba(255, 255, 255, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.035) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: radial-gradient(120% 100% at 20% 0%, #000 20%, transparent 78%);
  pointer-events: none;
}

.hud-glow {
  position: absolute;
  width: 520px;
  height: 320px;
  right: -120px;
  top: -140px;
  background: radial-gradient(circle, rgba(232, 114, 44, 0.34), transparent 66%);
  pointer-events: none;
}

.hud-scan {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(232, 114, 44, 0.6),
    rgba(255, 255, 255, 0.28),
    transparent
  );
  animation: scan 7s linear infinite;
  opacity: 0.5;
  pointer-events: none;
}

@keyframes scan {
  0% {
    transform: translateY(0);
    opacity: 0;
  }
  12% {
    opacity: 0.55;
  }
  88% {
    opacity: 0.55;
  }
  100% {
    transform: translateY(300px);
    opacity: 0;
  }
}

.hud.offline .hud-glow {
  background: radial-gradient(circle, rgba(214, 84, 78, 0.32), transparent 66%);
}

.hud-head {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
}

.hud-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  letter-spacing: 1.6px;
}

.hud-pulse {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #4cc38a;
  box-shadow: 0 0 0 0 rgba(76, 195, 138, 0.6);
  animation: pulse 2.4s ease-out infinite;
}

.hud-pulse.bad {
  background: #ef6b64;
  box-shadow: 0 0 0 0 rgba(239, 107, 100, 0.6);
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 currentColor;
  }
  70% {
    box-shadow: 0 0 0 9px rgba(76, 195, 138, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(76, 195, 138, 0);
  }
}

.hud-brand-en {
  color: #7d8ea6;
  font-weight: 600;
}

.hud-brand-sep {
  color: #3d4b5f;
}

.hud-brand-cn {
  color: #c8d4e4;
  letter-spacing: 3px;
}

.hud-tools {
  display: flex;
  align-items: center;
  gap: 10px;
}

.hud-chip {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 20px;
  font-size: 11.5px;
  letter-spacing: 0.3px;
  border: 1px solid rgba(76, 195, 138, 0.34);
  background: rgba(76, 195, 138, 0.1);
  color: #6fd6a3;
}

.hud-chip.bad {
  border-color: rgba(239, 107, 100, 0.36);
  background: rgba(239, 107, 100, 0.1);
  color: #f08b84;
}

.hud-btn {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 7px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: #a9b7c9;
  cursor: pointer;
  transition: all 0.2s ease;
}

.hud-btn:hover {
  color: #fff;
  border-color: rgba(232, 114, 44, 0.5);
  background: rgba(232, 114, 44, 0.16);
}

.hud-btn.spinning .el-icon {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.hud-main {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  padding: 18px 0 20px;
  flex-wrap: wrap;
}

.hud-greet h1 {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  font-size: 23px;
  font-weight: 600;
  letter-spacing: 0.6px;
  color: #f4f8fd;
}

.hud-role {
  font-size: 11px;
  font-weight: 400;
  letter-spacing: 0.5px;
  padding: 2px 8px;
  border-radius: 20px;
  color: #f3b184;
  background: rgba(232, 114, 44, 0.14);
  border: 1px solid rgba(232, 114, 44, 0.3);
}

.hud-greet p {
  margin: 7px 0 0;
  font-size: 12.5px;
  color: #8698b0;
  letter-spacing: 0.4px;
}

.hud-clock {
  text-align: right;
}

.clock-time {
  font-family: 'JetBrains Mono', 'DIN Alternate', Consolas, monospace;
  font-size: 30px;
  font-weight: 600;
  letter-spacing: 2px;
  line-height: 1;
  color: #f0f5fc;
  font-variant-numeric: tabular-nums;
}

.clock-meta {
  margin-top: 7px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  font-size: 11px;
  color: #7b8ca3;
  letter-spacing: 0.6px;
}

.clock-live {
  color: #e8722c;
  font-weight: 600;
}

.clock-live::before {
  content: '';
  display: inline-block;
  width: 5px;
  height: 5px;
  margin-right: 4px;
  border-radius: 50%;
  background: #e8722c;
  vertical-align: middle;
  animation: pulse 2s infinite;
}

.clock-sep {
  color: #3d4b5f;
}

/* 指标条 */
.hud-metrics {
  position: relative;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  border-top: 1px solid rgba(255, 255, 255, 0.07);
}

.hud-metric {
  position: relative;
  padding: 16px 18px 14px;
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  transition: background 0.28s ease;
  animation: metric-in 0.5s ease both;
}

.hud-metric:last-child {
  border-right: none;
}

.hud-metric:hover {
  background: rgba(255, 255, 255, 0.035);
}

@keyframes metric-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}

.hm-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.hm-label {
  font-size: 12px;
  color: #8698b0;
  letter-spacing: 0.5px;
}

.hm-icon {
  font-size: 13px;
  color: #4a5a70;
  transition: color 0.24s ease;
}

.hud-metric:hover .hm-icon {
  color: #e8722c;
}

.hm-value {
  margin-top: 6px;
  font-family: 'JetBrains Mono', 'DIN Alternate', Consolas, monospace;
  font-size: 26px;
  font-weight: 600;
  line-height: 1.15;
  color: #f2f7fd;
  font-variant-numeric: tabular-nums;
}

.hm-unit {
  margin-left: 4px;
  font-family: var(--el-font-family);
  font-size: 11.5px;
  font-weight: 400;
  color: #7b8ca3;
}

.hm-foot {
  margin-top: 5px;
  height: 16px;
  font-size: 11px;
  color: #6b7c94;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hm-track {
  margin-top: 10px;
  height: 3px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.07);
  overflow: hidden;
}

.hm-fill {
  height: 100%;
  border-radius: 3px;
  background: linear-gradient(90deg, rgba(232, 114, 44, 0.55), #e8722c);
  transition: width 0.9s cubic-bezier(0.22, 0.68, 0.24, 1);
  box-shadow: 0 0 8px rgba(232, 114, 44, 0.45);
}

.hm-fill.ok {
  background: linear-gradient(90deg, rgba(76, 195, 138, 0.5), #4cc38a);
  box-shadow: 0 0 8px rgba(76, 195, 138, 0.4);
}

.hm-fill.warn {
  background: linear-gradient(90deg, rgba(224, 165, 44, 0.5), #e0a52c);
  box-shadow: 0 0 8px rgba(224, 165, 44, 0.4);
}

.hm-fill.bad {
  background: linear-gradient(90deg, rgba(239, 107, 100, 0.5), #ef6b64);
  box-shadow: 0 0 8px rgba(239, 107, 100, 0.4);
}

/* ---------- 面板通用 ---------- */
.dash-row {
  display: grid;
  gap: 16px;
  margin-bottom: 16px;
}

.row-a {
  grid-template-columns: minmax(0, 1.65fr) minmax(0, 1fr);
  animation: rise 0.55s var(--jj-ease) 0.1s both;
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
}

.row-b {
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) minmax(0, 1fr);
  margin-bottom: 0;
  animation: rise 0.55s var(--jj-ease) 0.2s both;
}

.panel {
  margin: 0 !important;
  border-radius: 12px;
  border-color: #e8ebf0;
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04);
  transition: box-shadow 0.28s ease, transform 0.28s ease;
  display: flex;
  flex-direction: column;
}

.panel:hover {
  box-shadow: 0 14px 30px -18px rgba(16, 24, 40, 0.32);
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid #f0f2f6;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.3px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1f2836;
}

.panel-title::before {
  content: '';
  width: 3px;
  height: 13px;
  border-radius: 2px;
  background: linear-gradient(180deg, #f2a36e, #e8722c);
}

.panel-sub {
  font-size: 11.5px;
  font-weight: 400;
  color: #93a0b2;
  letter-spacing: 0;
}

.panel-figure {
  display: flex;
  align-items: baseline;
  gap: 3px;
  color: #93a0b2;
  font-size: 11.5px;
}

.panel-figure b {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 17px;
  color: #1f2836;
  font-variant-numeric: tabular-nums;
}

.panel-body {
  padding: 16px 18px 18px;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.panel-body > .env-foot {
  margin-top: auto;
}

.panel-foot {
  padding: 0 18px 12px;
}

.panel-empty {
  padding: 34px 0;
  text-align: center;
  font-size: 12.5px;
  color: #a5afbd;
}

.status-pill {
  font-size: 11px;
  padding: 2px 9px;
  border-radius: 20px;
  background: #eaf7f0;
  color: #2f9e6e;
}

.status-pill.bad {
  background: #fdeeee;
  color: #c4534d;
}

/* ---------- 折线图 ---------- */
.chart-wrap {
  position: relative;
  padding: 10px 8px 4px;
  flex: 1;
}

.chart {
  display: block;
  width: 100%;
  height: 232px;
  overflow: visible;
}

.chart-grid line {
  stroke: #eef1f6;
  stroke-width: 1;
  stroke-dasharray: 3 5;
}

.chart-grid text {
  fill: #a9b3c1;
  font-size: 10px;
  text-anchor: end;
  font-family: 'JetBrains Mono', Consolas, monospace;
}

.chart-area {
  fill: url(#areaFill);
  animation: area-in 0.9s ease both;
}

@keyframes area-in {
  from {
    opacity: 0;
  }
}

.chart-line {
  fill: none;
  stroke: #e8722c;
  stroke-width: 2.2;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  animation: draw 1.15s cubic-bezier(0.35, 0.1, 0.2, 1) 0.12s forwards;
  filter: drop-shadow(0 4px 8px rgba(232, 114, 44, 0.24));
}

@keyframes draw {
  to {
    stroke-dashoffset: 0;
  }
}

.chart-dots circle {
  fill: #fff;
  stroke: #e8722c;
  stroke-width: 2;
  transition: r 0.18s ease;
}

.chart-dots circle.on {
  fill: #e8722c;
  stroke: #fff;
  stroke-width: 2.5;
}

.chart-cross {
  stroke: #e8b48c;
  stroke-width: 1;
  stroke-dasharray: 3 3;
}

.chart-x text {
  fill: #a9b3c1;
  font-size: 10px;
  text-anchor: middle;
  font-family: 'JetBrains Mono', Consolas, monospace;
}

.chart-tip {
  position: absolute;
  transform: translate(-50%, -140%);
  background: #1d2531;
  color: #fff;
  font-size: 11.5px;
  padding: 5px 9px;
  border-radius: 7px;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 8px 18px -8px rgba(16, 24, 40, 0.6);
}

.chart-tip b {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 13px;
  color: #f6a875;
}

.chart-tip span {
  margin-left: 5px;
  color: #93a0b2;
}

.chart-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12.5px;
  color: #adb6c3;
  pointer-events: none;
}

/* ---------- 环图 ---------- */
.donut-body {
  display: flex;
  align-items: center;
  gap: 22px;
  padding: 20px 22px 22px;
  flex: 1;
}

.donut {
  position: relative;
  width: 132px;
  height: 132px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: background 0.15s linear;
}

.donut-hole {
  position: absolute;
  inset: 17px;
  border-radius: 50%;
  background: #fff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 1px 3px rgba(16, 24, 40, 0.05);
}

.donut-hole b {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 24px;
  font-weight: 600;
  color: #1f2836;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.donut-hole i {
  font-style: normal;
  font-size: 12px;
  color: #93a0b2;
  margin-left: 1px;
}

.donut-hole span {
  margin-top: 5px;
  font-size: 11px;
  color: #93a0b2;
  letter-spacing: 0.5px;
}

.donut-legend {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.dl-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 12.5px;
  border-bottom: 1px dashed #f1f3f7;
}

.dl-item:last-child {
  border-bottom: none;
}

.dl-bar {
  width: 8px;
  height: 8px;
  border-radius: 2px;
  flex-shrink: 0;
}

.dl-name {
  color: #5c6878;
  flex: 1;
}

.dl-val {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 13px;
  color: #1f2836;
  font-variant-numeric: tabular-nums;
}

/* ---------- 动态 ---------- */
.seg {
  display: inline-flex;
  padding: 2px;
  border-radius: 8px;
  background: #f2f4f7;
}

.seg-item {
  border: none;
  background: transparent;
  padding: 4px 12px;
  font-size: 12px;
  color: #78838f;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.seg-item.on {
  background: #fff;
  color: #1f2836;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.1);
}

.feed {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.feed-item {
  display: flex;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px dashed #f1f3f7;
}

.feed-item:last-child {
  border-bottom: none;
}

.feed-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
  background: #4cc38a;
}

.feed-dot.bad {
  background: #ef6b64;
}

.feed-main {
  min-width: 0;
  flex: 1;
}

.feed-line {
  display: flex;
  align-items: baseline;
  gap: 7px;
  font-size: 12.5px;
  overflow: hidden;
}

.feed-who {
  color: #1f2836;
  font-weight: 600;
  flex-shrink: 0;
}

.feed-what {
  color: #64707f;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.feed-meta {
  margin-top: 2px;
  font-size: 11px;
  color: #a3adba;
  font-family: 'JetBrains Mono', Consolas, monospace;
}

/* ---------- 知识库 ---------- */
.stack {
  display: flex;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
  background: #f2f4f7;
}

.stack-seg {
  height: 100%;
  transition: width 0.7s cubic-bezier(0.22, 0.68, 0.24, 1);
}

.stack-seg + .stack-seg {
  border-left: 1px solid #fff;
}

.stack-seg.empty {
  background: repeating-linear-gradient(
    45deg,
    #eef1f6,
    #eef1f6 5px,
    #f7f9fb 5px,
    #f7f9fb 10px
  );
}

.kb-stats {
  display: flex;
  margin-top: 16px;
}

.kb-stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding-left: 12px;
  border-left: 2px solid #eef1f6;
}

.kb-stat:first-child {
  padding-left: 0;
  border-left: none;
}

.kb-num {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 18px;
  font-weight: 600;
  color: #1f2836;
  font-variant-numeric: tabular-nums;
}

.kb-stat.bad .kb-num {
  color: #c4534d;
}

.kb-label {
  font-size: 11.5px;
  color: #93a0b2;
}

.kb-hint {
  margin: 16px 0 0;
  font-size: 11.5px;
  line-height: 1.65;
  color: #8b97a8;
  padding: 9px 11px;
  border-radius: 8px;
  background: #f8f9fb;
}

/* ---------- 运行环境 ---------- */
.env {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 10px 14px;
  margin: 0;
  font-size: 12.5px;
}

.env dt {
  color: #8b97a8;
  white-space: nowrap;
}

.env dd {
  margin: 0;
  text-align: right;
  color: #2b3544;
}

.env dd.mono {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.env dd.danger {
  color: #c4534d;
}

.env-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed #f1f3f7;
}

/* ---------- 响应式 ---------- */
@media (max-width: 1500px) {
  .hud-metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .hud-metric:nth-child(3n) {
    border-right: none;
  }
  .hud-metric:nth-child(-n + 3) {
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }
  .row-b {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
  .row-b > .panel:first-child {
    grid-column: 1 / -1;
  }
}

@media (max-width: 1100px) {
  .row-a,
  .row-b {
    grid-template-columns: minmax(0, 1fr);
  }
  .row-b > .panel:first-child {
    grid-column: auto;
  }
  .hud-main {
    padding-bottom: 14px;
  }
  .clock-time {
    font-size: 24px;
  }
}

@media (max-width: 760px) {
  .dash {
    padding: 14px;
  }
  .hud-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .hud-metric {
    border-right: none;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }
  .donut-body {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dash,
  .hud,
  .hud-metric,
  .row-a,
  .row-b,
  .chart-line,
  .chart-area,
  .hud-scan {
    animation: none !important;
  }
  .chart-line {
    stroke-dashoffset: 0;
  }
}
</style>
