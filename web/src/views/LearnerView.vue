<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">学员与学习记录</h2>
        <p class="jj-page-desc">
          学员在小程序授权后自动建档，学习进度由小程序上报。这里按专业与水平筛选，点进详情看逐门课程的进度。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" @click="openCreate">
          <el-icon><Plus /></el-icon>手动建档
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="demoCount > 0"
      class="demo-tip"
      type="info"
      :closable="false"
      show-icon
    >
      <template #title>当前有 {{ demoCount }} 名演示学员，用于小程序接入前预览页面效果</template>
      <template #default>
        演示学员的 openId 以 demo- 开头，与真实微信用户不会冲突。
        <el-button v-if="canEdit" link type="primary" size="small" @click="cleanDemo">一键清理</el-button>
      </template>
    </el-alert>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><User /></el-icon>学员总数</div>
        <div class="jj-stat-value">{{ normalCount + disabledCount }}<span class="jj-stat-unit">人</span></div>
        <div class="jj-stat-foot">正常 {{ normalCount }} · 禁用 {{ disabledCount }}</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><Sunny /></el-icon>今日活跃</div>
        <div class="jj-stat-value">{{ activeToday }}<span class="jj-stat-unit">人</span></div>
        <div class="jj-stat-foot">按最近活跃时间统计</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Plus /></el-icon>今日新增</div>
        <div class="jj-stat-value">{{ newToday }}<span class="jj-stat-unit">人</span></div>
        <div class="jj-stat-foot">小程序授权建档</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Medal /></el-icon>水平分布</div>
        <div class="jj-stat-value">{{ levelCounts[1] || 0 }}<span class="jj-stat-unit">入门</span></div>
        <div class="jj-stat-foot">
          进阶 {{ levelCounts[2] || 0 }} · 高级 {{ levelCounts[3] || 0 }}
        </div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">学员列表</div>
        <div class="head-tools">
          <el-select v-model="filters.majorId" placeholder="全部专业" clearable style="width: 140px" @change="reload">
            <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
          <el-select v-model="filters.level" placeholder="全部水平" clearable style="width: 120px" @change="reload">
            <el-option label="入门" :value="1" />
            <el-option label="进阶" :value="2" />
            <el-option label="高级" :value="3" />
          </el-select>
          <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 120px" @change="reload">
            <el-option label="正常" :value="1" />
            <el-option label="已禁用" :value="0" />
          </el-select>
          <el-input v-model="filters.keyword" placeholder="搜索昵称 / 手机号" clearable style="width: 190px" @input="reload">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button text :loading="loading" @click="load">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="学员" min-width="240">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="34" :src="row.avatar || undefined">{{ (row.nickname || '学')[0] }}</el-avatar>
              <div class="user-meta">
                <div class="user-name">
                  {{ row.nickname }}
                  <span v-if="row.demo" class="demo-tag">演示</span>
                  <span v-if="row.status === 0" class="jj-tag offline">已禁用</span>
                </div>
                <div class="jj-sub">{{ row.phone || row.openId }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="专业" width="120">
          <template #default="{ row }">
            <span class="chip" :class="{ muted: !row.majorId }">{{ row.majorName }}</span>
          </template>
        </el-table-column>
        <el-table-column label="水平" width="90">
          <template #default="{ row }">
            <span class="level" :class="'lv' + row.level">{{ row.levelText }}</span>
          </template>
        </el-table-column>
        <el-table-column label="课程 / 完成" width="110" align="right">
          <template #default="{ row }">
            <span class="num">{{ row.courseCount }}</span>
            <span class="jj-sub"> / {{ row.finishedCount }}</span>
          </template>
        </el-table-column>
        <el-table-column label="平均进度" width="160">
          <template #default="{ row }">
            <el-progress
              :percentage="row.avgProgress"
              :stroke-width="8"
              :color="progressColor(row.avgProgress)"
            />
          </template>
        </el-table-column>
        <el-table-column label="最近学习" width="150">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.lastStudyAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/learners/${row.id}`)">详情</el-button>
            <template v-if="canEdit">
              <el-button v-if="row.status === 1" link type="warning" @click="toggle(row, 0)">禁用</el-button>
              <el-button v-else link type="success" @click="toggle(row, 1)">恢复</el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">
            还没有学员。小程序接入后，用户授权即自动建档；也可点右上角「手动建档」造一条测试数据。
          </div>
        </template>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          background
          @current-change="load"
          @size-change="reload"
        />
      </div>
    </div>

    <el-drawer v-model="drawer" title="手动建档" size="480px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" placeholder="学员昵称" maxlength="64" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="form.phone" placeholder="选填" maxlength="20" />
        </el-form-item>
        <el-form-item label="专业">
          <el-select v-model="form.majorId" placeholder="未选择" clearable style="width: 100%">
            <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="水平">
          <el-radio-group v-model="form.level">
            <el-radio :value="1">入门</el-radio>
            <el-radio :value="2">进阶</el-radio>
            <el-radio :value="3">高级</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" maxlength="200" show-word-limit placeholder="选填" />
        </el-form-item>
      </el-form>
      <div class="jj-hint">手动建档的 openId 会以 manual- 开头，小程序端真实登录不会与之冲突。</div>
      <template #footer>
        <el-button @click="drawer = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">创建</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = canWrite('learners')

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const majors = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const drawer = ref(false)
const formRef = ref(null)

const normalCount = ref(0)
const disabledCount = ref(0)
const newToday = ref(0)
const activeToday = ref(0)
const demoCount = ref(0)
const levelCounts = ref({})

const filters = reactive({ majorId: null, level: null, status: null, keyword: '' })

const form = reactive({ nickname: '', phone: '', majorId: null, level: 1, remark: '' })
const rules = { nickname: [{ required: true, message: '请输入学员昵称', trigger: 'blur' }] }

function fmt(v) {
  if (!v) return '从未学习'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

function progressColor(v) {
  if (v >= 100) return '#16a34a'
  if (v >= 60) return '#2563eb'
  return '#e8722c'
}

async function load() {
  loading.value = true
  try {
    const data = await api.learners.list({
      page: page.value,
      size: size.value,
      majorId: filters.majorId || undefined,
      level: filters.level || undefined,
      status: filters.status === null || filters.status === '' ? undefined : filters.status,
      keyword: filters.keyword || undefined
    })
    list.value = data.list || []
    total.value = data.total || 0
    majors.value = data.majors || []
    normalCount.value = data.normalCount || 0
    disabledCount.value = data.disabledCount || 0
    newToday.value = data.newToday || 0
    activeToday.value = data.activeToday || 0
    demoCount.value = data.demoCount || 0
    levelCounts.value = data.levelCounts || {}
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function openCreate() {
  Object.assign(form, { nickname: '', phone: '', majorId: filters.majorId || null, level: 1, remark: '' })
  formRef.value?.clearValidate()
  drawer.value = true
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await api.learners.save({ ...form })
    ElMessage.success('已创建学员档案')
    drawer.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function toggle(row, status) {
  try {
    await api.learners.status(row.id, status)
    ElMessage.success(status === 1 ? '已恢复' : '已禁用，该学员无法继续学习')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除学员「${row.nickname}」？其 ${row.courseCount} 条学习记录会一并删除，且不可恢复。`,
      '删除学员',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.learners.remove(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function cleanDemo() {
  try {
    await ElMessageBox.confirm('将删除全部 openId 以 demo- 开头的演示学员及其学习记录，真实学员不受影响。', '清理演示数据', {
      confirmButtonText: '清理', cancelButtonText: '取消', type: 'warning'
    })
  } catch {
    return
  }
  try {
    const data = await api.learners.cleanDemo()
    ElMessage.success(`已清理 ${data.removed} 名演示学员`)
    reload()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.head-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.demo-tip {
  margin-bottom: 14px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-meta {
  min-width: 0;
}

.user-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 14px;
}

.demo-tag {
  padding: 0 6px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  color: #64748b;
  background: rgba(148, 163, 184, 0.2);
}

.chip {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 8px;
  font-size: 12px;
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
}

.chip.muted {
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
}

.level {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
}

.level.lv1 {
  background: rgba(22, 163, 74, 0.12);
  color: #16a34a;
}

.level.lv2 {
  background: rgba(37, 99, 235, 0.12);
  color: #2563eb;
}

.level.lv3 {
  background: rgba(220, 38, 38, 0.12);
  color: #dc2626;
}

.num {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
}

.jj-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
}

.jj-tag.offline {
  background: rgba(220, 38, 38, 0.12);
  color: #dc2626;
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding: 14px 4px 2px;
}
</style>
