<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">计数管理</h2>
        <p class="jj-page-desc">
          模板自带的示例业务数据，用于验证前后端与数据库链路。小程序端接入后可替换为真实业务表。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="create">
          <el-icon><Plus /></el-icon>新建记录
        </el-button>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Coin /></el-icon>主计数器</div>
        <div class="jj-stat-value">{{ mainCount }}<span class="jj-stat-unit">次</span></div>
        <div class="jj-stat-foot">对应 id = 1 的记录</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><Tickets /></el-icon>记录总数</div>
        <div class="jj-stat-value">{{ list.length }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">Counters 表</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Timer /></el-icon>最近更新</div>
        <div class="jj-stat-value small">{{ lastUpdated }}</div>
        <div class="jj-stat-foot">按更新时间取最新一条</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Link /></el-icon>对外接口</div>
        <div class="jj-stat-value small">/api/count</div>
        <div class="jj-stat-foot">GET 读取 · POST 自增/清零</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">计数记录</div>
        <el-button text :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="当前计数" width="140">
          <template #default="{ row }">
            <span class="count-value">{{ row.count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="{ row }">
            <template v-if="canEdit">
              <el-button link type="primary" @click="incr(row)">+1</el-button>
              <el-button link type="primary" @click="openSet(row)">设为指定值</el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
            <span v-else class="jj-sub">只读</span>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">暂无计数记录</div>
        </template>
      </el-table>
    </div>

    <el-dialog v-model="setVisible" title="设置计数值" width="380px">
      <el-form label-width="80px">
        <el-form-item label="计数值">
          <el-input-number v-model="setValue" :min="0" :max="999999999" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="setVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitSet">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = computed(() => canWrite('counters'))

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const setVisible = ref(false)
const setValue = ref(0)
const setTarget = ref(null)

const mainCount = computed(() => list.value.find((i) => i.id === 1)?.count ?? 0)
const lastUpdated = computed(() => {
  if (!list.value.length) return '-'
  const latest = list.value.reduce((a, b) => (new Date(a.updatedAt) > new Date(b.updatedAt) ? a : b))
  return fmt(latest.updatedAt)
})

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.counters.list()
    list.value = data.list || []
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function incr(row) {
  try {
    await api.counters.incr(row.id, 1)
    ElMessage.success('已 +1')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

function openSet(row) {
  setTarget.value = row
  setValue.value = row.count
  setVisible.value = true
}

async function submitSet() {
  saving.value = true
  try {
    await api.counters.set(setTarget.value.id, setValue.value)
    ElMessage.success('已保存')
    setVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function create() {
  try {
    await api.counters.create(0)
    ElMessage.success('已新建一条记录')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(`确认删除 id=${row.id} 的计数记录？`, '删除记录', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.counters.remove(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.jj-stat-value.small {
  font-size: 17px;
  font-weight: 600;
  margin-top: 12px;
}

.count-value {
  font-size: 16px;
  font-weight: 650;
  color: var(--jj-orange);
  font-variant-numeric: tabular-nums;
}
</style>
