<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">AI 助手配置</h2>
        <p class="jj-page-desc">
          模型接入、生成参数与检索参数都在这里维护。密钥只保存在数据库，页面回显时会脱敏，
          保存后小程序端与调试台同时生效。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" :loading="testing" @click="testConnection">
          <el-icon><Connection /></el-icon>测试连接
        </el-button>
        <el-button v-if="canEdit" type="primary" :loading="saving" @click="save">
          <el-icon><Check /></el-icon>保存配置
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="!loading && !hasKey"
      class="tip-alert"
      type="warning"
      :closable="false"
      show-icon
      title="还没有配置 API Key"
      description="没有密钥时 AI 不会真正回答。可以先把资料整理好并发布，再回来填密钥。"
    />

    <div class="jj-stats">
      <div class="jj-stat" :class="ready ? 'ok' : ''">
        <div class="jj-stat-label"><el-icon><MagicStick /></el-icon>接入状态</div>
        <div class="jj-stat-value status-text">{{ statusText }}</div>
        <div class="jj-stat-foot">{{ form.chatModel || '未选择模型' }}</div>
      </div>
      <div class="jj-stat" :class="stats.searchable ? '' : 'warn'">
        <div class="jj-stat-label"><el-icon><Files /></el-icon>可检索资料</div>
        <div class="jj-stat-value">
          {{ stats.searchable || 0 }}<span class="jj-stat-unit">份</span>
        </div>
        <div class="jj-stat-foot">已发布 {{ stats.published || 0 }} 份 · 草稿不计入</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Grid /></el-icon>知识分块</div>
        <div class="jj-stat-value">{{ stats.chunks || 0 }}<span class="jj-stat-unit">块</span></div>
        <div class="jj-stat-foot">每块约 500 字，检索的最小单位</div>
      </div>
      <div class="jj-stat" :class="lastTest.ok ? 'ok' : lastTest.at ? 'bad' : ''">
        <div class="jj-stat-label"><el-icon><Odometer /></el-icon>最近测试</div>
        <div class="jj-stat-value status-text">{{ lastTest.at ? (lastTest.ok ? '通过' : '失败') : '未测试' }}</div>
        <div class="jj-stat-foot">
          <template v-if="lastTest.at">{{ lastTest.latency }} ms · {{ fmt(lastTest.at) }}</template>
          <template v-else>保存后点右上角「测试连接」</template>
        </div>
      </div>
    </div>

    <div class="ai-grid" v-loading="loading">
      <div class="ai-main">
        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">模型接入<span class="jj-card-title-sub">OpenAI 兼容协议</span></div>
          </div>
          <div class="jj-card-body">
            <el-form label-width="110px" :disabled="!canEdit">
              <el-form-item label="启用 AI">
                <el-switch v-model="form.enabled" :disabled="!canEdit" />
                <span class="jj-sub inline-tip">
                  {{ form.enabled ? '已开启，调试台与小程序可调用' : '已关闭，所有问答接口会直接拒绝' }}
                </span>
              </el-form-item>
              <el-form-item label="服务商">
                <el-select v-model="form.provider" style="width: 220px" :disabled="!canEdit">
                  <el-option label="DeepSeek" value="deepseek" />
                  <el-option label="OpenAI" value="openai" />
                  <el-option label="自建 / 代理" value="custom" />
                </el-select>
              </el-form-item>
              <el-form-item label="接口地址">
                <el-input v-model="form.baseUrl" :disabled="!canEdit" placeholder="https://api.deepseek.com" />
                <div class="jj-sub field-tip">
                  填到域名即可，系统会自动补 <code>/chat/completions</code>；也可以直接填到 <code>/v1</code>。
                </div>
              </el-form-item>
              <el-form-item label="API Key">
                <div class="key-row">
                  <el-input
                    v-model="form.apiKey"
                    type="password"
                    show-password
                    :disabled="!canEdit"
                    :placeholder="hasKey ? '已保存，修改时直接覆盖' : 'sk-...'"
                  />
                  <el-button v-if="canEdit && hasKey" text type="danger" @click="clearKey">清除</el-button>
                </div>
                <div class="jj-sub field-tip">
                  数据库中保存的是明文密钥，接口只返回 <code>sk-****后四位</code>；
                  保存时如果这串没改动，会保留原密钥。
                </div>
              </el-form-item>
              <div class="row-2">
                <el-form-item label="对话模型">
                  <el-select
                    v-model="form.chatModel"
                    filterable
                    allow-create
                    default-first-option
                    :disabled="!canEdit"
                    placeholder="deepseek-chat"
                  >
                    <el-option label="deepseek-chat（快，日常问答）" value="deepseek-chat" />
                    <el-option label="deepseek-reasoner（慢，复杂推理）" value="deepseek-reasoner" />
                  </el-select>
                </el-form-item>
                <el-form-item label="向量模型">
                  <el-input v-model="form.embeddingModel" :disabled="!canEdit" placeholder="留空 = 关键词检索" />
                </el-form-item>
              </div>
              <div class="jj-sub field-tip block-tip">
                向量模型留空时使用内置的关键词检索（中文按二元词组切分打分），
                不依赖外部 embedding 服务；填了也不会立即生效，需要先接入向量库。
              </div>
            </el-form>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">生成参数</div>
          </div>
          <div class="jj-card-body">
            <el-form label-width="110px" :disabled="!canEdit">
              <el-form-item label="温度">
                <div class="slider-row">
                  <el-slider v-model="form.temperature" :min="0" :max="2" :step="0.05" :disabled="!canEdit" />
                  <span class="slider-value">{{ form.temperature.toFixed(2) }}</span>
                </div>
                <div class="jj-sub field-tip">工艺参数类问答建议 0.2 - 0.4，越低越稳定，越高越发散。</div>
              </el-form-item>
              <el-form-item label="最大输出">
                <el-input-number v-model="form.maxTokens" :min="64" :max="8192" :step="100" :disabled="!canEdit" />
                <span class="jj-sub inline-tip">token，约等于 {{ Math.round(form.maxTokens * 0.7) }} 个汉字</span>
              </el-form-item>
              <el-form-item label="系统提示词">
                <el-input
                  v-model="form.systemPrompt"
                  type="textarea"
                  :rows="6"
                  maxlength="4000"
                  show-word-limit
                  :disabled="!canEdit"
                  placeholder="留空使用内置的机匠助手指令"
                />
                <div class="jj-sub field-tip">
                  这段指令决定回答风格与边界。内置版本要求「只依据资料回答，资料里没有就直说」。
                </div>
              </el-form-item>
              <el-form-item label="无答案话术">
                <el-input v-model="form.refuseText" :disabled="!canEdit" placeholder="兜底关闭或模型不可用时的回复" />
                <div class="jj-sub field-tip">只在「未命中兜底」关闭或模型调用失败时使用，不会额外消耗 token。</div>
              </el-form-item>
              <el-form-item label="未命中兜底">
                <div class="switch-row">
                  <el-switch v-model="form.fallbackEnabled" :disabled="!canEdit" />
                  <span class="jj-sub inline-tip">
                    {{ form.fallbackEnabled ? '已开启：资料没命中时由模型凭通用知识作答' : '已关闭：直接返回上面的无答案话术' }}
                  </span>
                </div>
                <div class="jj-sub field-tip">
                  开启后不会再出现「没找到」的死胡同：答案带【未引用资料】声明返回，同时自动进「问答沉淀」候选池，
                  人工核实定稿后才成为可直接命中的标准答案。这是知识库从零积累内容的主要入口。
                </div>
                <div class="jj-sub field-tip">
                  兜底受「
                  <el-link type="primary" :underline="false" @click="$router.push('/kb/rule')">知识规则</el-link>
                  」约束：范围外的问题不调模型、不入库；答案太短或像拒答也不会入库。
                </div>
              </el-form-item>
              <el-form-item v-if="form.fallbackEnabled" label="兜底提示词">
                <el-input
                  v-model="form.fallbackPrompt"
                  type="textarea"
                  :rows="5"
                  maxlength="4000"
                  show-word-limit
                  :disabled="!canEdit"
                  placeholder="留空使用内置的兜底指令"
                />
                <div class="jj-sub field-tip">
                  兜底时模型拿不到任何资料，这段指令要强调「不确定就说不确定」和「具体参数需按手册确认」，
                  避免它编出一个看起来合理的切削用量。
                </div>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">检索参数</div>
          </div>
          <div class="jj-card-body">
            <el-form label-width="110px" :disabled="!canEdit">
              <el-form-item label="检索条数">
                <div class="slider-row">
                  <el-slider v-model="form.topK" :min="1" :max="20" :step="1" :disabled="!canEdit" />
                  <span class="slider-value">{{ form.topK }} 条</span>
                </div>
                <div class="jj-sub field-tip">每次提问带几段资料给模型。条数多召回全，但上下文变长、响应变慢。</div>
              </el-form-item>
              <el-form-item label="最低相关度">
                <div class="slider-row">
                  <el-slider v-model="form.minScore" :min="0" :max="0.9" :step="0.01" :disabled="!canEdit" />
                  <span class="slider-value">{{ form.minScore.toFixed(2) }}</span>
                </div>
                <div class="jj-sub field-tip">低于该分数的分块不进上下文。调高更严谨，调低更容易给出答案。</div>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>

      <div class="ai-side">
        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">就绪检查</div>
          </div>
          <div class="jj-card-body">
            <ul class="checklist">
              <li v-for="item in checklist" :key="item.label" :class="{ done: item.done }">
                <el-icon class="check-icon">
                  <CircleCheck v-if="item.done" />
                  <CircleClose v-else />
                </el-icon>
                <div class="check-body">
                  <div class="check-label">{{ item.label }}</div>
                  <div class="jj-sub">{{ item.hint }}</div>
                </div>
              </li>
            </ul>
            <div v-if="!stats.searchable" class="check-action">
              <el-button size="small" @click="goKb">去发布资料</el-button>
            </div>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">调用说明</div>
          </div>
          <div class="jj-card-body">
            <p class="note">
              问答流程：问题 → 中文二元词组切分 → 在已发布资料的分块里粗召回 →
              按覆盖率与词频打分 → 取前 {{ form.topK }} 条拼进上下文 → 调用模型。
            </p>
            <p class="note">
              模型只拿到检索到的分块，不接触原始资料表，因此调低「检索条数」会直接影响回答覆盖面。
            </p>
            <p class="note">
              调试台可以先用「只检索」验证召回，确认命中正确再提问，避免反复消耗 token。
            </p>
          </div>
        </div>
      </div>
    </div>

    <el-alert
      v-if="testResult"
      class="test-alert"
      :type="testResult.ok ? 'success' : 'error'"
      :closable="true"
      show-icon
      :title="testResult.ok ? '连接正常' : '连接失败'"
      @close="testResult = null"
    >
      <div class="test-detail">
        <span>{{ testResult.message }}</span>
        <span v-if="testResult.latencyMs"> · 耗时 {{ testResult.latencyMs }} ms</span>
        <span v-if="testResult.model"> · 模型 {{ testResult.model }}</span>
        <span v-if="testResult.reply"> · 回复「{{ testResult.reply }}」</span>
      </div>
    </el-alert>
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
// 必须用 computed：刷新页面时权限是异步加载的，setup 阶段读一次会永久拿到空权限，
// 表现是整个配置页变成只读。
const canEdit = computed(() => canWrite('ai'))

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const hasKey = ref(false)
const ready = ref(false)
const testResult = ref(null)

const stats = ref({ docs: 0, published: 0, chunks: 0, searchable: 0, parseFailed: 0 })
const lastTest = ref({ at: null, ok: false, message: '', latency: 0 })

const form = reactive({
  enabled: false,
  provider: 'deepseek',
  baseUrl: 'https://api.deepseek.com',
  apiKey: '',
  chatModel: 'deepseek-chat',
  embeddingModel: '',
  temperature: 0.3,
  maxTokens: 1200,
  topK: 5,
  minScore: 0.15,
  systemPrompt: '',
  refuseText: '',
  fallbackEnabled: true,
  fallbackPrompt: ''
})

const statusText = computed(() => {
  if (!form.enabled) return '已关闭'
  if (!hasKey.value) return '缺密钥'
  return ready.value ? '可用' : '待补资料'
})

const testHint = computed(() => {
  if (!lastTest.value.at) return '保存后点「测试连接」验证地址与密钥'
  if (lastTest.value.ok) return `连接正常 · ${lastTest.value.latency} ms`
  const msg = lastTest.value.message || ''
  if (/401|403|Authentication|Unauthorized/i.test(msg)) return '鉴权失败，检查 API Key 是否正确'
  if (/timeout|超时/i.test(msg)) return '请求超时，检查接口地址与网络'
  if (/404|model/i.test(msg)) return '地址或模型名不对，检查接口地址与模型'
  return msg.length > 42 ? msg.slice(0, 42) + '…' : msg
})

const checklist = computed(() => [
  {
    label: '填写 API Key',
    done: hasKey.value,
    hint: hasKey.value ? '已保存，密钥在页面上脱敏显示' : '没有密钥就无法调用模型'
  },
  {
    label: '打开 AI 助手开关',
    done: form.enabled,
    hint: form.enabled ? '接口可被调用' : '关闭状态下所有问答请求会被拒绝'
  },
  {
    label: '有可检索的资料',
    done: stats.value.searchable > 0,
    hint: `当前 ${stats.value.searchable || 0} 份已发布且解析成功`
  },
  {
    label: '已生成知识分块',
    done: stats.value.chunks > 0,
    hint: `共 ${stats.value.chunks || 0} 块，分块是检索的最小单位`
  },
  {
    label: '连接测试通过',
    done: lastTest.value.ok,
    hint: testHint.value
  }
])

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

function applyConfig(data) {
  const cfg = data.config || {}
  Object.assign(form, {
    enabled: !!cfg.enabled,
    provider: cfg.provider || 'deepseek',
    baseUrl: cfg.baseUrl || 'https://api.deepseek.com',
    apiKey: cfg.apiKey || '',
    chatModel: cfg.chatModel || 'deepseek-chat',
    embeddingModel: cfg.embeddingModel || '',
    temperature: Number(cfg.temperature ?? 0.3),
    maxTokens: Number(cfg.maxTokens ?? 1200),
    topK: Number(cfg.topK ?? 5),
    minScore: Number(cfg.minScore ?? 0.15),
    systemPrompt: cfg.systemPrompt || '',
    refuseText: cfg.refuseText || '',
    fallbackEnabled: cfg.fallbackEnabled !== false,
    fallbackPrompt: cfg.fallbackPrompt || ''
  })
  hasKey.value = !!data.hasKey
  ready.value = !!data.ready
  stats.value = data.stats || stats.value
  if (data.lastTest?.at) {
    lastTest.value = {
      at: data.lastTest.at,
      ok: !!data.lastTest.ok,
      message: data.lastTest.message || '',
      latency: data.lastTest.latencyMs || 0
    }
  }
}

async function load() {
  loading.value = true
  try {
    const data = await api.ai.configs()
    applyConfig(data)
  } catch (e) {
    if (!String(e.message).includes('权限')) ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!canEdit.value) return
  saving.value = true
  try {
    const data = await api.ai.save({ ...form })
    applyConfig({ ...data, lastTest: { at: lastTest.value.at, ok: lastTest.value.ok, message: lastTest.value.message, latencyMs: lastTest.value.latency } })
    ElMessage.success('AI 配置已保存')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!canEdit.value) return
  if (!hasKey.value && !form.apiKey) {
    ElMessage.warning('先填写 API Key 再测试')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const data = await api.ai.test()
    testResult.value = data
    lastTest.value = { at: new Date(), ok: !!data.ok, message: data.message, latency: data.latencyMs || 0 }
    if (data.ok) ElMessage.success('连接正常')
  } catch (e) {
    testResult.value = { ok: false, message: e.message }
  } finally {
    testing.value = false
  }
}

async function clearKey() {
  try {
    await ElMessageBox.confirm('清除后 AI 将无法回答，确定要清除已保存的密钥吗？', '清除密钥', {
      confirmButtonText: '清除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  saving.value = true
  try {
    const data = await api.ai.save({ ...form, apiKey: '' })
    applyConfig({ ...data })
    testResult.value = null
    ElMessage.success('密钥已清除')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

function goKb() {
  router.push('/kb')
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.tip-alert {
  margin-bottom: 14px;
}

.ai-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 14px;
  align-items: start;
}

.ai-main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.ai-side {
  display: flex;
  flex-direction: column;
  gap: 14px;
  position: sticky;
  top: 0;
}

.status-text {
  font-size: 22px;
}

.inline-tip {
  margin-left: 10px;
  line-height: 1.5;
}

.field-tip {
  margin-top: 4px;
  line-height: 1.6;
  width: 100%;
}

.block-tip {
  margin: -6px 0 4px 110px;
}

.field-tip code,
.block-tip code {
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(148, 163, 184, 0.18);
  font-size: 12px;
}

.key-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.key-row .el-input {
  max-width: 420px;
}

.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}

.slider-row {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
}

.slider-row .el-slider {
  flex: 1;
  max-width: 360px;
}

.slider-value {
  min-width: 56px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.checklist {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.checklist li {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.check-icon {
  margin-top: 2px;
  font-size: 16px;
  color: #dc2626;
}

.checklist li.done .check-icon {
  color: #16a34a;
}

.check-body {
  min-width: 0;
}

.check-label {
  font-size: 13px;
  font-weight: 600;
}

.checklist li.done .check-label {
  color: #15803d;
}

.check-action {
  margin-top: 14px;
}

.note {
  margin: 0 0 10px;
  font-size: 13px;
  line-height: 1.75;
  color: var(--jj-text-sub, #64748b);
}

.note:last-child {
  margin-bottom: 0;
}

.test-alert {
  margin-top: 14px;
}

.test-detail {
  font-size: 13px;
  line-height: 1.6;
}

@media (max-width: 1200px) {
  .ai-grid {
    grid-template-columns: 1fr;
  }

  .ai-side {
    position: static;
  }
}
</style>
