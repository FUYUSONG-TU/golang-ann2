<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h1 class="jj-page-title">计数管理</h1>
        <p class="jj-page-desc">管理 Counters 表中的记录，所有改动即时写入数据库</p>
      </div>
      <div class="jj-toolbar">
        <el-input
          v-model="keyword"
          placeholder="按 ID 或数值筛选"
          clearable
          style="width: 200px"
          prefix-icon="Search"
        />
        <el-button :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
        </el-button>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>
          <span style="margin-left: 4px">新建记录</span>
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      show-icon
      :title="'加载失败：' + error"
      style="margin-bottom: 16px"
    />

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">数据记录</div>
        <span class="jj-count-hint">共 {{ filtered.length }} 条</span>
      </div>

      <el-table
        v-loading="loading"
        :data="paged"
        stripe
        empty-text="暂无记录，点击右上角新建"
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="计数" width="120">
          <template #default="{ row }">
            <span class="jj-count-cell">{{ row.count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">
            <span class="jj-mono">{{ fmtTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">
            <span class="jj-mono">{{ fmtTime(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="316" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="incr(row, 1)">+1</el-button>
            <el-button size="small" plain @click="openIncr(row)">自增…</el-button>
            <el-button size="small" plain @click="openSet(row)">改值</el-button>
            <el-button size="small" type="danger" plain @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="jj-pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          :page-sizes="[10, 20, 50]"
          :total="filtered.length"
          layout="total, sizes, prev, pager, next"
          background
        />
      </div>
    </div>

    <el-dialog v-model="dlg.visible" :title="dlgTitle" width="380px" @closed="resetDlg">
      <div class="jj-dlg-body">
        <div class="jj-dlg-label">{{ dlg.mode === 'incr' ? '自增步长' : '数值' }}</div>
        <el-input-number
          v-model="dlg.value"
          :min="dlg.mode === 'incr' ? 1 : 0"
          :max="1000000"
          size="large"
          style="width: 100%"
        />
        <div class="jj-dlg-tip">
          <template v-if="dlg.mode === 'create'">将新建一条初始值为 {{ dlg.value }} 的记录</template>
          <template v-else-if="dlg.mode === 'set'">
            ID={{ dlg.id }} 的计数将被覆盖为 {{ dlg.value }}
          </template>
          <template v-else>ID={{ dlg.id }} 将增加 {{ dlg.value }}</template>
        </div>
      </div>
      <template #footer>
        <el-button @click="dlg.visible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitDlg">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'

const reloadTick = inject('jjReloadTick', ref(0))

const rows = ref([])
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const keyword = ref('')
const page = ref(1)
const size = ref(10)

const dlg = reactive({
  visible: false,
  mode: 'create',
  id: null,
  value: 1
})

const dlgTitle = computed(() => {
  if (dlg.mode === 'create') return '新建记录'
  if (dlg.mode === 'set') return '修改计数值'
  return '按步长自增'
})

const filtered = computed(() => {
  const k = keyword.value.trim()
  if (!k) return rows.value
  return rows.value.filter((r) => String(r.id).includes(k) || String(r.count).includes(k))
})

const paged = computed(() => {
  const start = (page.value - 1) * size.value
  return filtered.value.slice(start, start + size.value)
})

watch(keyword, () => {
  page.value = 1
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
    const data = await api.listCounters()
    rows.value = data.list || []
    const maxPage = Math.max(1, Math.ceil(filtered.value.length / size.value))
    if (page.value > maxPage) page.value = maxPage
  } catch (e) {
    error.value = e.message || '未知错误'
  } finally {
    loading.value = false
  }
}

function resetDlg() {
  dlg.id = null
  dlg.value = 1
  dlg.mode = 'create'
}

function openCreate() {
  dlg.mode = 'create'
  dlg.value = 1
  dlg.id = null
  dlg.visible = true
}

function openSet(row) {
  dlg.mode = 'set'
  dlg.id = row.id
  dlg.value = row.count
  dlg.visible = true
}

function openIncr(row) {
  dlg.mode = 'incr'
  dlg.id = row.id
  dlg.value = 1
  dlg.visible = true
}

async function submitDlg() {
  submitting.value = true
  try {
    if (dlg.mode === 'create') {
      await api.createCounter(dlg.value)
      ElMessage.success('记录已创建')
    } else if (dlg.mode === 'set') {
      await api.setCounter(dlg.id, dlg.value)
      ElMessage.success('已更新')
    } else {
      const r = await api.incrCounter(dlg.id, dlg.value)
      ElMessage.success(`已自增，当前值 ${r.count}`)
    }
    dlg.visible = false
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function incr(row, step) {
  try {
    const r = await api.incrCounter(row.id, step)
    ElMessage.success(`ID=${row.id} 当前值 ${r.count}`)
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除 ID=${row.id}（计数 ${row.count}）的记录？此操作不可恢复。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await api.deleteCounter(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.jj-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.jj-count-hint {
  font-size: 12.5px;
  color: #8a94a6;
}

.jj-count-cell {
  font-size: 16px;
  font-weight: 600;
  color: var(--jj-orange);
  font-variant-numeric: tabular-nums;
}

.jj-pager {
  display: flex;
  justify-content: flex-end;
  padding: 14px 18px;
  border-top: 1px solid var(--jj-line);
}

.jj-dlg-body {
  padding: 4px 2px 0;
}

.jj-dlg-label {
  font-size: 13px;
  color: #5c6675;
  margin-bottom: 8px;
}

.jj-dlg-tip {
  margin-top: 12px;
  font-size: 12.5px;
  color: #8a94a6;
  background: #f7f8fa;
  border-radius: 6px;
  padding: 9px 12px;
  line-height: 1.6;
}
</style>
