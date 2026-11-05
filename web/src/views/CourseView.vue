<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">课程管理</h2>
        <p class="jj-page-desc">
          课程隶属专业，章节在「课程编辑」里维护。新建课程为草稿，至少有一个章节才能发布。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建课程
        </el-button>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Reading /></el-icon>课程总数</div>
        <div class="jj-stat-value">{{ total }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">按当前筛选条件统计</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><CircleCheck /></el-icon>已发布</div>
        <div class="jj-stat-value">{{ statusCounts.published || 0 }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">小程序端可见</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><EditPen /></el-icon>草稿</div>
        <div class="jj-stat-value">{{ statusCounts.draft || 0 }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">尚未对外展示</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Bottom /></el-icon>已下架</div>
        <div class="jj-stat-value">{{ statusCounts.offline || 0 }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">历史内容，学员不可见</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">课程列表</div>
        <div class="head-tools">
          <el-select
            v-model="filters.majorId"
            placeholder="全部专业"
            clearable
            style="width: 160px"
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
            placeholder="搜索课程名称"
            clearable
            style="width: 200px"
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
        <el-table-column label="课程" min-width="260">
          <template #default="{ row }">
            <div class="course-name">{{ row.name }}</div>
            <div class="jj-sub">{{ row.summary || '暂无简介' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="专业" width="120">
          <template #default="{ row }">
            <span class="jj-sub">{{ row.majorName }}</span>
          </template>
        </el-table-column>
        <el-table-column label="难度" width="90">
          <template #default="{ row }">
            <span class="level" :class="'lv' + row.difficulty">{{ difficultyText(row.difficulty) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="章节" width="80" align="right">
          <template #default="{ row }">
            <span class="count-num">{{ row.chapterCount }}</span>
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
        <el-table-column label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/courses/${row.id}`)">
              {{ canEdit ? '编辑' : '查看' }}
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
          <div class="jj-empty">没有符合条件的课程</div>
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
      :title="form.id ? '编辑课程' : '新建课程'"
      size="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="课程名称" prop="name">
          <el-input v-model="form.name" placeholder="如：G74 端面啄钻循环" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="所属专业" prop="majorId">
          <el-select v-model="form.majorId" placeholder="请选择专业" style="width: 100%">
            <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="难度">
          <el-radio-group v-model="form.difficulty">
            <el-radio :value="1">入门</el-radio>
            <el-radio :value="2">进阶</el-radio>
            <el-radio :value="3">高级</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="课时">
          <el-input-number v-model="form.hours" :min="0" :max="999" />
          <span class="jj-sub unit">小时</span>
        </el-form-item>
        <el-form-item label="课程简介">
          <el-input
            v-model="form.summary"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="这门课讲什么、学完能做什么"
          />
        </el-form-item>
        <el-form-item v-if="form.id" label="发布状态">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="草稿" value="draft" />
            <el-option label="审核中" value="pending" />
            <el-option label="已发布" value="published" />
            <el-option label="已下架" value="offline" />
          </el-select>
          <div class="jj-sub field-tip">发布需要课程至少有 1 个章节</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawer = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">
          {{ form.id ? '保存' : '创建并去加章节' }}
        </el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { inject, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const route = useRoute()
const router = useRouter()
const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = canWrite('courses')

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
  majorId: null,
  name: '',
  difficulty: 1,
  hours: 0,
  summary: '',
  status: 'draft'
})

const rules = {
  name: [{ required: true, message: '请输入课程名称', trigger: 'blur' }],
  majorId: [{ required: true, message: '请选择所属专业', trigger: 'change' }]
}

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
    const data = await api.courses.list({
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

function resetForm() {
  Object.assign(form, {
    id: 0,
    majorId: filters.majorId || null,
    name: '',
    difficulty: 1,
    hours: 0,
    summary: '',
    status: 'draft'
  })
  formRef.value?.clearValidate()
}

function openCreate() {
  resetForm()
  drawer.value = true
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const data = await api.courses.save({ ...form })
    if (form.id) {
      ElMessage.success('已保存')
      drawer.value = false
      load()
    } else {
      ElMessage.success('课程已创建，接着添加章节')
      drawer.value = false
      router.push(`/courses/${data.id}`)
    }
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function changeStatus(row, status) {
  try {
    await api.courses.status(row.id, status)
    ElMessage.success(status === 'published' ? '已发布' : '已下架')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除课程「${row.name}」？其下 ${row.chapterCount} 个章节会一并停用。`,
      '删除课程',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.courses.remove(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)

onMounted(() => {
  if (route.query.majorId) filters.majorId = Number(route.query.majorId)
  load()
})
</script>

<style scoped>
.head-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.course-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 2px;
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
