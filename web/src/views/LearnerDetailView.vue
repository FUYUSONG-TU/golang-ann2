<template>
  <div class="jj-page" v-loading="loading">
    <div class="jj-page-head">
      <div>
        <div class="back-line">
          <el-button link @click="$router.push('/learners')">
            <el-icon><ArrowLeft /></el-icon>返回学员列表
          </el-button>
        </div>
        <h2 class="jj-page-title">
          {{ user.nickname || '学员详情' }}
          <span v-if="user.demo" class="demo-tag">演示数据</span>
          <span v-if="user.status === 0" class="jj-tag offline">已禁用</span>
        </h2>
        <p class="jj-page-desc">
          {{ user.majorName || '未选择专业' }} · {{ user.levelText || '入门' }} ·
          openId {{ user.openId || '-' }}
        </p>
      </div>
      <div class="jj-page-actions">
        <template v-if="canEdit">
          <el-button v-if="user.status === 1" type="warning" plain @click="toggleStatus(0)">禁用账号</el-button>
          <el-button v-else type="success" plain @click="toggleStatus(1)">恢复账号</el-button>
          <el-button type="danger" plain @click="removeUser">删除学员</el-button>
        </template>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Reading /></el-icon>学习课程</div>
        <div class="jj-stat-value">{{ stats.courseCount }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">已学完 {{ stats.finishedCount }} 门 · 完成率 {{ stats.finishRate }}%</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><TrendCharts /></el-icon>平均进度</div>
        <div class="jj-stat-value">{{ stats.avgProgress }}<span class="jj-stat-unit">%</span></div>
        <div class="jj-stat-foot">按全部门课程进度平均</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Timer /></el-icon>累计学时</div>
        <div class="jj-stat-value">{{ stats.studyHours }}<span class="jj-stat-unit">小时</span></div>
        <div class="jj-stat-foot">小程序上报的学习时长</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Clock /></el-icon>最近学习</div>
        <div class="jj-stat-value small">{{ fmtDate(stats.lastStudyAt) }}</div>
        <div class="jj-stat-foot">注册于 {{ fmtDate(user.createdAt) }}</div>
      </div>
    </div>

    <div class="detail-grid">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">
            学习记录
            <span class="jj-card-title-sub">共 {{ records.length }} 门课程有进度</span>
          </div>
        </div>
        <el-table :data="records" style="width: 100%">
          <el-table-column label="课程" min-width="200">
            <template #default="{ row }">
              <div class="course-name">{{ row.courseName }}</div>
              <div class="jj-sub">最近章节：{{ row.chapterTitle || '未细分到章节' }}</div>
            </template>
          </el-table-column>
          <el-table-column label="进度" width="170">
            <template #default="{ row }">
              <el-progress :percentage="row.progress" :stroke-width="8" :color="progressColor(row.progress)" />
            </template>
          </el-table-column>
          <el-table-column label="学习时长" width="100" align="right">
            <template #default="{ row }">
              <span class="num">{{ minutes(row.studySeconds) }}</span>
              <span class="jj-sub"> 分钟</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <span class="jj-tag" :class="row.status === 'finished' ? 'published' : 'pending'">
                {{ row.statusText }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="最近学习" width="150">
            <template #default="{ row }">
              <span class="jj-sub">{{ fmt(row.lastStudyAt) }}</span>
            </template>
          </el-table-column>
          <template #empty>
            <div class="jj-empty">该学员还没有学习记录。小程序上报进度后会自动出现在这里。</div>
          </template>
        </el-table>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">学员档案</div>
          <span class="jj-sub">{{ canEdit ? '可人工修正' : '只读' }}</span>
        </div>
        <div class="jj-card-body">
          <el-form :model="form" label-position="top" :disabled="!canEdit">
            <el-form-item label="昵称">
              <el-input v-model="form.nickname" maxlength="64" />
            </el-form-item>
            <el-form-item label="手机号">
              <el-input v-model="form.phone" maxlength="20" placeholder="选填" />
            </el-form-item>
            <el-form-item label="专业">
              <el-select v-model="form.majorId" clearable placeholder="未选择" style="width: 100%">
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
              <el-input v-model="form.remark" type="textarea" :rows="3" maxlength="200" show-word-limit />
            </el-form-item>
            <el-button v-if="canEdit" type="primary" :loading="saving" style="width: 100%" @click="save">
              保存档案
            </el-button>
          </el-form>

          <dl class="jj-kv info">
            <dt>最近活跃</dt>
            <dd>{{ fmtDate(user.lastActiveAt) }}</dd>
            <dt>登录次数</dt>
            <dd>{{ user.loginCount || 0 }} 次</dd>
            <dt>注册时间</dt>
            <dd>{{ fmtDate(user.createdAt) }}</dd>
          </dl>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const route = useRoute()
const router = useRouter()
const canEdit = canWrite('learners')

const loading = ref(false)
const saving = ref(false)
const user = ref({})
const records = ref([])
const majors = ref([])
const stats = ref({ courseCount: 0, finishedCount: 0, avgProgress: 0, studyHours: 0, finishRate: 0, lastStudyAt: null })
const form = reactive({ nickname: '', phone: '', majorId: null, level: 1, remark: '' })

const userId = Number(route.params.id)

async function load() {
  loading.value = true
  try {
    const data = await api.learners.detail(userId)
    user.value = data.user || {}
    records.value = data.records || []
    majors.value = data.majors || []
    stats.value = data.stats || stats.value
    Object.assign(form, {
      nickname: user.value.nickname || '',
      phone: user.value.phone || '',
      majorId: user.value.majorId || null,
      level: user.value.level || 1,
      remark: user.value.remark || ''
    })
  } catch (e) {
    ElMessage.error(e.message)
    router.replace('/learners')
  } finally {
    loading.value = false
  }
}

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

function fmtDate(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

function minutes(seconds) {
  return Math.round((seconds || 0) / 60)
}

function progressColor(v) {
  if (v >= 100) return '#16a34a'
  if (v >= 60) return '#2563eb'
  return '#e8722c'
}

async function save() {
  if (!form.nickname.trim()) {
    ElMessage.warning('昵称不能为空')
    return
  }
  saving.value = true
  try {
    await api.learners.save({ id: userId, ...form, status: user.value.status })
    ElMessage.success('已保存')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function toggleStatus(status) {
  try {
    await api.learners.status(userId, status)
    ElMessage.success(status === 1 ? '已恢复' : '已禁用')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function removeUser() {
  try {
    await ElMessageBox.confirm(
      `确认删除学员「${user.value.nickname}」？其 ${records.value.length} 条学习记录会一并删除。`,
      '删除学员',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.learners.remove(userId)
    ElMessage.success('已删除')
    router.push('/learners')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(load)
</script>

<style scoped>
.back-line {
  margin-bottom: 6px;
}

.detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) minmax(280px, 1fr);
  gap: 16px;
  align-items: start;
}

.course-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 2px;
}

.num {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
}

.demo-tag {
  padding: 1px 7px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
  background: rgba(148, 163, 184, 0.2);
  vertical-align: middle;
}

.jj-stat-value.small {
  font-size: 20px;
}

.jj-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
}

.jj-tag.published {
  background: rgba(22, 163, 74, 0.12);
  color: #16a34a;
}

.jj-tag.pending {
  background: rgba(245, 158, 11, 0.14);
  color: #d97706;
}

.jj-tag.offline {
  background: rgba(220, 38, 38, 0.12);
  color: #dc2626;
}

.jj-kv.info {
  margin-top: 18px;
  padding-top: 12px;
  border-top: 1px dashed var(--jj-line);
}

@media (max-width: 1180px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
