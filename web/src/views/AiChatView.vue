<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">AI 调试台</h2>
        <p class="jj-page-desc">
          用真实资料验证回答质量。左边是完整问答，右边只跑检索、不消耗 token，
          排查「答非所问」时先看右边命中了什么。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button :disabled="!messages.length" @click="clearChat">
          <el-icon><Delete /></el-icon>清空对话
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="!loading && !enabled"
      class="tip-alert"
      type="info"
      :closable="false"
      show-icon
      title="模型问答还没启用"
    >
      <template #default>
        <span>
          开关关闭时不会调用模型，但已经定稿的问答对仍然直接命中——这部分不需要 API Key。
          要让它能回答新问题，再去填一下 Key。
        </span>
        <el-button link type="primary" @click="$router.push('/ai/settings')">去配置</el-button>
      </template>
    </el-alert>

    <div class="chat-bar">
      <span class="bar-item">
        <el-icon><MagicStick /></el-icon>{{ form.chatModel || '未选择模型' }}
      </span>
      <span class="bar-sep" />
      <span class="bar-item">
        <el-icon><Files /></el-icon>可检索 {{ stats.searchable || 0 }} 份 / {{ stats.chunks || 0 }} 块
      </span>
      <span class="bar-sep" />
      <span class="bar-item" :class="enabled ? 'ok' : 'off'">
        <el-icon><CircleCheck v-if="enabled" /><CircleClose v-else /></el-icon>
        {{ enabled ? '模型已启用' : '模型未启用' }}
      </span>
      <span class="bar-spacer" />
      <span class="bar-hint">Enter 发送 · Shift + Enter 换行</span>
    </div>

    <div class="chat-grid">
      <div class="jj-card chat-card">
        <div ref="scrollRef" class="chat-body">
          <div v-if="!messages.length" class="chat-empty">
            <div class="empty-title">问点什么试试</div>
            <div class="jj-sub">
              回答只依据「已发布且解析成功」的资料。资料里没有的内容，模型会被要求直说没有。
            </div>
            <div class="suggests">
              <span v-for="item in suggests" :key="item" class="suggest" @click="ask(item)">{{ item }}</span>
            </div>
          </div>

          <div v-for="(msg, index) in messages" :key="index" class="msg" :class="msg.role">
            <div class="avatar">{{ msg.role === 'user' ? '我' : 'AI' }}</div>
            <div class="msg-main">
              <div v-if="msg.source && !msg.pending" class="src-badge" :class="msg.source">
                <template v-if="msg.source === 'qa'">
                  知识库直出 · 未调用模型<template v-if="msg.qaHitTotal"> · 第 {{ msg.qaHitTotal }} 次命中</template>
                </template>
                <template v-else-if="msg.source === 'model'">模型生成 · 已沉淀为待审核问答</template>
              </div>
              <div class="bubble" :class="{ error: msg.error }">
                <template v-if="msg.role === 'assistant' && msg.pending">
                  <span class="dot" /><span class="dot" /><span class="dot" />
                </template>
                <template v-else>{{ msg.content }}</template>
              </div>

              <div v-if="msg.meta" class="msg-meta">
                <span v-if="msg.source !== 'qa' && msg.meta.hitCount !== undefined">命中 {{ msg.meta.hitCount }} 段</span>
                <span v-if="msg.source !== 'qa' && msg.meta.latencyMs">耗时 {{ msg.meta.latencyMs }} ms</span>
                <span v-if="msg.meta.usage?.total_tokens">token {{ msg.meta.usage.total_tokens }}</span>
                <span v-if="msg.source !== 'qa' && msg.meta.model">{{ msg.meta.model }}</span>
              </div>

              <div v-if="msg.qaId && !msg.error && !msg.pending" class="feedback">
                <template v-if="msg.feedback">
                  <span class="fb-done">{{ msg.feedback === 'up' ? '已标记为有用' : '已标记为没用，会转人工复核' }}</span>
                </template>
                <template v-else>
                  <span class="fb-label">这个回答准吗</span>
                  <button class="fb-btn up" @click="feedback(msg, true)">有用</button>
                  <button class="fb-btn down" @click="feedback(msg, false)">没用</button>
                </template>
              </div>

              <div v-if="msg.references?.length" class="refs">
                <div class="refs-head">引用资料</div>
                <div
                  v-for="(ref, ri) in msg.references"
                  :key="ri"
                  class="ref"
                  :class="{ open: isOpen(index, ri) }"
                  @click="toggleRef(index, ri)"
                >
                  <div class="ref-line">
                    <span class="ref-no">{{ ri + 1 }}</span>
                    <span class="ref-title">{{ ref.docTitle }} · 第 {{ ref.seq }} 段</span>
                    <span class="ref-score">{{ (ref.score * 100).toFixed(0) }}%</span>
                  </div>
                  <div class="ref-text">{{ isOpen(index, ri) ? ref.content : snippet(ref.content) }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="chat-input">
          <el-input
            v-model="question"
            type="textarea"
            :rows="3"
            resize="none"
            maxlength="500"
            show-word-limit
            placeholder="例如：G74 端面切槽循环里 P 和 Q 分别代表什么？"
            :disabled="sending || !canChat"
            @keydown.enter.exact.prevent="send"
          />
          <div class="input-foot">
            <span class="jj-sub">
              {{ canChat ? '已定稿的问题直接命中问答库；新问题才调用模型，答完自动进候选池。' : '当前账号没有调用权限，只能使用右侧检索调试。' }}
            </span>
            <el-button type="primary" :loading="sending" :disabled="!canChat || !question.trim()" @click="send">
              <el-icon><Promotion /></el-icon>发送
            </el-button>
          </div>
        </div>
      </div>

      <div class="jj-card debug-card">
        <div class="jj-card-head">
          <div class="jj-card-title">检索调试<span class="jj-card-title-sub">不调用模型</span></div>
        </div>
        <div class="jj-card-body">
          <el-input
            v-model="probe"
            type="textarea"
            :rows="2"
            resize="none"
            maxlength="300"
            placeholder="输入问题，只看命中了哪些分块"
            @keydown.enter.exact.prevent="runProbe"
          />
          <div class="debug-params">
            <div class="param">
              <span class="param-label">条数</span>
              <el-slider v-model="topK" :min="1" :max="20" :step="1" size="small" />
              <span class="param-value">{{ topK }}</span>
            </div>
            <div class="param">
              <span class="param-label">阈值</span>
              <el-slider v-model="minScore" :min="0" :max="0.9" :step="0.01" size="small" />
              <span class="param-value">{{ minScore.toFixed(2) }}</span>
            </div>
          </div>
          <el-button class="probe-btn" :loading="probing" :disabled="!probe.trim()" @click="runProbe">
            <el-icon><Search /></el-icon>只检索
          </el-button>

          <div v-if="probeResult" class="probe-result">
            <div class="qa-probe">
              <div class="qa-probe-head">
                问答库
                <span v-if="probeResult.qaHits?.length" class="qa-probe-state hit">
                  命中 {{ probeResult.qaHits.length }} 条
                </span>
                <span v-else class="qa-probe-state miss">未命中</span>
                <span class="qa-probe-th">阈值 {{ ((probeResult.qaThreshold || 0.5) * 100).toFixed(0) }}%</span>
              </div>
              <div v-for="hit in probeResult.qaHits || []" :key="hit.id" class="qa-probe-item">
                <div class="qa-probe-q">
                  <span>{{ hit.question }}</span>
                  <b>{{ (hit.score * 100).toFixed(0) }}%</b>
                </div>
                <div class="qa-probe-meta">已命中 {{ hit.hitCount }} 次 · 命中后直接返回，不调模型</div>
              </div>
              <div v-if="!probeResult.qaHits?.length" class="qa-probe-tip">
                问答库没有命中，这次会走文档检索 + 模型作答，答完自动进候选池。
              </div>
            </div>

            <div class="probe-summary">
              候选 {{ probeResult.candidates }} 块 · 命中 {{ probeResult.count }} 块
              <span v-if="probeResult.terms?.length" class="terms">
                <span v-for="t in probeResult.terms" :key="t" class="term">{{ t }}</span>
              </span>
            </div>
            <div v-if="!probeResult.hits.length" class="jj-empty probe-empty">
              没有命中。可能问题里的关键词和资料用词不一致，换个说法，或把资料正文补全。
            </div>
            <div v-for="(hit, hi) in probeResult.hits" :key="hi" class="hit">
              <div class="hit-head">
                <span class="hit-title">{{ hit.docTitle }} · 第 {{ hit.seq }} 段</span>
                <span class="hit-score">{{ (hit.score * 100).toFixed(0) }}%</span>
              </div>
              <div class="hit-bar"><i :style="{ width: Math.max(4, hit.score * 100) + '%' }" /></div>
              <div class="hit-text">{{ snippet(hit.content) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canChat = canWrite('ai')

const loading = ref(false)
const sending = ref(false)
const probing = ref(false)
const enabled = ref(false)
const question = ref('')
const probe = ref('')
const messages = ref([])
const probeResult = ref(null)
const scrollRef = ref(null)
const openRefs = reactive({})

const stats = ref({ searchable: 0, chunks: 0 })
const form = ref({ chatModel: '' })
const topK = ref(5)
const minScore = ref(0.15)

const suggests = [
  'G74 端面切槽循环怎么用？',
  '走心机断屑不好应该调哪个参数？',
  '刀具磨损过快有哪些常见原因？'
]

function snippet(text, max = 120) {
  const value = (text || '').replace(/\s+/g, ' ').trim()
  return value.length > max ? value.slice(0, max) + '…' : value
}

function refKey(mi, ri) {
  return mi + '-' + ri
}

function isOpen(mi, ri) {
  return !!openRefs[refKey(mi, ri)]
}

function toggleRef(mi, ri) {
  const key = refKey(mi, ri)
  openRefs[key] = !openRefs[key]
}

function scrollToBottom() {
  nextTick(() => {
    const el = scrollRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

async function load() {
  loading.value = true
  try {
    const data = await api.ai.configs()
    const cfg = data.config || {}
    enabled.value = !!cfg.enabled && !!data.hasKey
    form.value = { chatModel: cfg.chatModel || '' }
    stats.value = data.stats || stats.value
    if (cfg.topK) topK.value = Number(cfg.topK)
    if (cfg.minScore !== undefined) minScore.value = Number(cfg.minScore)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function ask(text) {
  question.value = text
  send()
}

async function send() {
  const text = question.value.trim()
  if (!text || sending.value || !canChat) return

  messages.value.push({ role: 'user', content: text })
  const pending = { role: 'assistant', content: '', pending: true }
  messages.value.push(pending)
  question.value = ''
  sending.value = true
  scrollToBottom()

  try {
    const data = await api.ai.chat({ question: text, topK: topK.value, minScore: minScore.value })
    Object.assign(pending, {
      pending: false,
      content: data.answer || '（模型没有返回内容）',
      references: data.references || [],
      source: data.source || 'model',
      qaId: data.qaId || 0,
      qaHitTotal: data.qaHitTotal || 0,
      feedback: '',
      meta: {
        hitCount: data.hitCount || 0,
        latencyMs: data.latencyMs || 0,
        usage: data.usage || null,
        model: data.source === 'qa' ? '知识库' : data.model || ''
      }
    })
  } catch (e) {
    Object.assign(pending, { pending: false, content: e.message, error: true })
    ElMessage.error(e.message)
  } finally {
    sending.value = false
    messages.value = [...messages.value]
    scrollToBottom()
  }
}

async function feedback(msg, helpful) {
  if (!msg.qaId || msg.feedback) return
  try {
    const data = await api.qa.feedback(msg.qaId, helpful)
    msg.feedback = helpful ? 'up' : 'down'
    if (data.autoOffline) {
      ElMessage.warning('已记录。这条答案差评过多，已自动下架，请到「问答沉淀」复核')
    } else {
      ElMessage.success(helpful ? '已记录，谢谢' : '已记录，会转人工复核')
    }
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function runProbe() {
  const text = probe.value.trim()
  if (!text) return
  probing.value = true
  try {
    probeResult.value = await api.ai.retrieve({
      query: text,
      topK: topK.value,
      minScore: minScore.value,
      debug: true
    })
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    probing.value = false
  }
}

function clearChat() {
  messages.value = []
  Object.keys(openRefs).forEach((k) => delete openRefs[k])
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.tip-alert {
  margin-bottom: 14px;
}

.chat-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  margin-bottom: 14px;
  border-radius: 10px;
  background: rgba(148, 163, 184, 0.1);
  font-size: 13px;
  color: var(--jj-text-sub, #64748b);
  flex-wrap: wrap;
}

.bar-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.bar-item.ok {
  color: #16a34a;
}

.bar-item.off {
  color: #d97706;
}

.bar-sep {
  width: 1px;
  height: 12px;
  background: rgba(148, 163, 184, 0.4);
}

.bar-spacer {
  flex: 1;
}

.bar-hint {
  font-size: 12px;
  opacity: 0.75;
}

.chat-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 380px;
  gap: 14px;
  align-items: start;
}

.chat-card {
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 320px);
}

.chat-body {
  flex: 1;
  overflow-y: auto;
  padding: 18px;
  max-height: calc(100vh - 400px);
  min-height: 320px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chat-empty {
  margin: auto;
  text-align: center;
  max-width: 460px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 40px 0;
}

.empty-title {
  font-size: 16px;
  font-weight: 650;
}

.suggests {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  margin-top: 10px;
}

.suggest {
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 12px;
  cursor: pointer;
  background: rgba(37, 99, 235, 0.08);
  color: #2563eb;
  transition: background 0.15s;
}

.suggest:hover {
  background: rgba(37, 99, 235, 0.16);
}

.msg {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.msg.user {
  flex-direction: row-reverse;
}

.avatar {
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  border-radius: 9px;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
  background: var(--jj-orange, #f97316);
}

.msg.assistant .avatar {
  background: #2563eb;
}

.msg-main {
  max-width: 78%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.msg.user .msg-main {
  align-items: flex-end;
}

.bubble {
  padding: 10px 14px;
  border-radius: 12px;
  background: rgba(148, 163, 184, 0.12);
  font-size: 13.5px;
  line-height: 1.75;
  white-space: pre-wrap;
  word-break: break-word;
}

.msg.user .bubble {
  background: rgba(249, 115, 22, 0.12);
  color: #9a3412;
}

.bubble.error {
  background: rgba(220, 38, 38, 0.09);
  color: #b91c1c;
}

.dot {
  display: inline-block;
  width: 5px;
  height: 5px;
  margin-right: 4px;
  border-radius: 50%;
  background: #94a3b8;
  animation: blink 1.2s infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes blink {
  0%,
  100% {
    opacity: 0.25;
  }
  50% {
    opacity: 1;
  }
}

.msg-meta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 11.5px;
  color: #94a3b8;
}

.src-badge {
  align-self: flex-start;
  font-size: 11px;
  line-height: 1.5;
  padding: 3px 8px;
  border-radius: 6px;
}

.src-badge.qa {
  color: #15803d;
  background: rgba(22, 163, 74, 0.09);
  border: 1px solid rgba(22, 163, 74, 0.2);
}

.src-badge.model {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.07);
  border: 1px solid rgba(37, 99, 235, 0.16);
}

.feedback {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.fb-label {
  color: #94a3b8;
}

.fb-btn {
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: #fff;
  border-radius: 7px;
  padding: 3px 10px;
  font-size: 12px;
  color: #64748b;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s, background 0.2s;
}

.fb-btn.up:hover {
  color: #15803d;
  border-color: rgba(22, 163, 74, 0.5);
  background: rgba(22, 163, 74, 0.06);
}

.fb-btn.down:hover {
  color: #d43b3b;
  border-color: rgba(212, 59, 59, 0.5);
  background: rgba(212, 59, 59, 0.06);
}

.fb-done {
  color: #94a3b8;
}

.qa-probe {
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 10px;
  padding: 10px 12px;
  margin-bottom: 12px;
  background: rgba(148, 163, 184, 0.05);
}

.qa-probe-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  font-weight: 600;
  color: #475569;
  margin-bottom: 8px;
}

.qa-probe-state.hit {
  color: #15803d;
  font-size: 11.5px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 5px;
  background: rgba(22, 163, 74, 0.1);
}

.qa-probe-state.miss {
  color: #94a3b8;
  font-size: 11.5px;
  font-weight: 400;
  padding: 1px 6px;
  border-radius: 5px;
  background: rgba(148, 163, 184, 0.15);
}

.qa-probe-th {
  margin-left: auto;
  font-weight: 400;
  font-size: 11px;
  color: #94a3b8;
}

.qa-probe-item + .qa-probe-item {
  margin-top: 8px;
}

.qa-probe-q {
  display: flex;
  gap: 8px;
  font-size: 12.5px;
  line-height: 1.6;
  color: #1f2836;
}

.qa-probe-q span {
  flex: 1;
  min-width: 0;
}

.qa-probe-q b {
  color: #15803d;
  font-variant-numeric: tabular-nums;
}

.qa-probe-meta {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

.qa-probe-tip {
  font-size: 11.5px;
  line-height: 1.7;
  color: #94a3b8;
}

.refs {
  border-radius: 10px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  overflow: hidden;
}

.refs-head {
  padding: 7px 12px;
  font-size: 12px;
  font-weight: 600;
  background: rgba(148, 163, 184, 0.08);
  color: #64748b;
}

.ref {
  padding: 9px 12px;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  cursor: pointer;
}

.ref:hover {
  background: rgba(37, 99, 235, 0.04);
}

.ref-line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
}

.ref-no {
  width: 18px;
  height: 18px;
  border-radius: 5px;
  display: grid;
  place-items: center;
  font-size: 11px;
  font-weight: 700;
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.ref-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 550;
}

.ref-score {
  font-variant-numeric: tabular-nums;
  color: var(--jj-orange, #f97316);
  font-weight: 650;
}

.ref-text {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.7;
  color: #64748b;
}

.ref.open .ref-text {
  white-space: pre-wrap;
  max-height: 260px;
  overflow-y: auto;
}

.chat-input {
  border-top: 1px solid rgba(148, 163, 184, 0.18);
  padding: 12px 14px 14px;
}

.input-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
}

.debug-card {
  position: sticky;
  top: 0;
}

.debug-params {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.param {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: #64748b;
}

.param-label {
  width: 30px;
  flex: 0 0 30px;
}

.param .el-slider {
  flex: 1;
}

.param-value {
  width: 36px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange, #f97316);
}

.probe-btn {
  width: 100%;
  margin-top: 12px;
}

.probe-result {
  margin-top: 14px;
  border-top: 1px dashed rgba(148, 163, 184, 0.3);
  padding-top: 12px;
}

.probe-summary {
  font-size: 12px;
  color: #64748b;
  line-height: 1.8;
}

.terms {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.term {
  padding: 1px 6px;
  border-radius: 5px;
  background: rgba(148, 163, 184, 0.15);
  font-size: 11px;
  color: #64748b;
}

.probe-empty {
  padding: 16px 4px;
  text-align: center;
  font-size: 12px;
  line-height: 1.7;
}

.hit {
  padding: 10px 0;
  border-bottom: 1px dashed rgba(148, 163, 184, 0.22);
}

.hit:last-child {
  border-bottom: none;
}

.hit-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12.5px;
}

.hit-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hit-score {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange, #f97316);
}

.hit-bar {
  height: 3px;
  border-radius: 2px;
  background: rgba(148, 163, 184, 0.2);
  margin: 6px 0;
  overflow: hidden;
}

.hit-bar i {
  display: block;
  height: 100%;
  border-radius: 2px;
  background: linear-gradient(90deg, #fbbf24, #f97316);
}

.hit-text {
  font-size: 12px;
  line-height: 1.65;
  color: #64748b;
}

@media (max-width: 1200px) {
  .chat-grid {
    grid-template-columns: 1fr;
  }

  .debug-card {
    position: static;
  }
}
</style>
