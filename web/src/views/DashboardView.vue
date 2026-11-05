<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h1 class="jj-page-title">仪表盘</h1>
        <p class="jj-page-desc">服务运行概览与数据快照，数据来自实时接口</p>
      </div>
      <el-button :loading="loading" @click="load">
        <el-icon><Refresh /></el-icon>
        <span style="margin-left: 4px">刷新</span>
      </el-button>
    </div>

    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      show-icon
      :title="'数据加载失败：' + error"
      style="margin-bottom: 16px"
    />

    <el-skeleton v-if="loading && !ov" :rows="6" animated />

    <template v-else>
      <div class="jj-stats">
        <div class="jj-stat">
          <div class="jj-stat-label"><el-icon><Odometer /></el-icon> 当前计数</div>
          <div class="jj-stat-value">{{ ov?.count ?? '-' }}</div>
          <div class="jj-stat-foot">主记录 ID = {{ ov?.defaultCounterId ?? 1 }}</div>
        </div>
        <div class="jj-stat">
          <div class="jj-stat-label"><el-icon><Tickets /></el-icon> 记录总数</div>
          <div class="jj-stat-value">{{ ov?.total ?? '-' }}</div>
          <div class="jj-stat-foot">Counters 表行数</div>
        </div>
        <div class="jj-stat">
          <div class="jj-stat-label"><el-icon><Timer /></el-icon> 数据库延迟</div>
          <div class="jj-stat-value">
            {{ ov?.dbLatencyMs ?? '-' }}<span class="jj-stat-unit">ms</span>
          </div>
          <div class="jj-stat-foot">{{ ov?.dbOk ? '连接正常' : '连接异常' }}</div>
        </div>
        <div class="jj-stat">
          <div class="jj-stat-label"><el-icon><Clock /></el-icon> 运行时长</div>
          <div class="jj-stat-value">{{ uptimeText }}</div>
          <div class="jj-stat-foot">当前容器实例</div>
        </div>
      </div>

      <div class="jj-grid">
        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">服务信息</div>
            <el-tag :type="ov?.dbOk ? 'success' : 'danger'" size="small" effect="plain">
              {{ ov?.dbOk ? '运行中' : '数据库异常' }}
            </el-tag>
          </div>
          <div class="jj-card-body">
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="服务版本">{{ ov?.version || '-' }}</el-descriptions-item>
              <el-descriptions-item label="服务器时间">
                <span class="jj-mono">{{ fmtTime(ov?.serverTime) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="时区">
                <span class="jj-mono">{{ ov?.timeZone || '-' }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="最近更新">
                <span class="jj-mono">{{ fmtTime(ov?.lastUpdatedAt) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="访问控制">
                <el-tag v-if="ov?.authRequired" type="success" size="small">已启用</el-tag>
                <el-tag v-else type="warning" size="small">未启用</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="数据库错误">
                <span :class="{ 'jj-err': ov?.dbError }">{{ ov?.dbError || '无' }}</span>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">最近记录</div>
            <el-button text type="primary" size="small" @click="$router.push('/counters')">
              前往管理
            </el-button>
          </div>
          <div class="jj-card-body">
            <el-table :data="recent" size="small" empty-text="暂无数据">
              <el-table-column prop="id" label="ID" width="64" />
              <el-table-column prop="count" label="计数" width="80" />
              <el-table-column label="更新时间">
                <template #default="{ row }">
                  <span class="jj-mono">{{ fmtTime(row.updatedAt) }}</span>
                </template>
              </el-table-column>
            </el-table>

            <div class="jj-quick">
              <el-button type="primary" :loading="acting" @click="quickIncr">
                <el-icon><Plus /></el-icon>
                <span style="margin-left: 4px">计数 +1</span>
              </el-button>
              <el-button :loading="loading" @click="load">重新加载</el-button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const reloadTick = inject('jjReloadTick', ref(0))
const onLine = inject('jjOnline', ref(true))

const loading = ref(false)
const acting = ref(false)
const error = ref('')
const ov = ref(null)
const recent = ref([])

const uptimeText = computed(() => {
  const s = ov.value?.uptimeSeconds
  if (s === undefined || s === null) return '-'
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m`
  if (s < 86400) return `${Math.floor(s / 3600)}h${Math.floor((s % 3600) / 60)}m`
  return `${Math.floor(s / 86400)}d`
})

function fmtTime(v) {
  if (!v) return '-'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [o, c] = await Promise.all([api.overview(), api.listCounters()])
    ov.value = o
    recent.value = (c.list || []).slice(0, 6)
    onLine.value = true
  } catch (e) {
    error.value = e.message || '未知错误'
    onLine.value = false
  } finally {
    loading.value = false
  }
}

async function quickIncr() {
  if (!ov.value) return
  acting.value = true
  try {
    const r = await api.incrCounter(ov.value.defaultCounterId || 1, 1)
    ElMessage.success(`已自增，当前值 ${r.count}`)
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    acting.value = false
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.jj-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.jj-quick {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.jj-err {
  color: #f56c6c;
}

@media (max-width: 1100px) {
  .jj-grid {
    grid-template-columns: 1fr;
  }
}
</style>
