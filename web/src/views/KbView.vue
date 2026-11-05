<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">AI 知识库资料</h2>
        <p class="jj-page-desc">
          资料是 AI 回答的唯一依据。正文解析成纯文本后按约 500 字分块，只有解析成功且已发布的资料才会被检索。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建资料
        </el-button>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Files /></el-icon>资料总数</div>
        <div class="jj-stat-value">{{ totals.total }}<span class="jj-stat-unit">份</span></div>
        <div class="jj-stat-foot">按当前筛选统计</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><CircleCheck /></el-icon>已发布</div>
        <div class="jj-stat-value">{{ statusCounts.published || 0 }}<span class="jj-stat-unit">份</span></div>
        <div class="jj-stat-foot">可被 AI 检索</div>
      </div>
      <div class="jj-stat" :class="parseCounts.failed ? 'bad' : ''">
        <div class="jj-stat-label"><el-icon><Warning /></el-icon>解析失败</div>
        <div class="jj-stat-value">{{ parseCounts.failed || 0 }}<span class="jj-stat-unit">份</span></div>
        <div class="jj-stat-foot">失败原因见列表</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Reading /></el-icon>正文规模</div>
        <div class="jj-stat-value">{{ charText }}<span class="jj-stat-unit">万字</span></div>
        <div class="jj-stat-foot">{{ totals.chunks }} 个分块 · {{ totals.files }} 份带文件</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">资料列表</div>
        <div class="head-tools">
          <el-select v-model="filters.category" placeholder="全部分类" clearable style="width: 140px" @change="reload">
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
          <el-select v-model="filters.parseStatus" placeholder="解析状态" clearable style="width: 130px" @change="reload">
            <el-option label="解析成功" value="success" />
            <el-option label="解析失败" value="failed" />
            <el-option label="待解析" value="pending" />
          </el-select>
          <el-select v-model="filters.status" placeholder="发布状态" clearable style="width: 130px" @change="reload">
            <el-option label="草稿" value="draft" />
            <el-option label="审核中" value="pending" />
            <el-option label="已发布" value="published" />
            <el-option label="已下架" value="offline" />
          </el-select>
          <el-input v-model="filters.keyword" placeholder="搜索标题 / 来源 / 标签" clearable style="width: 200px" @input="reload">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button text :loading="loading" @click="load">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="资料" min-width="260">
          <template #default="{ row }">
            <div class="doc-title">{{ row.title }}</div>
            <div class="jj-sub">
              {{ row.source || '未填写来源' }}
              <template v-if="row.tags"> · {{ row.tags }}</template>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="分类" width="110">
          <template #default="{ row }">
            <span class="chip">{{ row.category || '未分类' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="版权" width="100">
          <template #default="{ row }">
            <span class="jj-sub" :class="{ warnText: row.copyright === 'unknown' }">{{ row.copyrightText }}</span>
          </template>
        </el-table-column>
        <el-table-column label="字数 / 分块" width="120" align="right">
          <template #default="{ row }">
            <span class="num">{{ row.charCount || 0 }}</span>
            <span class="jj-sub"> / {{ row.chunkCount || 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="解析" width="130">
          <template #default="{ row }">
            <el-tooltip v-if="row.parseStatus === 'failed'" :content="row.parseError" placement="top">
              <span class="jj-tag offline">解析失败</span>
            </el-tooltip>
            <span v-else class="jj-tag" :class="row.parseStatus === 'success' ? 'published' : 'draft'">
              {{ row.parseText }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <span class="jj-tag" :class="row.status">{{ statusText(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="150">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">{{ canEdit ? '编辑' : '查看' }}</el-button>
            <template v-if="canEdit">
              <el-button link type="info" @click="parse(row)">解析</el-button>
              <el-button v-if="row.status !== 'published'" link type="success" @click="changeStatus(row, 'published')">发布</el-button>
              <el-button v-else link type="warning" @click="changeStatus(row, 'offline')">下架</el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">还没有资料。AI 回答完全依赖这里的正文，先上传一份操作手册试试。</div>
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

    <el-drawer v-model="drawer" :title="form.id ? '编辑资料' : '新建资料'" size="720px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="资料标题" prop="title">
          <el-input v-model="form.title" placeholder="如：G74 端面切槽循环编程要点" maxlength="128" show-word-limit />
        </el-form-item>
        <div class="row-2">
          <el-form-item label="分类">
            <el-select v-model="form.category" filterable allow-create default-first-option placeholder="选择或输入分类" style="width: 100%">
              <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
            </el-select>
          </el-form-item>
          <el-form-item label="版权归属">
            <el-select v-model="form.copyright" style="width: 100%">
              <el-option label="自有资料" value="own" />
              <el-option label="已获授权" value="authorized" />
              <el-option label="公开资料" value="public" />
              <el-option label="待确认" value="unknown" />
            </el-select>
          </el-form-item>
        </div>
        <div class="row-2">
          <el-form-item label="来源">
            <el-input v-model="form.source" placeholder="如：厂商手册 / 内部讲义" maxlength="64" />
          </el-form-item>
          <el-form-item label="来源链接">
            <el-input v-model="form.sourceUrl" placeholder="选填，原文地址" maxlength="255" />
          </el-form-item>
        </div>
        <el-form-item label="标签">
          <el-input v-model="form.tags" placeholder="多个标签用逗号分隔，如：G74,断屑,走心机" maxlength="200" />
        </el-form-item>

        <el-form-item label="文件">
          <el-upload
            :auto-upload="false"
            :show-file-list="false"
            :on-change="onFileChange"
            :limit="1"
          >
            <el-button><el-icon><Upload /></el-icon>选择本地文件</el-button>
          </el-upload>
          <div class="jj-sub field-tip">
            <template v-if="form.fileName">
              已选择 {{ form.fileName }}（{{ sizeText }}）。当前只记录文件信息，
              文件解析服务接入后会自动读取内容；在此之前请把正文粘贴到下面。
            </template>
            <template v-else>选填。对象存储接入后可直接上传 PDF / Word / 图片。</template>
          </div>
        </el-form-item>

        <el-form-item label="正文">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="10"
            maxlength="500000"
            placeholder="粘贴资料正文，建议一段一个自然段。分块与检索都基于这段文字。"
          />
          <div class="jj-sub field-tip">
            当前 {{ contentChars }} 字，预计 {{ estimateChunks }} 个分块（每块约 500 字）。
            保存时会自动解析一次。
          </div>
        </el-form-item>

        <el-form-item v-if="form.id" label="解析情况">
          <div class="parse-box" :class="form.parseStatus">
            <div class="parse-line">
              <span class="jj-tag" :class="form.parseStatus === 'success' ? 'published' : 'offline'">
                {{ parseStatusText }}
              </span>
              <span class="jj-sub" v-if="form.parsedAt">最近解析 {{ fmt(form.parsedAt) }}</span>
            </div>
            <div v-if="form.parseError" class="parse-err">{{ form.parseError }}</div>
          </div>
        </el-form-item>

        <el-form-item v-if="canEdit" label="发布状态">
          <el-radio-group v-model="form.status">
            <el-radio value="draft">草稿</el-radio>
            <el-radio value="pending">审核中</el-radio>
            <el-radio value="published">已发布</el-radio>
            <el-radio value="offline">已下架</el-radio>
          </el-radio-group>
          <div class="jj-sub field-tip">发布要求：解析成功且版权不是「待确认」。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawer = false">取消</el-button>
        <el-button v-if="canEdit" type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = canWrite('kb')

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const drawer = ref(false)
const formRef = ref(null)
const categories = ref([])

const filters = reactive({ category: '', parseStatus: '', status: '', keyword: '' })
const parseCounts = ref({ pending: 0, parsing: 0, success: 0, failed: 0 })
const statusCounts = ref({ draft: 0, pending: 0, published: 0, offline: 0 })
const totals = ref({ total: 0, chars: 0, chunks: 0, files: 0 })

const form = reactive({
  id: 0,
  title: '',
  category: '',
  source: '',
  sourceUrl: '',
  copyright: 'unknown',
  tags: '',
  fileName: '',
  fileKey: '',
  fileSize: 0,
  mimeType: '',
  content: '',
  status: 'draft',
  parseStatus: '',
  parseError: '',
  parsedAt: null
})

const rules = {
  title: [{ required: true, message: '请输入资料标题', trigger: 'blur' }]
}

const charText = computed(() => ((totals.value.chars || 0) / 10000).toFixed(2).replace(/\.00$/, ''))
const contentChars = computed(() => (form.content || '').length)
const estimateChunks = computed(() => Math.ceil(contentChars.value / 500) || 0)
const sizeText = computed(() => {
  const mb = (form.fileSize || 0) / 1024 / 1024
  return mb >= 1 ? mb.toFixed(1) + ' MB' : Math.max(1, Math.round((form.fileSize || 0) / 1024)) + ' KB'
})
const parseStatusText = computed(
  () => ({ pending: '待解析', parsing: '解析中', success: '解析成功', failed: '解析失败' }[form.parseStatus] || '待解析')
)

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
    const data = await api.kb.list({
      page: page.value,
      size: size.value,
      category: filters.category || undefined,
      parseStatus: filters.parseStatus || undefined,
      status: filters.status || undefined,
      keyword: filters.keyword || undefined
    })
    list.value = data.list || []
    total.value = data.total || 0
    categories.value = data.categories || []
    parseCounts.value = data.parseCounts || parseCounts.value
    statusCounts.value = data.statusCounts || statusCounts.value
    totals.value = { total: data.total || 0, ...(data.totals || {}) }
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
    title: '',
    category: filters.category || '',
    source: '',
    sourceUrl: '',
    copyright: 'unknown',
    tags: '',
    fileName: '',
    fileKey: '',
    fileSize: 0,
    mimeType: '',
    content: '',
    status: 'draft',
    parseStatus: '',
    parseError: '',
    parsedAt: null
  })
}

function openCreate() {
  resetForm()
  formRef.value?.clearValidate()
  drawer.value = true
}

async function openEdit(row) {
  resetForm()
  formRef.value?.clearValidate()
  drawer.value = true
  try {
    const data = await api.kb.detail(row.id)
    Object.assign(form, data)
  } catch (e) {
    ElMessage.error(e.message)
    drawer.value = false
  }
}

function onFileChange(file) {
  form.fileName = file.name
  form.fileSize = file.size
  form.mimeType = file.raw?.type || ''
  form.fileKey = 'local/' + file.name
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const data = await api.kb.save({ ...form })
    ElMessage.success(form.id ? '已保存并重新解析' : '资料已创建')
    if (!form.id) drawer.value = false
    if (data?.id) Object.assign(form, data)
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function parse(row) {
  try {
    const data = await api.kb.parse(row.id)
    if (data.ok) {
      ElMessage.success(`解析成功：${data.charCount} 字，${data.chunkCount} 个分块`)
    } else {
      ElMessage.warning('解析失败：' + (data.parseError || '未知原因'))
    }
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function changeStatus(row, status) {
  try {
    await api.kb.status(row.id, status)
    ElMessage.success(status === 'published' ? '已发布，AI 可检索' : '已下架')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除资料「${row.title}」？删除后它的正文与分块都不再参与检索。`,
      '删除资料',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.kb.remove(row.id)
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

.doc-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 2px;
}

.chip {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 8px;
  font-size: 12px;
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
}

.num {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
}

.warnText {
  color: #d97706;
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

.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}

.field-tip {
  margin-top: 4px;
  line-height: 1.6;
  width: 100%;
}

.parse-box {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.1);
  line-height: 1.7;
}

.parse-box.success {
  background: rgba(22, 163, 74, 0.08);
}

.parse-box.failed {
  background: rgba(220, 38, 38, 0.07);
}

.parse-line {
  display: flex;
  align-items: center;
  gap: 10px;
}

.parse-err {
  margin-top: 6px;
  font-size: 12px;
  color: #dc2626;
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding: 14px 4px 2px;
}
</style>
