<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">系统信息</h2>
        <p class="jj-page-desc">运行时与数据库状态，用于排查部署环境和连接问题。</p>
      </div>
      <div class="jj-page-actions">
        <el-button :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
      </div>
    </div>

    <div class="jj-stats" v-loading="loading">
      <div class="jj-stat" :class="info.dbOk ? 'ok' : 'bad'">
        <div class="jj-stat-label"><el-icon><Coin /></el-icon>数据库</div>
        <div class="jj-stat-value small">{{ info.dbOk ? '连接正常' : '连接异常' }}</div>
        <div class="jj-stat-foot">延迟 {{ info.dbLatencyMs }} ms</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Timer /></el-icon>已运行</div>
        <div class="jj-stat-value small">{{ uptimeText }}</div>
        <div class="jj-stat-foot">启动于 {{ fmt(info.startAt) }}</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Cpu /></el-icon>Go 版本</div>
        <div class="jj-stat-value small">{{ info.goVersion || '-' }}</div>
        <div class="jj-stat-foot">{{ info.os }} · {{ info.cpu }} 核</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Odometer /></el-icon>协程数</div>
        <div class="jj-stat-value">{{ info.goroutines }}<span class="jj-stat-unit">个</span></div>
        <div class="jj-stat-foot">版本 v{{ info.version }}</div>
      </div>
    </div>

    <div class="grid-2">
      <div class="jj-card">
        <div class="jj-card-head"><div class="jj-card-title">服务端</div></div>
        <div class="jj-card-body">
          <dl class="jj-kv">
            <dt>后台版本</dt><dd>v{{ info.version }}</dd>
            <dt>服务器时间</dt><dd>{{ fmt(info.serverTime) }}</dd>
            <dt>时区</dt><dd>{{ info.timeZone }}</dd>
            <dt>Go 运行时</dt><dd>{{ info.goVersion }}</dd>
            <dt>系统架构</dt><dd>{{ info.os }} · {{ info.cpu }} 核</dd>
            <dt>协程数</dt><dd>{{ info.goroutines }}</dd>
            <dt>运行时长</dt><dd>{{ uptimeText }}</dd>
          </dl>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">数据库与站点</div>
          <el-tag size="small" :type="info.dbOk ? 'success' : 'danger'" effect="plain">
            {{ info.dbOk ? '连通' : '异常' }}
          </el-tag>
        </div>
        <div class="jj-card-body">
          <dl class="jj-kv">
            <dt>连接状态</dt><dd>{{ info.dbOk ? '正常' : info.dbError || '异常' }}</dd>
            <dt>响应延迟</dt><dd>{{ info.dbLatencyMs }} ms</dd>
            <dt>MySQL 版本</dt><dd>{{ info.dbVersion || '-' }}</dd>
            <dt>站点名称</dt><dd>{{ info.siteName || '-' }}</dd>
            <dt>维护模式</dt>
            <dd>
              <el-tag size="small" :type="info.maintenance ? 'warning' : 'info'" effect="plain">
                {{ info.maintenance ? '已开启' : '已关闭' }}
              </el-tag>
            </dd>
            <dt>当前账号</dt><dd>{{ info.accounts?.current }}（{{ info.accounts?.role }}）</dd>
          </dl>
        </div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">部署提示</div>
      </div>
      <div class="jj-card-body">
        <ul class="tips">
          <li>数据库连接来自环境变量 <code>MYSQL_USERNAME</code> / <code>MYSQL_PASSWORD</code> / <code>MYSQL_ADDRESS</code> / <code>MYSQL_DATABASE</code>。</li>
          <li>登录令牌签名密钥取 <code>ADMIN_SECRET</code>，未配置时回退到数据库口令派生值；多副本部署请务必显式配置。</li>
          <li>首次启动会自动建表并写入内置角色与初始管理员，已存在则不会重复创建。</li>
          <li>修改初始密码后 <code>mustChangePwd</code> 会置为 false，看板上的安全提醒随之消失。</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const reloadTick = inject('jjReloadTick', ref(0))
const loading = ref(false)
const info = reactive({
  version: '',
  goVersion: '',
  os: '',
  cpu: 0,
  goroutines: 0,
  startAt: '',
  uptimeSeconds: 0,
  serverTime: '',
  timeZone: '',
  dbOk: false,
  dbLatencyMs: 0,
  dbError: '',
  dbVersion: '',
  siteName: '',
  maintenance: false,
  accounts: {}
})

const uptimeText = computed(() => {
  const s = info.uptimeSeconds
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d) return `${d} 天 ${h} 小时`
  if (h) return `${h} 小时 ${m} 分`
  return `${m} 分钟`
})

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.systemInfo()
    Object.assign(info, data)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.jj-stat-value.small {
  font-size: 18px;
  font-weight: 600;
  margin-top: 10px;
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

.tips {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 2;
  color: #5c6675;
}

.tips code {
  background: #f2f4f7;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 12px;
  color: #7a5a3a;
}

@media (max-width: 1000px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
