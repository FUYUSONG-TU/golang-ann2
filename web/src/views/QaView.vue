<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">问答沉淀</h2>
        <p class="jj-page-desc">
          学员每问一次，这里就多一条资产。待审核是模型刚答出来的，定稿后同样的提问直接命中，
          不再调用模型——既省钱又快，答案也稳定。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button @click="openCreate">
          <el-icon><Plus /></el-icon>手工录入
        </el-button>
        <el-button
          :type="autoPassCount ? 'primary' : 'default'"
          :disabled="!canManage || !autoReviewOn || !statusCounts.candidate"
          :loading="autoReviewing"
          @click="runAutoReview"
        >
          <el-icon><MagicStick /></el-icon>
          {{ autoReviewOn ? '自动审核 ' + (statusCounts.candidate || 0) + ' 条待审' : '自动审核已关闭' }}
        </el-button>
        <el-button
          :disabled="!canManage || !suggestCount"
          @click="bulkPublish"
        >
          <el-icon><CircleCheck /></el-icon>
          {{ suggestCount ? '一键定稿 ' + suggestCount + ' 条' : '暂无可定稿' }}
        </el-button>
      </div>
    </div>

    <div class="jj-stats qa-stats">
      <div class="jj-stat warn">
        <div class="jj-stat-label">待审核</div>
        <div class="jj-stat-value">{{ statusCounts.candidate || 0 }}</div>
        <div class="jj-stat-foot">
          {{ autoPassCount ? autoPassCount + ' 条自动审核可达标' : '等待新提问沉淀' }}
        </div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label">已定稿</div>
        <div class="jj-stat-value">{{ statusCounts.published || 0 }}</div>
        <div class="jj-stat-foot">可直接命中，不再调模型</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label">直接命中</div>
        <div class="jj-stat-value">{{ totals.hits || 0 }}</div>
        <div class="jj-stat-foot">约省下 {{ totals.hits || 0 }} 次模型调用</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label">累计提问</div>
        <div class="jj-stat-value">{{ totals.asks || 0 }}</div>
        <div class="jj-stat-foot">含归并到同一条的多次提问</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-toolbar qa-toolbar">
        <div class="seg">
          <button
            v-for="tab in statusTabs"
            :key="tab.value"
            class="seg-item"
            :class="{ active: filter.status === tab.value }"
            @click="switchStatus(tab.value)"
          >
            {{ tab.label }}
            <span v-if="tab.value && statusCounts[tab.value]" class="seg-count">{{ statusCounts[tab.value] }}</span>
          </button>
        </div>
        <span class="grow" />
        <el-input
          v-model="filter.keyword"
          class="tool-input"
          clearable
          placeholder="搜索问题或答案"
          :prefix-icon="Search"
          @keyup.enter="reload"
          @clear="reload"
        />
        <el-select v-model="filter.category" class="tool-select" clearable placeholder="全部分类" @change="reload">
          <el-option v-for="item in categories" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="filter.sort" class="tool-select" @change="reload">
          <el-option label="按热度" value="heat" />
          <el-option label="按最新" value="newest" />
          <el-option label="按质量分" value="quality" />
        </el-select>
      </div>

      <div v-loading="loading" class="qa-list">
        <div v-if="!loading && !list.length" class="jj-empty qa-empty">
          <div class="empty-title">还没有问答记录</div>
          <div class="jj-sub">
            学员在 AI 助手里提问后，带出处的答案会自动落到这里等你确认；也可以先手工录几条高频问题打底。
          </div>
        </div>

        <div v-for="item in list" :key="item.id" class="qa-item">
          <div class="qa-top">
            <span class="tag" :class="'st-' + item.status">{{ item.statusText }}</span>
            <span
              v-if="item.status === 'candidate' && item.reviewPreview"
              class="tag"
              :class="'rv-' + item.reviewPreview.verdict"
              :title="item.reviewPreview.note"
            >
              规则评分 {{ item.reviewPreview.score }} · {{ verdictLabel(item.reviewPreview.verdict) }}
            </span>
            <span v-if="item.status === 'published' && item.reviewedBy === 'auto'" class="tag tag-auto" title="按知识规则自动定稿，未经过人工">
              规则定稿
            </span>
            <span v-if="item.suggestPublish" class="tag tag-suggest">建议定稿</span>
            <span class="tag tag-ghost">{{ item.sourceText }}</span>
            <span v-if="item.category" class="tag tag-ghost">{{ item.category }}</span>
            <span class="grow" />
            <span class="heat" :title="'热度 = 被问 + 直接命中×2 + 有用×3 - 没用×4'">
              热度 {{ item.heat }}
            </span>
          </div>

          <div class="qa-question">
            <span class="q-mark">问</span>
            <span class="q-text">{{ item.question }}</span>
          </div>

          <div class="qa-answer" :class="{ open: isOpen(item.id) }" @click="toggle(item.id)">
            <span class="q-mark a-mark">答</span>
            <span class="a-text">{{ item.answer }}</span>
          </div>

          <div v-if="item.aliases" class="qa-aliases">
            <span class="alias-label">相似问法</span>
            <span v-for="(alias, ai) in aliasList(item.aliases)" :key="ai" class="alias">{{ alias }}</span>
          </div>

          <div v-if="item.refList && item.refList.length" class="qa-refs">
            <span class="alias-label">出处</span>
            <span v-for="(ref, ri) in item.refList" :key="ri" class="ref">
              《{{ ref.docTitle }}》第 {{ ref.seq }} 段
            </span>
          </div>

          <div v-else-if="item.status === 'candidate'" class="qa-refs warn">
            <span class="alias-label">出处</span>
            <span class="ref-none">这条没有引用来源，定稿前请先核对，模型可能是凭常识答的</span>
          </div>

          <div v-if="item.reviewNote && item.status !== 'candidate'" class="qa-review" :class="item.reviewVerdict">
            <span class="alias-label">{{ item.reviewedBy === 'auto' ? '规则审核' : '人工审核' }}</span>
            <span class="review-text">{{ item.reviewNote }}</span>
          </div>

          <div class="qa-bottom">
            <div class="metrics">
              <span class="metric"><b>{{ item.askCount + item.hitCount }}</b> 次提问</span>
              <span class="metric"><b>{{ item.hitCount }}</b> 次直出</span>
              <span class="metric up"><b>{{ item.upCount }}</b> 有用</span>
              <span class="metric down"><b>{{ item.downCount }}</b> 没用</span>
              <span class="metric mute">{{ fmtTime(item.createdAt) }}</span>
            </div>
            <div class="ops">
              <el-button v-if="item.status !== 'published'" size="small" type="primary" plain @click="setStatus(item, 'published')">
                定稿
              </el-button>
              <el-button v-else size="small" plain @click="setStatus(item, 'offline')">下架</el-button>
              <el-button size="small" text @click="openEdit(item)">编辑</el-button>
              <el-dropdown trigger="click" @command="(cmd) => onCommand(cmd, item)">
                <el-button size="small" text>
                  更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="merge" :disabled="item.status === 'merged'">合并到其他问题</el-dropdown-item>
                    <el-dropdown-item v-if="item.status !== 'rejected'" command="rejected">驳回</el-dropdown-item>
                    <el-dropdown-item v-if="item.status === 'offline'" command="candidate">退回待审核</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>
      </div>

      <div v-if="total > size" class="jj-pager">
        <el-pagination
          v-model:current-page="page"
          :page-size="size"
          :total="total"
          layout="total, prev, pager, next"
          background
          @current-change="load"
        />
      </div>
    </div>

    <el-drawer v-model="drawer.open" :title="drawer.form.id ? '编辑问答对' : '手工录入问答对'" size="520px">
      <div class="drawer-body">
        <div class="field">
          <label>问题<span class="req">*</span></label>
          <el-input v-model="drawer.form.question" type="textarea" :rows="2" maxlength="200" show-word-limit
            placeholder="用学员的真实问法，越接近口语召回越好" />
        </div>
        <div class="field">
          <label>答案<span class="req">*</span></label>
          <el-input v-model="drawer.form.answer" type="textarea" :rows="10" placeholder="定稿后学员会直接看到这段内容" />
        </div>
        <div class="field">
          <label>相似问法</label>
          <el-input v-model="drawer.form.aliases" type="textarea" :rows="4"
            placeholder="每行一条。同一个问题的不同说法，能显著提高命中率" />
          <div class="field-hint">学员提出相似问题时会自动并进这里，也可以手动补充。</div>
        </div>
        <div class="field">
          <label>分类</label>
          <el-input v-model="drawer.form.category" maxlength="32" placeholder="如 编程指令 / 设备维护" />
        </div>
        <div class="field">
          <label>备注</label>
          <el-input v-model="drawer.form.remark" maxlength="200" placeholder="内部说明，学员看不到" />
        </div>
      </div>
      <template #footer>
        <el-button @click="drawer.open = false">取消</el-button>
        <el-button type="primary" :loading="drawer.saving" @click="saveQa">保存</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="mergeDlg.open" title="合并重复问题" width="560px">
      <div class="merge-tip">
        把这条并入另一条问答对：相似问法、热度、反馈都会累加过去，源条目保留为「已合并」便于追溯。
      </div>
      <div class="merge-source">
        <span class="q-mark">源</span>
        <span>{{ mergeDlg.source?.question }}</span>
      </div>
      <el-input
        v-model="mergeDlg.keyword"
        clearable
        placeholder="搜索目标问答对"
        :prefix-icon="Search"
        @input="searchTargets"
      />
      <div class="merge-list">
        <div
          v-for="opt in mergeDlg.options"
          :key="opt.id"
          class="merge-opt"
          :class="{ active: mergeDlg.targetId === opt.id }"
          @click="mergeDlg.targetId = opt.id"
        >
          <div class="merge-q">{{ opt.question }}</div>
          <div class="merge-meta">{{ opt.statusText }} · 热度 {{ opt.heat }} · 命中 {{ opt.hitCount }}</div>
        </div>
        <div v-if="!mergeDlg.options.length" class="jj-empty merge-empty">
          {{ mergeDlg.searching ? '搜索中…' : '没有找到可合并的问答对' }}
        </div>
      </div>
      <template #footer>
        <el-button @click="mergeDlg.open = false">取消</el-button>
        <el-button type="primary" :disabled="!mergeDlg.targetId" :loading="mergeDlg.saving" @click="doMerge">确认合并</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheck, MagicStick, Plus, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canManage = canWrite('qa')

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const size = 10
const statusCounts = ref({})
const totals = ref({})
const suggestCount = ref(0)
const autoReviewOn = ref(true)
const autoPassCount = ref(0)
const autoReviewing = ref(false)
const categories = ref([])
const expanded = reactive({})

const filter = reactive({ keyword: '', status: '', category: '', sort: 'heat' })

const statusTabs = [
  { value: '', label: '全部' },
  { value: 'candidate', label: '待审核' },
  { value: 'published', label: '已定稿' },
  { value: 'offline', label: '已下架' },
  { value: 'rejected', label: '已驳回' }
]

const drawer = reactive({
  open: false,
  saving: false,
  form: { id: 0, question: '', answer: '', aliases: '', category: '', remark: '' }
})

const mergeDlg = reactive({
  open: false,
  saving: false,
  searching: false,
  source: null,
  keyword: '',
  options: [],
  targetId: 0
})

function aliasList(raw) {
  return (raw || '').split('\n').map((s) => s.trim()).filter(Boolean)
}

function isOpen(id) {
  return !!expanded[id]
}

function toggle(id) {
  expanded[id] = !expanded[id]
}

function fmtTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getMonth() + 1}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.qa.list({ ...filter, page: page.value, size })
    list.value = data.list || []
    total.value = data.total || 0
    statusCounts.value = data.statusCounts || {}
    totals.value = data.totals || {}
    suggestCount.value = data.suggestCount || 0
    autoReviewOn.value = data.autoReview !== false
    autoPassCount.value = data.autoPassCount || 0
    categories.value = data.categories || []
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

function switchStatus(value) {
  filter.status = value
  reload()
}

function openCreate() {
  drawer.form = { id: 0, question: '', answer: '', aliases: '', category: '', remark: '' }
  drawer.open = true
}

function openEdit(item) {
  drawer.form = {
    id: item.id,
    question: item.question,
    answer: item.answer,
    aliases: item.aliases || '',
    category: item.category || '',
    remark: item.remark || ''
  }
  drawer.open = true
}

async function saveQa() {
  if (!drawer.form.question.trim()) {
    ElMessage.warning('请填写问题')
    return
  }
  if (!drawer.form.answer.trim()) {
    ElMessage.warning('请填写答案')
    return
  }
  drawer.saving = true
  try {
    await api.qa.save(drawer.form)
    ElMessage.success('已保存')
    drawer.open = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    drawer.saving = false
  }
}

async function setStatus(item, status) {
  const tips = {
    published: '定稿后，学员问到相似问题会直接拿到这条答案，不再调用模型。',
    offline: '下架后不再参与命中，已有记录保留。',
    rejected: '驳回后不再参与匹配，也不会再被归并。',
    candidate: '退回待审核，重新进入候选池。'
  }
  try {
    await ElMessageBox.confirm(tips[status] || '确认变更状态？', '变更状态', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      type: status === 'published' ? 'success' : 'warning'
    })
  } catch {
    return
  }
  try {
    await api.qa.status(item.id, status)
    ElMessage.success('已更新')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function onCommand(cmd, item) {
  if (cmd === 'merge') {
    openMerge(item)
    return
  }
  if (cmd === 'delete') {
    try {
      await ElMessageBox.confirm('删除后这条问答对不再参与任何匹配，确认删除？', '删除问答对', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }
    try {
      await api.qa.remove(item.id)
      ElMessage.success('已删除')
      load()
    } catch (e) {
      ElMessage.error(e.message)
    }
    return
  }
  setStatus(item, cmd)
}

function openMerge(item) {
  mergeDlg.open = true
  mergeDlg.source = item
  mergeDlg.keyword = ''
  mergeDlg.targetId = 0
  mergeDlg.options = []
  searchTargets()
}

let searchTimer = null
function searchTargets() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    mergeDlg.searching = true
    try {
      const data = await api.qa.list({ keyword: mergeDlg.keyword, page: 1, size: 8 })
      mergeDlg.options = (data.list || []).filter(
        (opt) => opt.id !== mergeDlg.source?.id && opt.status !== 'merged'
      )
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      mergeDlg.searching = false
    }
  }, 260)
}

async function doMerge() {
  mergeDlg.saving = true
  try {
    await api.qa.merge(mergeDlg.source.id, mergeDlg.targetId)
    ElMessage.success('已合并')
    mergeDlg.open = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    mergeDlg.saving = false
  }
}

async function bulkPublish() {
  try {
    await ElMessageBox.confirm(
      `将把 ${suggestCount.value} 条「有出处、有真实需求、没有差评」的候选直接定稿。定稿前建议先扫一眼答案。`,
      '一键定稿',
      { confirmButtonText: '确认定稿', cancelButtonText: '取消', type: 'success' }
    )
  } catch {
    return
  }
  try {
    const data = await api.qa.bulkPublish()
    ElMessage.success(`已定稿 ${data.published} 条`)
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

const verdictLabel = (v) => ({ pass: '可自动定稿', hold: '需人工', reject: '会驳回' })[v] || '待判定'

// 自动审核：把整批待审核交给规则层判一次。
// 前端只负责展示结果，判什么、怎么判全在服务端——页面上的分数和结论是同一套规则算出来的。
async function runAutoReview() {
  const pending = statusCounts.value.candidate || 0
  try {
    await ElMessageBox.confirm(
      `将按「知识规则」审核 ${pending} 条待审问答：分数达标的自动定稿，明显不合格的自动驳回，拿不准的继续留在这里。` +
        `\n\n所有结论都带评分依据，可在每条记录上看到。`,
      '自动审核',
      { confirmButtonText: '开始审核', cancelButtonText: '取消', type: 'info' }
    )
  } catch {
    return
  }
  autoReviewing.value = true
  try {
    const data = await api.qa.autoReview({ limit: 500 })
    const parts = [`扫描 ${data.scanned} 条`, `定稿 ${data.passed} 条`, `驳回 ${data.rejected} 条`]
    if (data.held) parts.push(`留人工 ${data.held} 条`)
    ElMessage.success(parts.join(' · '))
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    autoReviewing.value = false
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.qa-stats {
  margin-bottom: 14px;
}

.qa-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 12px 16px;
}

.seg {
  display: inline-flex;
  padding: 3px;
  border-radius: 10px;
  background: rgba(148, 163, 184, 0.12);
}

.seg-item {
  border: none;
  background: transparent;
  padding: 5px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  color: var(--jj-text-sub);
  cursor: pointer;
  transition: background 0.25s var(--jj-ease), color 0.25s var(--jj-ease);
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.seg-item:hover {
  color: var(--jj-text);
}

.seg-item.active {
  background: #fff;
  color: var(--jj-ink);
  font-weight: 550;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.1);
}

.seg-count {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  padding: 0 5px;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.16);
}

.seg-item.active .seg-count {
  background: rgba(232, 114, 44, 0.12);
  color: var(--jj-orange);
}

.tool-input {
  width: 220px;
}

.tool-select {
  width: 130px;
}

.qa-list {
  padding: 14px 16px 18px;
  min-height: 200px;
}

.qa-empty {
  padding: 60px 20px;
  text-align: center;
}

.empty-title {
  font-size: 15px;
  font-weight: 550;
  margin-bottom: 8px;
}

.qa-item {
  border: 1px solid var(--jj-line);
  border-radius: var(--jj-radius);
  padding: 14px 16px 12px;
  background: #fff;
  transition: border-color 0.28s var(--jj-ease), box-shadow 0.28s var(--jj-ease);
}

.qa-item + .qa-item {
  margin-top: 12px;
}

.qa-item:hover {
  border-color: rgba(232, 114, 44, 0.28);
  box-shadow: var(--jj-shadow-2);
}

.qa-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.grow {
  flex: 1;
}

.tag {
  font-size: 11.5px;
  line-height: 1;
  padding: 4px 8px;
  border-radius: 6px;
  font-weight: 550;
  white-space: nowrap;
}

.st-candidate {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.st-published {
  color: #15803d;
  background: rgba(22, 163, 74, 0.12);
}

.st-offline,
.st-merged,
.st-rejected {
  color: #64748b;
  background: rgba(100, 116, 139, 0.12);
}

.tag-suggest {
  color: #fff;
  background: var(--jj-orange);
}

.tag-ghost {
  color: var(--jj-text-sub);
  background: rgba(148, 163, 184, 0.12);
  font-weight: 400;
}

/* 自动审核预判：绿=可放行，琥珀=要人看，红=会被驳回。
   同一套配色和「待审核/已定稿」保持区分——这是审核建议，不是最终状态。 */
.rv-pass {
  color: #15803d;
  background: rgba(22, 163, 74, 0.1);
  border: 1px solid rgba(22, 163, 74, 0.22);
}

.rv-hold {
  color: #b45309;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.22);
}

.rv-reject {
  color: #b91c1c;
  background: rgba(220, 38, 38, 0.08);
  border: 1px solid rgba(220, 38, 38, 0.2);
}

.tag-auto {
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.08);
  border: 1px solid rgba(37, 99, 235, 0.18);
  font-weight: 400;
}

.qa-review {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.08);
  font-size: 12.5px;
  color: var(--jj-text-sub);
  line-height: 1.6;
}

.qa-review.reject {
  background: rgba(220, 38, 38, 0.06);
  color: #b91c1c;
}

.review-text {
  flex: 1;
}

.heat {
  font-size: 11.5px;
  color: var(--jj-text-mute);
  font-variant-numeric: tabular-nums;
  cursor: help;
}

.qa-question {
  display: flex;
  gap: 9px;
  align-items: flex-start;
  font-size: 14px;
  font-weight: 550;
  line-height: 1.65;
  color: var(--jj-ink);
}

.q-mark {
  flex: 0 0 20px;
  width: 20px;
  height: 20px;
  margin-top: 2px;
  border-radius: 6px;
  display: grid;
  place-items: center;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  background: var(--jj-orange);
}

.q-mark.a-mark {
  background: #2563eb;
}

.qa-answer {
  display: flex;
  gap: 9px;
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.8;
  color: var(--jj-text);
  cursor: pointer;
}

.a-text {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.qa-answer.open .a-text {
  display: block;
  overflow: visible;
}

.qa-aliases,
.qa-refs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  align-items: center;
  margin-top: 9px;
  font-size: 12px;
}

.alias-label {
  color: var(--jj-text-mute);
  flex: 0 0 auto;
}

.alias {
  padding: 2px 7px;
  border-radius: 5px;
  background: rgba(37, 99, 235, 0.07);
  color: #2563eb;
}

.ref {
  color: var(--jj-text-sub);
}

.qa-refs.warn .ref-none {
  color: #b45309;
}

.qa-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--jj-line-soft);
}

.metrics {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--jj-text-sub);
}

.metric b {
  font-weight: 600;
  color: var(--jj-text);
  font-variant-numeric: tabular-nums;
}

.metric.up b {
  color: #15803d;
}

.metric.down b {
  color: #d43b3b;
}

.metric.mute {
  color: var(--jj-text-mute);
}

.ops {
  display: flex;
  align-items: center;
  gap: 4px;
}

.drawer-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.field label {
  font-size: 12.5px;
  font-weight: 550;
  color: var(--jj-text);
}

.req {
  color: #e04a4a;
  margin-left: 2px;
}

.field-hint {
  font-size: 11.5px;
  color: var(--jj-text-mute);
  line-height: 1.6;
}

.merge-tip {
  font-size: 12.5px;
  line-height: 1.8;
  color: var(--jj-text-sub);
  margin-bottom: 12px;
}

.merge-source {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(148, 163, 184, 0.1);
  font-size: 13px;
  margin-bottom: 12px;
}

.merge-list {
  margin-top: 12px;
  max-height: 280px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.merge-opt {
  padding: 10px 12px;
  border: 1px solid var(--jj-line);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.2s var(--jj-ease), background 0.2s var(--jj-ease);
}

.merge-opt:hover {
  border-color: rgba(232, 114, 44, 0.4);
}

.merge-opt.active {
  border-color: var(--jj-orange);
  background: rgba(232, 114, 44, 0.06);
}

.merge-q {
  font-size: 13px;
  font-weight: 550;
  line-height: 1.6;
}

.merge-meta {
  margin-top: 4px;
  font-size: 11.5px;
  color: var(--jj-text-mute);
}

.merge-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 12.5px;
}

@media (max-width: 1280px) {
  .tool-input {
    width: 170px;
  }
}

@media (max-width: 1180px) {
  .qa-toolbar .grow {
    display: none;
  }

  /* 窄屏让分段筛选独占一行，搜索与下拉在第二行并排，避免排序框孤零零掉到最后 */
  .seg {
    width: 100%;
  }

  .tool-input {
    flex: 1 1 200px;
    width: auto;
  }

  .tool-select {
    flex: 0 0 128px;
    width: 128px;
  }
}

@media (max-width: 900px) {
  .qa-toolbar {
    align-items: stretch;
  }

  .seg {
    overflow-x: auto;
  }

  .tool-input,
  .tool-select {
    width: 100%;
  }

  .qa-bottom {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
