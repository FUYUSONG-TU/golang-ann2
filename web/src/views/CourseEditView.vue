<template>
  <div class="jj-page" v-loading="loading">
    <div class="back-row">
      <el-button text @click="$router.push('/courses')">
        <el-icon><ArrowLeft /></el-icon>返回课程列表
      </el-button>
    </div>

    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">
          {{ form.name || '课程编辑' }}
          <span class="jj-tag" :class="form.status">{{ statusText(form.status) }}</span>
        </h2>
        <p class="jj-page-desc">
          {{ majorName || '未分配专业' }} · {{ difficultyText(form.difficulty) }} ·
          {{ chapterTotal }} 个章节
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button
          v-if="canEdit && form.status !== 'published'"
          :loading="statusSaving"
          @click="changeStatus('published')"
        >
          发布课程
        </el-button>
        <el-button
          v-else-if="canEdit"
          :loading="statusSaving"
          @click="changeStatus('offline')"
        >
          下架课程
        </el-button>
        <el-button v-if="canEdit" type="primary" :loading="savingCourse" @click="saveCourse">
          <el-icon><Check /></el-icon>保存课程
        </el-button>
      </div>
    </div>

    <div class="edit-grid">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">课程信息</div>
        </div>
        <div class="jj-card-body">
          <el-form ref="courseFormRef" :model="form" :rules="rules" label-width="80px">
            <el-form-item label="课程名称" prop="name">
              <el-input v-model="form.name" maxlength="64" :disabled="!canEdit" />
            </el-form-item>
            <el-form-item label="所属专业" prop="majorId">
              <el-select v-model="form.majorId" style="width: 100%" :disabled="!canEdit">
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
            <el-form-item label="课时">
              <el-input-number v-model="form.hours" :min="0" :max="999" :disabled="!canEdit" />
              <span class="jj-sub unit">小时</span>
            </el-form-item>
            <el-form-item label="课程简介">
              <el-input
                v-model="form.summary"
                type="textarea"
                :rows="5"
                maxlength="500"
                show-word-limit
                :disabled="!canEdit"
                placeholder="这门课讲什么、学完能做什么"
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
            创建于 {{ fmt(course.createdAt) }} · 最后修改 {{ fmt(course.updatedAt) }}
          </div>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">
            章节结构
            <span class="jj-card-title-sub">{{ chapterTotal }} 项</span>
          </div>
          <el-button v-if="canEdit" size="small" type="primary" @click="openChapter(null, 0)">
            <el-icon><Plus /></el-icon>添加章
          </el-button>
        </div>

        <div class="chapter-wrap">
          <div v-if="!chapters.length" class="jj-empty chapter-empty">
            <p>还没有章节。课程至少需要一个章节才能发布。</p>
            <el-button v-if="canEdit" type="primary" @click="openChapter(null, 0)">
              添加第一个章
            </el-button>
          </div>

          <div v-for="(node, index) in chapters" :key="node.id" class="chapter-block">
            <div class="chapter-row root">
              <div class="chapter-main">
                <span class="chapter-index">{{ index + 1 }}</span>
                <div>
                  <div class="chapter-title">{{ node.title }}</div>
                  <div class="jj-sub">
                    {{ node.summary || '无简介' }} · {{ node.duration || 0 }} 分钟
                  </div>
                </div>
              </div>
              <div class="chapter-actions">
                <template v-if="canEdit">
                  <el-button text :disabled="index === 0" @click="moveNode(chapters, index, -1, 0)">
                    <el-icon><Top /></el-icon>
                  </el-button>
                  <el-button
                    text
                    :disabled="index === chapters.length - 1"
                    @click="moveNode(chapters, index, 1, 0)"
                  >
                    <el-icon><Bottom /></el-icon>
                  </el-button>
                  <el-button link type="primary" @click="openChapter(null, node.id)">
                    加知识点
                  </el-button>
                  <el-button link type="primary" @click="openChapter(node, 0)">编辑</el-button>
                  <el-button link type="danger" @click="removeChapter(node, true)">删除</el-button>
                </template>
                <span v-else class="jj-sub">只读</span>
              </div>
            </div>

            <div v-if="node.children?.length" class="children">
              <div v-for="(child, ci) in node.children" :key="child.id" class="chapter-row child">
                <div class="chapter-main">
                  <span class="dot"></span>
                  <div>
                    <div class="chapter-title small">{{ child.title }}</div>
                    <div class="jj-sub">{{ child.duration || 0 }} 分钟</div>
                  </div>
                </div>
                <div class="chapter-actions">
                  <template v-if="canEdit">
                    <el-button
                      text
                      :disabled="ci === 0"
                      @click="moveNode(node.children, ci, -1, node.id)"
                    >
                      <el-icon><Top /></el-icon>
                    </el-button>
                    <el-button
                      text
                      :disabled="ci === node.children.length - 1"
                      @click="moveNode(node.children, ci, 1, node.id)"
                    >
                      <el-icon><Bottom /></el-icon>
                    </el-button>
                    <el-button link type="primary" @click="openChapter(child, 0)">编辑</el-button>
                    <el-button link type="danger" @click="removeChapter(child, false)">删除</el-button>
                  </template>
                </div>
              </div>
            </div>

            <div v-else class="children-empty jj-sub">暂无知识点</div>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="chapterDialog"
      :title="chapterForm.id ? '编辑章节' : chapterForm.parentId ? '添加知识点' : '添加章'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="chapterFormRef" :model="chapterForm" :rules="chapterRules" label-width="80px">
        <el-form-item label="标题" prop="title">
          <el-input
            v-model="chapterForm.title"
            maxlength="64"
            show-word-limit
            :placeholder="chapterForm.parentId ? '如：1.1 啄钻参数含义' : '如：第一章 循环指令基础'"
          />
        </el-form-item>
        <el-form-item label="简介">
          <el-input
            v-model="chapterForm.summary"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-word-limit
            placeholder="这一节讲什么"
          />
        </el-form-item>
        <el-form-item label="时长">
          <el-input-number v-model="chapterForm.duration" :min="0" :max="9999" />
          <span class="jj-sub unit">分钟</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="chapterForm.status" style="width: 100%">
            <el-option label="草稿" value="draft" />
            <el-option label="已发布" value="published" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="chapterDialog = false">取消</el-button>
        <el-button type="primary" :loading="chapterSaving" @click="submitChapter">保存</el-button>
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
const canEdit = canWrite('courses')

const courseId = Number(route.params.id)
const loading = ref(false)
const savingCourse = ref(false)
const statusSaving = ref(false)
const chapterSaving = ref(false)

const course = ref({})
const majors = ref([])
const chapters = ref([])
const chapterDialog = ref(false)
const courseFormRef = ref(null)
const chapterFormRef = ref(null)

const form = reactive({
  id: courseId,
  majorId: null,
  name: '',
  difficulty: 1,
  hours: 0,
  summary: '',
  status: 'draft'
})

const chapterForm = reactive({
  id: 0,
  parentId: 0,
  title: '',
  summary: '',
  duration: 0,
  status: 'published'
})

const rules = {
  name: [{ required: true, message: '请输入课程名称', trigger: 'blur' }],
  majorId: [{ required: true, message: '请选择所属专业', trigger: 'change' }]
}

const chapterRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }]
}

const majorName = computed(() => majors.value.find((m) => m.id === form.majorId)?.name || '')
const chapterTotal = computed(() =>
  chapters.value.reduce((sum, node) => sum + 1 + (node.children?.length || 0), 0)
)

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
    const data = await api.courses.detail(courseId)
    course.value = data.course || {}
    chapters.value = data.chapters || []
    majors.value = data.majors || []
    Object.assign(form, {
      id: courseId,
      majorId: course.value.majorId,
      name: course.value.name,
      difficulty: course.value.difficulty,
      hours: course.value.hours,
      summary: course.value.summary,
      status: course.value.status
    })
  } catch (e) {
    ElMessage.error(e.message)
    router.push('/courses')
  } finally {
    loading.value = false
  }
}

async function saveCourse() {
  const valid = await courseFormRef.value?.validate().catch(() => false)
  if (!valid) return
  savingCourse.value = true
  try {
    await api.courses.save({ ...form })
    ElMessage.success('已保存')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    savingCourse.value = false
  }
}

async function changeStatus(status) {
  statusSaving.value = true
  try {
    await api.courses.status(courseId, status)
    ElMessage.success(status === 'published' ? '已发布' : '已下架')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    statusSaving.value = false
  }
}

function openChapter(node, parentId) {
  Object.assign(chapterForm, {
    id: node?.id || 0,
    parentId: node ? node.parentId : parentId,
    title: node?.title || '',
    summary: node?.summary || '',
    duration: node?.duration || 0,
    status: node?.status || 'published'
  })
  chapterFormRef.value?.clearValidate()
  chapterDialog.value = true
}

async function submitChapter() {
  const valid = await chapterFormRef.value?.validate().catch(() => false)
  if (!valid) return
  chapterSaving.value = true
  try {
    await api.chapters.save({ ...chapterForm, courseId })
    ElMessage.success(chapterForm.id ? '已保存' : '已添加')
    chapterDialog.value = false
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    chapterSaving.value = false
  }
}

async function removeChapter(node, isRoot) {
  const tip = isRoot && node.children?.length
    ? `「${node.title}」下还有 ${node.children.length} 个知识点，删除会一并移除。确认？`
    : `确认删除「${node.title}」？`
  try {
    await ElMessageBox.confirm(tip, '删除章节', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.chapters.remove(node.id)
    ElMessage.success('已删除')
    loadDetail()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function moveNode(items, index, delta, parentId) {
  const target = index + delta
  if (target < 0 || target >= items.length) return
  const arr = [...items]
  ;[arr[index], arr[target]] = [arr[target], arr[index]]

  if (parentId === 0) {
    chapters.value = arr
  } else {
    const parent = chapters.value.find((c) => c.id === parentId)
    if (parent) parent.children = arr
  }

  try {
    await api.chapters.sort(arr.map((i) => i.id))
  } catch (e) {
    ElMessage.error(e.message)
    loadDetail()
  }
}

onMounted(() => {
  if (!courseId) {
    ElMessage.error('缺少课程 id')
    router.push('/courses')
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
  grid-template-columns: 400px minmax(0, 1fr);
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

.chapter-wrap {
  padding: 6px 0 0;
}

.chapter-empty {
  padding: 48px 20px;
  text-align: center;
}

.chapter-empty p {
  margin-bottom: 14px;
  color: #64748b;
}

.chapter-block {
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
}

.chapter-block:last-child {
  border-bottom: none;
}

.chapter-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 4px;
}

.chapter-row.root {
  background: rgba(248, 250, 252, 0.6);
  border-radius: 10px;
  padding: 12px 12px;
  margin-top: 8px;
}

.chapter-row.child {
  padding: 9px 12px 9px 34px;
}

.chapter-main {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.chapter-index {
  width: 22px;
  height: 22px;
  border-radius: 6px;
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

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
  margin-top: 7px;
}

.chapter-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
}

.chapter-title.small {
  font-size: 13px;
  font-weight: 500;
}

.chapter-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.children {
  padding-bottom: 6px;
}

.children-empty {
  padding: 6px 0 14px 34px;
  font-size: 12px;
}

.unit {
  margin-left: 8px;
}
</style>
