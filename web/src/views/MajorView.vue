<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">专业管理</h2>
        <p class="jj-page-desc">
          专业是内容分类的第一层，小程序端按专业分组展示课程。删除专业前需先清空其下课程。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建专业
        </el-button>
      </div>
    </div>

    <div class="jj-stats">
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Collection /></el-icon>专业总数</div>
        <div class="jj-stat-value">{{ list.length }}<span class="jj-stat-unit">个</span></div>
        <div class="jj-stat-foot">已删除的不计入</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><CircleCheck /></el-icon>启用中</div>
        <div class="jj-stat-value">{{ enabledCount }}<span class="jj-stat-unit">个</span></div>
        <div class="jj-stat-foot">停用后小程序端不展示</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><CircleClose /></el-icon>已停用</div>
        <div class="jj-stat-value">{{ list.length - enabledCount }}<span class="jj-stat-unit">个</span></div>
        <div class="jj-stat-foot">保留数据但对外隐藏</div>
      </div>
      <div class="jj-stat warn">
        <div class="jj-stat-label"><el-icon><Reading /></el-icon>课程总数</div>
        <div class="jj-stat-value">{{ courseTotal }}<span class="jj-stat-unit">门</span></div>
        <div class="jj-stat-foot">所有专业合计</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">专业列表</div>
        <div class="head-tools">
          <el-input
            v-model="keyword"
            placeholder="搜索名称或编码"
            clearable
            style="width: 220px"
            @input="load"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button text :loading="loading" @click="load">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column label="专业" min-width="220">
          <template #default="{ row }">
            <div class="major-cell">
              <span class="major-badge">{{ row.icon || row.name.slice(0, 1) }}</span>
              <div>
                <div class="major-name">{{ row.name }}</div>
                <div class="jj-sub">编码：{{ row.code || '未设置' }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="summary" label="简介" min-width="240">
          <template #default="{ row }">
            <span class="jj-sub">{{ row.summary || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="课程数" width="100" align="right">
          <template #default="{ row }">
            <span class="count-num">{{ row.courseCount }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="jj-tag" :class="row.status === 1 ? 'on' : 'off'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="排序" width="120">
          <template #default="{ $index }">
            <div v-if="canEdit" class="sort-btns">
              <el-button text :disabled="$index === 0" @click="move($index, -1)">
                <el-icon><Top /></el-icon>
              </el-button>
              <el-button text :disabled="$index === list.length - 1" @click="move($index, 1)">
                <el-icon><Bottom /></el-icon>
              </el-button>
            </div>
            <span v-else class="jj-sub">{{ $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <template v-if="canEdit">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link type="primary" @click="$router.push(`/courses?majorId=${row.id}`)">
                课程
              </el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
            <span v-else class="jj-sub">只读</span>
          </template>
        </el-table-column>
        <template #empty>
          <div class="jj-empty">暂无专业，先新建一个</div>
        </template>
      </el-table>
    </div>

    <el-drawer
      v-model="drawer"
      :title="form.id ? '编辑专业' : '新建专业'"
      size="460px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="专业名称" prop="name">
          <el-input v-model="form.name" placeholder="如：走心机" maxlength="32" show-word-limit />
        </el-form-item>
        <el-form-item label="专业编码" prop="code">
          <el-input v-model="form.code" placeholder="英文标识，如 zouxinji，可留空" maxlength="32" />
          <div class="jj-sub field-tip">给小程序端做稳定标识，建议一次定好不要改</div>
        </el-form-item>
        <el-form-item label="图标文字" prop="icon">
          <el-input v-model="form.icon" placeholder="单个汉字，如 走 / 车 / 铣" maxlength="2" />
          <div class="jj-sub field-tip">列表与小程序端展示的圆形徽章文字</div>
        </el-form-item>
        <el-form-item label="简介" prop="summary">
          <el-input
            v-model="form.summary"
            type="textarea"
            :rows="3"
            maxlength="255"
            show-word-limit
            placeholder="一句话说明这个专业学什么"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
          <span class="jj-sub switch-tip">{{ form.status === 1 ? '启用' : '停用' }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawer = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
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
const canEdit = computed(() => canWrite('majors'))

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const keyword = ref('')
const drawer = ref(false)
const formRef = ref(null)

const form = reactive({ id: 0, name: '', code: '', icon: '', summary: '', status: 1 })

const rules = {
  name: [{ required: true, message: '请输入专业名称', trigger: 'blur' }]
}

const enabledCount = computed(() => list.value.filter((i) => i.status === 1).length)
const courseTotal = computed(() => list.value.reduce((sum, i) => sum + (i.courseCount || 0), 0))

async function load() {
  loading.value = true
  try {
    const data = await api.majors.list({ keyword: keyword.value || undefined })
    list.value = data.list || []
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { id: 0, name: '', code: '', icon: '', summary: '', status: 1 })
  formRef.value?.clearValidate()
}

function openCreate() {
  resetForm()
  drawer.value = true
}

function openEdit(row) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    code: row.code,
    icon: row.icon,
    summary: row.summary,
    status: row.status
  })
  formRef.value?.clearValidate()
  drawer.value = true
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await api.majors.save({ ...form })
    ElMessage.success(form.id ? '已保存' : '已创建')
    drawer.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function move(index, delta) {
  const target = index + delta
  if (target < 0 || target >= list.value.length) return
  const arr = [...list.value]
  ;[arr[index], arr[target]] = [arr[target], arr[index]]
  list.value = arr
  try {
    await api.majors.sort(arr.map((i) => i.id))
  } catch (e) {
    ElMessage.error(e.message)
    load()
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      row.courseCount > 0
        ? `「${row.name}」下还有 ${row.courseCount} 门课程，删除专业会被拒绝。确认继续？`
        : `确认删除专业「${row.name}」？删除后不可恢复。`,
      '删除专业',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.majors.remove(row.id)
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
}

.sort-btns {
  display: flex;
  align-items: center;
  gap: 0;
}

.sort-btns :deep(.el-button + .el-button) {
  margin-left: 4px;
}

.major-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.major-badge {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(135deg, #ff8a3d, #ff6b1a);
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.major-name {
  font-weight: 600;
  font-size: 14px;
}

.count-num {
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  color: var(--jj-orange);
}

.jj-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
}

.jj-tag.on {
  background: rgba(22, 163, 74, 0.12);
  color: #16a34a;
}

.jj-tag.off {
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
}

.field-tip {
  margin-top: 4px;
}

.switch-tip {
  margin-left: 10px;
}
</style>
