<template>
  <div class="jj-page" v-loading="loading">
    <div class="back-row">
      <el-button text @click="$router.push('/paths')">
        <el-icon><ArrowLeft /></el-icon>返回路线列表
      </el-button>
    </div>

    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">
          {{ form.name || '路线编排' }}
          <span class="jj-tag" :class="form.status">{{ statusText(form.status) }}</span>
        </h2>
        <p class="jj-page-desc">
          {{ majorName || '通用路线' }} · {{ difficultyText(form.difficulty) }} ·
          {{ stats.stageCount }} 个阶段 · {{ stats.courseCount }} 门课程 · {{ stats.totalHours }} 学时
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button
          v-if="canEdit && form.status !== 'published'"
          :loading="statusSaving"
          @click="changeStatus('published')"
        >
          发布路线
        </el-button>
        <el-button v-else-if="canEdit" :loading="statusSaving" @click="changeStatus('offline')">
          下架路线
        </el-button>
        <el-button v-if="canEdit" type="primary" :loading="savingPath" @click="savePath">
          <el-icon><Check /></el-icon>保存路线
        </el-button>
      </div>
    </div>

    <div class="edit-grid">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">路线信息</div>
        </div>
        <div class="jj-card-body">
          <el-form ref="pathFormRef" :model="form" :rules="rules" label-width="80px">
            <el-form-item label="路线名称" prop="name">
              <el-input v-model="form.name" maxlength="64" :disabled="!canEdit" />
            </el-form-item>
            <el-form-item label="所属专业">
              <el-select
                v-model="form.majorId"
                placeholder="不限专业（通用路线）"
                clearable
                style="width: 100%"
                :disabled="!canEdit"
              >
                <el-option v-for="m in majors" :key="m.id" :label="m.name" :value="m.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="难度">
              <el-radio-group v-model="form.difficulty" :disabled="!canEdit">
                <el-radio :value="1">入门</el-radio>
                <el-radio :value="2">进阶</el-radio>
                <el-radio :value="3">高级</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="预计学时">
              <el-input-number v-model="form.hours" :min="0" :max="9999" :disabled="!canEdit" />
              <span class="jj-sub unit">小时</span>
            </el-form-item>
            <el-form-item label="路线简介">
              <el-input
                v-model="form.summary"
                type="textarea"
                :rows="5"
                maxlength="200"
                show-word-limit
                :disabled="!canEdit"
                placeholder="学完这条路线能做什么"
              />
            </el-form-item>
            <el-form-item label="发布状态">
              <el-select v-model="form.status" style="width: 100%" :disabled="!canEdit">
                <el-option label="草稿" value="draft" />
                <el-option label="审核中" value="pending" />
                <el-option label="已发布" value="published" />
                <el-option label="已下架" value="offline" />
              </el-select>
            </el-form-item>
          </el-form>
          <div class="meta-tip">
            创建于 {{ fmt(pathInfo.createdAt) }} · 最后修改 {{ fmt(pathInfo.updatedAt) }}
          </div>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">
            阶段与课程
            <span class="jj-card-title-sub">{{ stats.stageCount }} 阶段 / {{ stats.courseCount }} 课程</span>
          </div>
          <el-button v-if="canEdit" size="small" type="primary" @click="openStage(null)">
            <el-icon><Plus /></el-icon>添加阶段
          </el-button>
        </div>

        <div class="stage-wrap">
          <div v-if="!stages.length" class="jj-empty stage-empty">
            <p>还没有阶段。至少配好一个阶段并加入课程，路线才能发布。</p>
            <el-button v-if="canEdit" type="primary" @click="openStage(null)">添加第一个阶段</el-button>
          </div>

          <div v-for="(stage, si) in stages" :key="stage.id" class="stage-block">
            <div class="stage-head">
              <div class="stage-main">
                <span class="stage-index">{{ si + 1 }}</span>
                <div class="stage-text">
                  <div class="stage-title">{{ stage.title }}</div>
                  <div class="jj-sub">
                    {{ stage.summary || '无说明' }} · {{ stage.courseNum }} 门课 / {{ stage.totalHours }} 学时
                  </div>
                </div>
              </div>
              <div class="stage-actions">
                <template v-if="canEdit">
                  <el-button text :disabled="si === 0" @click="moveStage(si, -1)">
                    <el-icon><Top /></el-icon>
                  </el-button>
                  <el-button text :disabled="si === stages.length - 1" @click="moveStage(si, 1)">
                    <el-icon><Bottom /></el-icon>
                  </el-button>
                  <el-button link type="primary" @click="openStage(stage)">编辑</el-button>
                  <el-button link type="danger" @click="removeStage(stage)">删除</el-button>
                </template>
                <span v-else class="jj-sub">只读</span>
              </div>
            </div>

            <div class="item-list">
              <div v-for="(item, ii) in stage.items" :key="item.id" class="item-row">
                <span class="item-order">{{ ii + 1 }}</span>
                <div class="item-main">
                  <div class="item-title">
                    {{ item.name }}
                    <span class="jj-tag sm" :class="item.status">{{ statusText(item.status) }}</span>
                    <span v-if="!item.required" class="opt-flag">选修</span>
                  </div>
                  <div class="jj-sub">
                    {{ item.majorName }} · {{ difficultyText(item.difficulty) }} · {{ item.chapterCount }} 章
                    <template v-if="item.unlockRule"> · 解锁：{{ item.unlockRule }}</template>
                  </div>
                </div>
                <div class="item-actions">
                  <template v-if="canEdit">
                    <el-button text :disabled="ii === 0" @click="moveItem(stage, ii, -1)">
                      <el-icon><Top /></el-icon>
                    </el-button>
                    <el-button
                      text
                      :disabled="ii === stage.items.length - 1"
                      @click="moveItem(stage, ii, 1)"
                    >
                      <el-icon><Bottom /></el-icon>
                    </el-button>
                    <el-button link type="primary" @click="openItem(stage, item)">设置</el-button>
                    <el-button link type="danger" @click="removeItem(stage, item)">移除</el-button>
                  </template>
                </div>
              </div>

              <div v-if="!stage.items.length" class="item-empty jj-sub">该阶段还没有课程</div>

              <div v-if="canEdit" class="item-add">
                <el-button link type="primary" @click="openPicker(stage)">
                  <el-icon><Plus /></el-icon>添加课程
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="stageDialog"
      :title="stageForm.id ? '编辑阶段' : '添加阶段'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="stageFormRef" :model="stageForm" :rules="stageRules" label-width="80px">
        <el-form-item label="阶段名称" prop="title">
          <el-input v-model="stageForm.title" maxlength="64" show-word-limit placeholder="如：第一阶段 识图与设备认知" />
        </el-form-item>
        <el-form-item label="阶段说明">
          <el-input
            v-model="stageForm.summary"
            type="textarea"
            :rows="3"
            maxlength="200"
            show-word-limit
            placeholder="这一阶段的目标是什么"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="stageDialog = false">取消</el-button>
        <el-button type="primary" :loading="stageSaving" @click="submitStage">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="pickerDialog"
      :title="`向「${pickerStage?.title || ''}」添加课程`"
      width="760px"
      :close-on-click-modal="false"
    >
      <div class="picker-tools">
        <el-input v-model="pickerKeyword" placeholder="搜索课程名称" clearable style="width: 220px">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <span class="jj-sub">已选 {{ picked.length }} 门</span>
      </div>
      <el-table
        ref="pickerTableRef"
        :data="pickerFiltered"
        height="360"
        row-key="id"
        @selection-change="onPicked"
      >
        <el-table-column type="selection" width="46" reserve-selection />
        <el-table-column label="课程" min-width="220">
          <template #default="{ row }">
            <div class="pick-name">{{ row.name }}</div>
            <div class="jj-sub">{{ row.majorName }} · {{ row.chapterCount }} 章</div>
          </template>
        </el-table-column>
        <el-table-column label="难度" width="80">
          <template #default="{ row }">
            <span class="jj-sub">{{ difficultyText(row.difficulty) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <span class="jj-tag sm" :class="row.status">{{ statusText(row.status) }}</span>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">没有匹配的课程</div>
        </template>
      </el-table>
      <div class="jj-sub picker-tip">
        已在路线中的课程不会重复添加；草稿课程加进路线后，学员端仍看不到，直到课程本身发布。
      </div>
      <template #footer>
        <el-button @click="pickerDialog = false">取消</el-button>
        <el-button type="primary" :loading="picking" @click="submitPick">
          添加{{ picked.length ? ` ${picked.length} 门` : '' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="itemDialog" title="课程设置" width="520px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="课程">{{ itemForm.name }}</el-form-item>
        <el-form-item label="必学">
          <el-switch v-model="itemForm.required" />
          <span class="jj-sub unit">关闭后标记为选修，不影响路线完成判定</span>
        </el-form-item>
        <el-form-item label="解锁条件">
          <el-input
            v-model="itemForm.unlockRule"
            maxlength="120"
            show-word-limit
            placeholder="如：完成上一阶段全部课程后解锁"
          />
          <div class="jj-sub field-tip">
            当前仅作为提示文案展示，不做自动解锁判断，等小程序端需要时再启用规则
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemDialog = false">取消</el-button>
        <el-button type="primary" :loading="itemSaving" @click="submitItem">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const route = useRoute()
const router = useRouter()
const canEdit = canWrite('paths')
const pathId = Number(route.params.id)

const loading = ref(false)
const savingPath = ref(false)
const statusSaving = ref(false)
const stageSaving = ref(false)
const picking = ref(false)
const itemSaving = ref(false)

const pathInfo = ref({})
const stages = ref([])
const majors = ref([])
const stats = ref({ stageCount: 0, courseCount: 0, totalHours: 0 })

const stageDialog = ref(false)
const pickerDialog = ref(false)
const itemDialog = ref(false)
const pickerStage = ref(null)
const pathFormRef = ref(null)
const stageFormRef = ref(null)
const pickerTableRef = ref(null)

const pickList = ref([])
const pickerKeyword = ref('')
const picked = ref([])

const form = reactive({
  id: pathId,
  name: '',
  majorId: null,
  difficulty: 1,
  hours: 0,
  summary: '',
  status: 'draft'
})

const stageForm = reactive({ id: 0, title: '', summary: '' })
const itemForm = reactive({ id: 0, name: '', required: true, unlockRule: '' })

const rules = {
  name: [{ required: true, message: '请输入路线名称', trigger: 'blur' }]
}

const stageRules = {
  title: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }]
}

const majorName = computed(() => {
  if (!form.majorId) return ''
  return majors.value.find((m) => m.id === form.majorId)?.name || ''
})

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

async function loadDetail() {
  loading.value = true
  try {
    const data = await api.paths.detail(pathId)
    pathInfo.value = data.path || {}
    stages.value = data.stages || []
    stats.value = data.stats || stats.value
    majors.value = data.majors || []
    Object.assign(form, {
      id: pathId,
      name: pathInfo.value.name,
      majorId: pathInfo.value.majorId || null,
      difficulty: pathInfo.value.difficulty,
      hours: pathInfo.value.hours,
      summary: pathInfo.value.summary,
      status: pathInfo.value.status
    })
  } catch (e) {
    ElMessage.error(e.message)
    router.push('/paths')
  } finally {
    loading.value = false
  }
}

async function savePath() {
  const valid = await pathFormRef.value?.validate().catch(() => false)
  if (!valid) return
  savingPath.value = true
  try {
    await api.paths.save({ ...form, majorId: form.majorId || 0 })
    ElMessage.success('已保存')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    savingPath.value = false
  }
}

async function changeStatus(status) {
  statusSaving.value = true
  try {
    await api.paths.status(pathId, status)
    ElMessage.success(status === 'published' ? '已发布' : '已下架')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    statusSaving.value = false
  }
}

function openStage(stage) {
  Object.assign(stageForm, {
    id: stage?.id || 0,
    title: stage?.title || '',
    summary: stage?.summary || ''
  })
  stageFormRef.value?.clearValidate()
  stageDialog.value = true
}

async function submitStage() {
  const valid = await stageFormRef.value?.validate().catch(() => false)
  if (!valid) return
  stageSaving.value = true
  try {
    await api.paths.stageSave({ ...stageForm, pathId })
    ElMessage.success(stageForm.id ? '已保存' : '已添加')
    stageDialog.value = false
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    stageSaving.value = false
  }
}

async function removeStage(stage) {
  const tip = stage.items.length
    ? `「${stage.title}」下有 ${stage.items.length} 门课程，删除阶段会一并从路线中移除，课程本身不受影响。确认？`
    : `确认删除阶段「${stage.title}」？`
  try {
    await ElMessageBox.confirm(tip, '删除阶段', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.paths.stageRemove(stage.id)
    ElMessage.success('已删除')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function moveStage(index, delta) {
  const target = index + delta
  if (target < 0 || target >= stages.value.length) return
  const arr = [...stages.value]
  ;[arr[index], arr[target]] = [arr[target], arr[index]]
  stages.value = arr
  try {
    await api.paths.stageSort(arr.map((i) => i.id))
  } catch (e) {
    ElMessage.error(e.message)
    loadDetail()
  }
}

async function moveItem(stage, index, delta) {
  const target = index + delta
  if (target < 0 || target >= stage.items.length) return
  const arr = [...stage.items]
  ;[arr[index], arr[target]] = [arr[target], arr[index]]
  stage.items = arr
  try {
    await api.paths.sortItems(arr.map((i) => i.id))
  } catch (e) {
    ElMessage.error(e.message)
    loadDetail()
  }
}

function openItem(stage, item) {
  Object.assign(itemForm, {
    id: item.id,
    name: item.name,
    required: item.required,
    unlockRule: item.unlockRule || ''
  })
  itemDialog.value = true
}

async function submitItem() {
  itemSaving.value = true
  try {
    await api.paths.updateItem({
      id: itemForm.id,
      required: itemForm.required,
      unlockRule: itemForm.unlockRule
    })
    ElMessage.success('已保存')
    itemDialog.value = false
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    itemSaving.value = false
  }
}

async function removeItem(stage, item) {
  try {
    await ElMessageBox.confirm(`从路线中移除「${item.name}」？课程本身不会被删除。`, '移除课程', {
      confirmButtonText: '移除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.paths.removeItem(item.id)
    ElMessage.success('已移除')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function openPicker(stage) {
  pickerStage.value = stage
  pickerKeyword.value = ''
  picked.value = []
  pickerDialog.value = true
  try {
    const data = await api.courses.list({ page: 1, size: 200 })
    const existed = new Set()
    stages.value.forEach((s) => s.items.forEach((i) => existed.add(i.courseId)))
    pickList.value = (data.list || []).filter((c) => !existed.has(c.id))
  } catch (e) {
    ElMessage.error(e.message)
  }
}

const pickerFiltered = computed(() => {
  const kw = pickerKeyword.value.trim().toLowerCase()
  if (!kw) return pickList.value
  return pickList.value.filter(
    (c) => c.name.toLowerCase().includes(kw) || (c.majorName || '').toLowerCase().includes(kw)
  )
})

function onPicked(rows) {
  picked.value = rows
}

async function submitPick() {
  if (!picked.value.length) {
    ElMessage.warning('请先选择要添加的课程')
    return
  }
  picking.value = true
  try {
    const res = await api.paths.addCourses(
      pathId,
      pickerStage.value.id,
      picked.value.map((c) => c.id)
    )
    if (res.skipped?.length) {
      ElMessage.warning(`已添加 ${res.added} 门，跳过：${res.skipped.join('、')}`)
    } else {
      ElMessage.success(`已添加 ${res.added} 门课程`)
    }
    pickerDialog.value = false
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    picking.value = false
  }
}

onMounted(() => {
  if (!pathId) {
    ElMessage.error('缺少路线 id')
    router.push('/paths')
    return
  }
  loadDetail()
})
</script>

<style scoped>
.back-row {
  margin-bottom: 6px;
}

.jj-page-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.jj-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
  vertical-align: middle;
}

.jj-tag.sm {
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 500;
  vertical-align: middle;
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

.edit-grid {
  display: grid;
  grid-template-columns: 380px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

@media (max-width: 1180px) {
  .edit-grid {
    grid-template-columns: 1fr;
  }
}

.meta-tip {
  margin-top: 6px;
  padding-top: 12px;
  border-top: 1px dashed rgba(148, 163, 184, 0.3);
  font-size: 12px;
  color: #94a3b8;
}

.stage-wrap {
  padding: 4px 0 0;
}

.stage-empty {
  padding: 48px 20px;
  text-align: center;
}

.stage-empty p {
  margin-bottom: 14px;
  color: #64748b;
}

.stage-block {
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 12px;
  margin-bottom: 14px;
  overflow: hidden;
}

.stage-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: rgba(248, 250, 252, 0.75);
}

.stage-main {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.stage-index {
  width: 24px;
  height: 24px;
  border-radius: 7px;
  background: var(--jj-orange);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 1px;
}

.stage-text {
  min-width: 0;
}

.stage-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
}

.stage-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.item-list {
  padding: 4px 12px 10px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 6px;
  border-bottom: 1px dashed rgba(148, 163, 184, 0.22);
}

.item-row:last-of-type {
  border-bottom: none;
}

.item-order {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.item-main {
  min-width: 0;
  flex: 1;
}

.item-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 2px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.opt-flag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
  font-weight: 500;
}

.item-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.item-empty {
  padding: 10px 6px;
  font-size: 12px;
}

.item-add {
  padding: 8px 6px 2px;
}

.picker-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.pick-name {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 2px;
}

.picker-tip {
  margin-top: 10px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.6;
}

.unit {
  margin-left: 8px;
}

.field-tip {
  margin-top: 4px;
}
</style>
