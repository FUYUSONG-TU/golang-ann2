<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">操作日志</h2>
        <p class="jj-page-desc">
          登录与写操作都会自动留痕，用于排查问题和追溯变更。日志只增不改，可整体清空。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canManage" type="danger" plain @click="clearAll">
          <el-icon><Delete /></el-icon>清空日志
        </el-button>
      </div>
    </div>

    <div class="jj-card">
      <el-tabs v-model="tab" class="log-tabs" @tab-change="onTabChange">
        <el-tab-pane label="登录日志" name="login" />
        <el-tab-pane label="操作日志" name="operation" />
      </el-tabs>

      <div class="jj-toolbar">
        <el-input
          v-model="query.keyword"
          :placeholder="tab === 'login' ? '搜索账号 / IP' : '搜索账号 / 动作 / 路径'"
          clearable
          style="width: 240px"
          @keyup.enter="search"
          @clear="search"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-if="tab === 'operation'" v-model="query.module" clearable placeholder="全部模块" style="width: 140px" @change="search">
          <el-option label="账号" value="users" />
          <el-option label="角色" value="roles" />
          <el-option label="计数" value="counters" />
          <el-option label="日志" value="logs" />
          <el-option label="设置" value="configs" />
          <el-option label="登录态" value="auth" />
        </el-select>
        <el-select v-model="query.success" style="width: 130px" @change="search">
          <el-option label="全部结果" :value="-1" />
          <el-option label="成功" :value="1" />
          <el-option label="失败" :value="0" />
        </el-select>
        <el-button type="primary" plain @click="search">查询</el-button>
        <el-button @click="reset">重置</el-button>
        <span class="grow" />
        <el-button text :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>

      <el-table v-if="tab === 'login'" :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="时间" width="170">
          <template #default="{ row }"><span class="jj-sub">{{ fmt(row.createdAt) }}</span></template>
        </el-table-column>
        <el-table-column prop="username" label="账号" width="140" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column label="结果" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="plain">
              {{ row.success ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="说明" min-width="140" />
        <el-table-column prop="userAgent" label="客户端" min-width="220" show-overflow-tooltip />
        <template #empty><div class="jj-empty">暂无登录日志</div></template>
      </el-table>

      <el-table v-else :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="时间" width="170">
          <template #default="{ row }"><span class="jj-sub">{{ fmt(row.createdAt) }}</span></template>
        </el-table-column>
        <el-table-column label="账号" width="130">
          <template #default="{ row }">{{ row.username || '系统' }}</template>
        </el-table-column>
        <el-table-column label="模块" width="100">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" type="info">{{ moduleName(row.module) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="动作" width="150" />
        <el-table-column label="请求" min-width="240">
          <template #default="{ row }">
            <span class="jj-method">{{ row.method }}</span>
            <span class="jj-mono">{{ row.path }}</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="plain">
              {{ row.success ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column label="详情" min-width="180" show-overflow-tooltip>
          <template #default="{ row }"><span class="jj-sub">{{ row.detail || '—' }}</span></template>
        </el-table-column>
        <template #empty><div class="jj-empty">暂无操作日志</div></template>
      </el-table>

      <div class="jj-pager">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.size"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="load"
          @current-change="load"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canManage = computed(() => canWrite('logs'))

const tab = ref('login')
const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ keyword: '', module: '', success: -1, page: 1, size: 10 })

const moduleNames = {
  auth: '登录态',
  users: '账号',
  roles: '角色',
  counters: '计数',
  logs: '日志',
  configs: '设置',
  mp: '小程序'
}

function moduleName(code) {
  return moduleNames[code] || code || '—'
}

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

async function load() {
  loading.value = true
  try {
    const params = {
      page: query.page,
      size: query.size,
      keyword: query.keyword,
      success: query.success
    }
    const data =
      tab.value === 'login'
        ? await api.logs.login(params)
        : await api.logs.operation({ ...params, module: query.module })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function search() {
  query.page = 1
  load()
}

function reset() {
  query.keyword = ''
  query.module = ''
  query.success = -1
  search()
}

function onTabChange() {
  list.value = []
  query.page = 1
  load()
}

async function clearAll() {
  try {
    await ElMessageBox.confirm(
      '将清空全部登录日志与操作日志，历史记录不可恢复。确认继续？',
      '清空日志',
      { confirmButtonText: '确认清空', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.logs.clear('all')
    ElMessage.success('日志已清空')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.log-tabs {
  padding: 4px 18px 0;
}

.log-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.jj-method {
  display: inline-block;
  min-width: 48px;
  padding: 0 6px;
  margin-right: 8px;
  border-radius: 3px;
  background: #eef2f7;
  color: #5c6675;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}
</style>
