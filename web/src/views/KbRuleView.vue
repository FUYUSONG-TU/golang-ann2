<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">知识规则</h2>
        <p class="jj-page-desc">
          规定这个知识库收什么、不收什么。判定在服务端执行：范围外的问题直接拒答，
          不调用模型，也不会写进问答库——不指望模型自己守规矩。
        </p>
      </div>
      <div class="jj-page-actions">
        <template v-if="canEdit">
          <el-button :disabled="loading || saving" @click="restoreDefaults">恢复默认词表</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存规则</el-button>
        </template>
        <el-tag v-else type="info" effect="plain">只读</el-tag>
      </div>
    </div>

    <el-alert
      class="tip-alert"
      type="info"
      :closable="false"
      show-icon
      title="这里定的是准入门槛，不是提示词"
      description="所有自动入库都要经过「沉淀闸门」。规则写在服务端，即使模型不按指令作答，也拦得住；规则一旦放宽，噪声就会直接进候选池。"
    />

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Filter /></el-icon>准入词</div>
        <div class="jj-stat-value">{{ stats.includeCount || 0 }}<span class="jj-stat-unit">个</span></div>
        <div class="jj-stat-foot">命中任意一个即视为收录范围内</div>
      </div>
      <div class="jj-stat" :class="quotaUsed >= 0.9 ? 'warn' : ''">
        <div class="jj-stat-label"><el-icon><Download /></el-icon>今日入库</div>
        <div class="jj-stat-value">
          {{ stats.todayCollected || 0 }}<span class="jj-stat-unit">条</span>
        </div>
        <div class="jj-stat-foot">
          {{ form.dailyQuota > 0 ? `每日上限 ${form.dailyQuota} 条` : '未设上限' }}
        </div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Clock /></el-icon>待审核</div>
        <div class="jj-stat-value">{{ stats.pending || 0 }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">其中兜底无出处 {{ stats.fallbackPending || 0 }} 条</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><Select /></el-icon>已定稿</div>
        <div class="jj-stat-value">{{ stats.published || 0 }}<span class="jj-stat-unit">条</span></div>
        <div class="jj-stat-foot">定稿后可直接命中，不再调用模型</div>
      </div>
    </div>

    <div class="rule-grid" v-loading="loading">
      <div class="rule-main">
        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">领域定义<span class="jj-card-title-sub">写进提示词，也用于拒答话术</span></div>
          </div>
          <div class="jj-card-body">
            <el-form label-width="110px" :disabled="!canEdit">
              <el-form-item label="领域名称">
                <el-input v-model="form.name" maxlength="32" show-word-limit placeholder="走心机与数控加工" />
                <div class="jj-sub field-tip">学员看到「本助手只回答 XX 相关问题」时用的就是这个名字。</div>
              </el-form-item>
              <el-form-item label="收录范围">
                <el-input
                  v-model="form.scope"
                  type="textarea"
                  :rows="4"
                  maxlength="1000"
                  show-word-limit
                  placeholder="描述这个知识库负责什么、不负责什么"
                />
                <div class="jj-sub field-tip">
                  这段文字会附在兜底作答的指令后面，让模型知道自己的职责边界，不要顺着问题往外滑。
                </div>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">
              准入词表
              <span class="jj-card-title-sub">每行一个，命中 {{ form.minHits }} 个即放行</span>
            </div>
            <div class="jj-card-extra">
              <span class="word-count">{{ includeWords.length }} 个词</span>
            </div>
          </div>
          <div class="jj-card-body">
            <el-input
              v-model="form.includeText"
              type="textarea"
              :rows="12"
              :disabled="!canEdit"
              class="word-area"
              placeholder="走心机&#10;数控车&#10;断屑&#10;..."
            />
            <div class="jj-sub field-tip">
              词越全，真问题越不会被误挡。宁可多放边缘词——它们还要再过一遍闸门；
              漏掉一个词，学员反而会被直接拒答。
            </div>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">
              排除词表
              <span class="jj-card-title-sub">只在没命中准入词时生效</span>
            </div>
            <div class="jj-card-extra">
              <span class="word-count">{{ excludeWords.length }} 个词</span>
            </div>
          </div>
          <div class="jj-card-body">
            <el-input
              v-model="form.excludeText"
              type="textarea"
              :rows="6"
              :disabled="!canEdit"
              class="word-area"
              placeholder="天气&#10;股票&#10;python"
            />
            <div class="jj-sub field-tip">
              命中准入词就不会被排除词挡住——「翻译一下这段 G 代码」命中 <code>g代码</code>，照常放行。
              只放明确无关的词，模糊词交给闸门兜。
            </div>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">闸门参数<span class="jj-card-title-sub">决定答案能不能进库</span></div>
          </div>
          <div class="jj-card-body">
            <el-form label-width="130px" :disabled="!canEdit">
              <el-form-item label="最少命中词数">
                <el-input-number v-model="form.minHits" :min="1" :max="5" :step="1" />
                <span class="jj-sub inline-tip">设为 1 最宽松；设 2 以上会明显收紧，只在误答很多时才调高。</span>
              </el-form-item>
              <el-form-item label="范围外处理">
                <el-radio-group v-model="form.policy">
                  <el-radio value="strict">直接拒答（推荐）</el-radio>
                  <el-radio value="open">照常作答但不入库</el-radio>
                </el-radio-group>
                <div class="jj-sub field-tip">
                  严格模式不花钱也不产生噪声；宽松模式适合词表还没配全的过渡期。
                </div>
              </el-form-item>
              <el-form-item label="自动入库">
                <el-switch v-model="form.collectEnabled" />
                <span class="jj-sub inline-tip">
                  {{ form.collectEnabled ? 'AI 作答后可进候选池，仍需人工核实定稿' : '只有人工录入能进库，AI 只答不存' }}
                </span>
              </el-form-item>
              <el-form-item label="最短答案">
                <el-input-number v-model="form.minAnswer" :min="5" :max="500" :step="5" />
                <span class="jj-sub inline-tip">字。太短的答案（「好的」「可以」）入库后没有复用价值。</span>
              </el-form-item>
              <el-form-item label="每日入库上限">
                <el-input-number v-model="form.dailyQuota" :min="0" :max="1000" :step="10" />
                <span class="jj-sub inline-tip">条。0 表示不限；上限只挡新增，不挡重复提问计数。</span>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>

      <div class="rule-side">
        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">规则试跑<span class="jj-card-title-sub">改完先验一句</span></div>
          </div>
          <div class="jj-card-body">
            <el-input
              v-model="testQuestion"
              type="textarea"
              :rows="3"
              placeholder="比如：今天天气怎么样"
              @keydown.enter.exact.prevent="runTest"
            />
            <el-button class="test-btn" type="primary" plain :loading="testing" @click="runTest">
              试跑这句
            </el-button>

            <div v-if="testResult" class="test-result" :class="testResult.verdict.inDomain ? 'in' : 'out'">
              <div class="tr-head">
                <el-icon><component :is="testResult.verdict.inDomain ? 'CircleCheck' : 'CircleClose'" /></el-icon>
                <span>{{ testResult.verdict.inDomain ? '属于收录范围' : '不在收录范围' }}</span>
              </div>
              <div class="tr-line">{{ testResult.explain }}</div>
              <div v-if="testResult.verdict.hits?.length" class="tr-words">
                <el-tag v-for="w in testResult.verdict.hits" :key="w" size="small" type="success" effect="plain">
                  {{ w }}
                </el-tag>
              </div>
              <div v-if="testResult.verdict.blocked?.length" class="tr-words">
                <el-tag v-for="w in testResult.verdict.blocked" :key="w" size="small" type="danger" effect="plain">
                  {{ w }}
                </el-tag>
              </div>
              <div class="tr-next">{{ testResult.next }}</div>
            </div>

            <div v-if="recentTests.length" class="recent">
              <div class="recent-head">最近试跑</div>
              <div
                v-for="(item, i) in recentTests"
                :key="i"
                class="recent-item"
                @click="testQuestion = item.question; runTest()"
              >
                <span class="dot" :class="item.verdict.inDomain ? 'in' : 'out'" />
                <span class="recent-text">{{ item.question }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">硬拦截<span class="jj-card-title-sub">不可修改</span></div>
          </div>
          <div class="jj-card-body">
            <div class="jj-sub">
              以下特征词优先于所有规则判定。带「走心机」三个字的越权请求同样是越权，
              所以这层不开放编辑，避免规则被改坏后连闸门一起失效。
            </div>
            <div class="injection-list">
              <el-tag v-for="w in injectionWords" :key="w" size="small" type="info" effect="plain">
                {{ w }}
              </el-tag>
            </div>
          </div>
        </div>

        <div class="jj-card">
          <div class="jj-card-head">
            <div class="jj-card-title">生效位置<span class="jj-card-title-sub">规则作用在哪</span></div>
          </div>
          <div class="jj-card-body">
            <ol class="flow">
              <li><b>问答调试台 / 小程序</b>：检索未命中时先判领域，范围外直接拒答，不调模型。</li>
              <li><b>兜底作答</b>：领域内才作答，答案带「未核实」声明，再过闸门决定是否入库。</li>
              <li><b>沉淀落库</b>：所有自动入库统一在服务端校验，绕过页面也改不了这个行为。</li>
            </ol>
            <div class="jj-sub field-tip">
              人工录入不受准入词限制——运营最清楚该收什么，只有自动沉淀才需要机器守门。
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

// 必须用 computed：刷新页面时权限是异步加载的，setup 阶段读一次会永久拿到空权限
const canEdit = computed(() => canWrite('kb'))

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const stats = ref({})
const injectionWords = ref([])
const defaults = ref(null)
const testQuestion = ref('')
const testResult = ref(null)
const recentTests = ref([])

const form = reactive({
  name: '走心机与数控加工',
  scope: '',
  includeText: '',
  excludeText: '',
  minHits: 1,
  policy: 'strict',
  collectEnabled: true,
  minAnswer: 20,
  dailyQuota: 50
})

const splitWords = (text) =>
  String(text || '')
    .split(/[\n,，、;；|]/)
    .map((s) => s.trim())
    .filter(Boolean)

const includeWords = computed(() => splitWords(form.includeText))
const excludeWords = computed(() => splitWords(form.excludeText))

const quotaUsed = computed(() => {
  const quota = Number(form.dailyQuota) || 0
  if (quota <= 0) return 0
  return (Number(stats.value.todayCollected) || 0) / quota
})

function applyRule(rule) {
  if (!rule) return
  form.name = rule.name || ''
  form.scope = rule.scope || ''
  form.includeText = (rule.include || []).join('\n')
  form.excludeText = (rule.exclude || []).join('\n')
  form.minHits = rule.minHits || 1
  form.policy = rule.policy || 'strict'
  form.collectEnabled = rule.collectEnabled !== false
  form.minAnswer = rule.minAnswer ?? 20
  form.dailyQuota = rule.dailyQuota ?? 50
}

async function load() {
  loading.value = true
  try {
    const data = await api.kb.rule()
    applyRule(data.rule)
    defaults.value = data.defaults
    injectionWords.value = data.injection || []
    stats.value = data.stats || {}
  } catch (e) {
    if (!String(e.message).includes('权限')) ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function restoreDefaults() {
  if (!defaults.value) return
  applyRule(defaults.value)
  ElMessage.info('已恢复默认词表，点「保存规则」后生效')
}

async function save() {
  if (!canEdit.value) return
  const include = includeWords.value
  if (!include.length) {
    ElMessage.warning('准入词表不能为空，否则任何人都进不了知识库')
    return
  }
  saving.value = true
  try {
    const data = await api.kb.saveRule({
      name: form.name,
      scope: form.scope,
      include,
      exclude: excludeWords.value,
      minHits: form.minHits,
      policy: form.policy,
      collectEnabled: form.collectEnabled,
      minAnswer: form.minAnswer,
      dailyQuota: form.dailyQuota
    })
    applyRule(data.rule)
    const refreshed = await api.kb.rule()
    stats.value = refreshed.stats || {}
    ElMessage.success('规则已保存，下一次提问立即生效')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function runTest() {
  const question = testQuestion.value.trim()
  if (!question) {
    ElMessage.warning('先输入一句要试跑的问题')
    return
  }
  testing.value = true
  try {
    const data = await api.kb.testRule(question)
    testResult.value = data
    recentTests.value = [
      { question, verdict: data.verdict },
      ...recentTests.value.filter((item) => item.question !== question)
    ].slice(0, 5)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    testing.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.tip-alert {
  margin-bottom: 16px;
}

.rule-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(340px, 1fr);
  gap: 16px;
  align-items: start;
}

@media (max-width: 1280px) {
  .rule-grid {
    grid-template-columns: 1fr;
  }
}

.rule-main,
.rule-side {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.word-count {
  font-size: 12px;
  color: var(--jj-text-sub);
  font-variant-numeric: tabular-nums;
}

.word-area :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.7;
}

.inline-tip {
  margin-left: 10px;
}

.test-btn {
  width: 100%;
  margin-top: 10px;
}

.test-result {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: var(--jj-radius-sm);
  border: 1px solid var(--jj-line);
  background: #fbfcfe;
}

.test-result.in {
  border-color: rgba(24, 160, 88, 0.28);
  background: rgba(24, 160, 88, 0.05);
}

.test-result.out {
  border-color: rgba(232, 114, 44, 0.3);
  background: rgba(232, 114, 44, 0.05);
}

.tr-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 650;
  font-size: 13px;
  margin-bottom: 6px;
}

.test-result.in .tr-head {
  color: #188a50;
}

.test-result.out .tr-head {
  color: var(--jj-orange-dark);
}

.tr-line {
  font-size: 13px;
  color: var(--jj-text);
  line-height: 1.6;
}

.tr-words {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.tr-next {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--jj-line);
  font-size: 12.5px;
  color: var(--jj-text-sub);
  line-height: 1.6;
}

.recent {
  margin-top: 16px;
  border-top: 1px solid var(--jj-line-soft);
  padding-top: 12px;
}

.recent-head {
  font-size: 12px;
  color: var(--jj-text-mute);
  margin-bottom: 8px;
}

.recent-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  cursor: pointer;
  font-size: 12.5px;
  color: var(--jj-text-sub);
}

.recent-item:hover {
  color: var(--jj-text);
}

.recent-item .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex: none;
}

.recent-item .dot.in {
  background: #18a058;
}

.recent-item .dot.out {
  background: var(--jj-orange);
}

.recent-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.injection-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.flow {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.85;
  color: var(--jj-text);
}

.flow li {
  margin-bottom: 6px;
}

.flow b {
  font-weight: 650;
}
</style>
