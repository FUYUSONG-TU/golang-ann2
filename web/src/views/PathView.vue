<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">学习路线</h2>
        <p class="jj-page-desc">
          路线由「阶段 → 课程」组成。课程只被引用、不被拥有，同一门课可以出现在多条路线里。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建路线
        </el-button>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Guide /></el-icon>路线总数</div>
        <div class="jj-stat-value">{{ total }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">按当前筛选条件统计</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><CircleCheck /></el-icon>已发布</div>
        <div class="jj-stat-value">{{ statusCounts.published || 0 }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">小程序端可见</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><EditPen /></el-icon>草稿</div>
        <div class="jj-stat-value">{{ statusCounts.draft || 0 }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">阶段未配齐，暂不对外</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Connection /></el-icon>课程引用</div>
        <div class="jj-stat-value">{{ courseRefs }}<span class="jj-stat-unit">次</span></div>
        <div class="jj-stat-foot">同一门课可被多条路线复用</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">路线列表</div>
        <div class="head-tools">
          <el-select
            v-model="filters.majorId"
            placeholder="全部专业"
            clearable
            style="width: 150px"
            @change="reload"
          >
            <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
          <el-select
            v-model="filters.status"
            placeholder="全部状态"
            clearable
            style="width: 130px"
            @change="reload"
          >
            <el-option label="草稿" value="draft" />
            <el-option label="审核中" value="pending" />
            <el-option label="已发布" value="published" />
            <el-option label="已下架" value="offline" />
          </el-select>
          <el-input
            v-model="filters.keyword"
            placeholder="搜索路线名称"
            clearable
            style="width: 190px"
            @input="reload"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button text :loading="loading" @click="load">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="学习路线" min-width="280">
          <template #default="{ row }">
            <div class="path-name">{{ row.name }}</div>
            <div class="jj-sub">{{ row.summary || '暂无简介' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="所属专业" width="120">
          <template #default="{ row }">
            <span class="major-chip" :class="{ general: row.majorId === 0 }">{{ row.majorName }}</span>
          </template>
        </el-table-column>
        <el-table-column label="难度" width="90">
          <template #default="{ row }">
            <span class="level" :class="'lv' + row.difficulty">{{ difficultyText(row.difficulty) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="阶段" width="80" align="right">
          <template #default="{ row }">
            <span class="count-num">{{ row.stageCount }}</span>
          </template>
        </el-table-column>
        <el-table-column label="课程" width="80" align="right">
          <template #default="{ row }">
            <span class="count-num">{{ row.courseCount }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="jj-tag" :class="row.status">{{ statusText(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="150">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/paths/${row.id}`)">
              {{ canEdit ? '编排' : '查看' }}
            </el-button>
            <template v-if="canEdit">
              <el-button
                v-if="row.status !== 'published'"
                link
                type="success"
                @click="changeStatus(row, 'published')"
              >
                发布
              </el-button>
              <el-button v-else link type="warning" @click="changeStatus(row, 'offline')">
                下架
              </el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">还没有学习路线，先建一条把课程串起来</div>
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

    <el-drawer
      v-model="drawer"
      :title="form.id ? '编辑路线' : '新建路线'"
      size="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="路线名称" prop="name">
          <el-input v-model="form.name" placeholder="如：走心机入门 30 天" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="所属专业">
          <el-select v-model="form.majorId" placeholder="不限专业（通用路线）" clearable style="width: 100%">
            <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
          <div class="jj-sub field-tip">留空表示通用路线，任何专业的学员都能看到</div>
        </el-form-item>
        <el-form-item label="难度">
          <el-radio-group v-model="form.difficulty">
            <el-radio :value="1">入门</el-radio>
            <el-radio :value="2">进阶</el-radio>
            <el-radio :value="3">高级</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="预计学时">
          <el-input-number v-model="form.hours" :min="0" :max="9999" />
          <span class="jj-sub unit">小时</span>
        </el-form-item>
        <el-form-item label="路线简介">
          <el-input
            v-model="form.summary"
            type="textarea"
            :rows="4"
            maxlength="200"
            show-word-limit
            placeholder="学完这条路线能做什么"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawer = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">
          {{ form.id ? '保存' : '创建并去编排' }}
        </el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const router = useRouter()
const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = canWrite('paths')

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const majors = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const drawer = ref(false)
const formRef = ref(null)

const filters = reactive({ majorId: null, status: '', keyword: '' })
const statusCounts = ref({ draft: 0, pending: 0, published: 0, offline: 0 })

const form = reactive({
  id: 0,
  name: '',
  majorId: null,
  difficulty: 1,
  hours: 0,
  summary: ''
})

const rules = {
  name: [{ required: true, message: '请输入路线名称', trigger: 'blur' }]
}

const courseRefs = computed(() => list.value.reduce((sum, item) => sum + (item.courseCount || 0), 0))

function difficultyText(v) {
  return { 1: '入门', 2: '进阶', 3: '高级' }[v] || '入门'
}

function statusText(v) {
  return { draft: '草稿', pending: '审核中', published: '已发布', offline: '已下架' }[v] || v
}

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.paths.list({
      page: page.value,
      size: size.value,
      majorId: filters.majorId || undefined,
      status: filters.status || undefined,
      keyword: filters.keyword || undefined
    })
    list.value = data.list || []
    total.value = data.total || 0
    majors.value = data.majors || []
    if (data.statusCounts) statusCounts.value = data.statusCounts
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
  Object.assign(form, {
    id: 0,
    name: '',
    majorId: filters.majorId || null,
    difficulty: 1,
    hours: 0,
    summary: ''
  })
  formRef.value?.clearValidate()
  drawer.value = true
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const data = await api.paths.save({ ...form })
    if (form.id) {
      ElMessage.success('已保存')
      drawer.value = false
      load()
    } else {
      ElMessage.success('路线已创建，接着加阶段和课程')
      drawer.value = false
      router.push(`/paths/${data.id}`)
    }
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function changeStatus(row, status) {
  try {
    await api.paths.status(row.id, status)
    ElMessage.success(status === 'published' ? '已发布' : '已下架')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除学习路线「${row.name}」？其下 ${row.stageCount} 个阶段与 ${row.courseCount} 门课程引用会一并清除，课程本身不受影响。`,
      '删除学习路线',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.paths.remove(row.id)
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
.head-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.path-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 2px;
}

.major-chip {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 8px;
  font-size: 12px;
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
}

.major-chip.general {
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
}

.count-num {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
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

.jj-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
}

.jj-tag.draft {
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
}

.jj-tag.pending {
  background: rgba(245, 158, 11, 0.14);
  color: #d97706;
}

.jj-tag.published {
  background: rgba(22, 163, 74, 0.12);
  color: #16a34a;
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

.unit {
  margin-left: 8px;
}

.field-tip {
  margin-top: 4px;
}
</style>
