<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">角色权限</h2>
        <p class="jj-page-desc">
          角色决定账号能看到哪些菜单、能做哪些操作。内置角色不可删除，自定义角色可自由组合权限。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建角色
        </el-button>
      </div>
    </div>

    <div class="jj-card">
      <el-table :data="roles" v-loading="loading" style="width: 100%">
        <el-table-column label="角色" min-width="200">
          <template #default="{ row }">
            <div class="role-name">
              {{ row.name }}
              <el-tag v-if="row.builtin" size="small" effect="plain" type="info">内置</el-tag>
            </div>
            <div class="jj-sub">{{ row.code }}</div>
          </template>
        </el-table-column>

        <el-table-column label="说明" prop="description" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="jj-sub">{{ row.description || '—' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="权限" min-width="260">
          <template #default="{ row }">
            <template v-if="row.permissions === '*'">
              <el-tag size="small" type="danger" effect="plain">全部权限</el-tag>
            </template>
            <template v-else>
              <el-tag
                v-for="code in splitPerms(row.permissions).slice(0, 5)"
                :key="code"
                size="small"
                effect="plain"
                class="perm-tag"
              >
                {{ permName(code) }}
              </el-tag>
              <span v-if="splitPerms(row.permissions).length > 5" class="jj-sub">
                +{{ splitPerms(row.permissions).length - 5 }}
              </span>
              <span v-if="!splitPerms(row.permissions).length" class="jj-sub">未分配</span>
            </template>
          </template>
        </el-table-column>

        <el-table-column label="账号数" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" :type="row.userCount ? 'warning' : 'info'">
              {{ row.userCount }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <template v-if="canEdit">
              <el-button link type="primary" @click="openEdit(row)">配置</el-button>
              <el-button link type="danger" :disabled="row.builtin" @click="remove(row)">删除</el-button>
            </template>
            <el-button v-else link type="primary" @click="openEdit(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">权限说明</div>
        <span class="jj-sub">共 {{ permissionCount }} 项权限，分 {{ groups.length }} 组</span>
      </div>
      <div class="jj-card-body">
        <div class="jj-perm-grid">
          <div v-for="group in groups" :key="group.group" class="jj-perm-group">
            <div class="jj-perm-group-title">{{ group.group }}</div>
            <div v-for="item in group.items" :key="item.code" class="jj-perm-item">
              <el-tag size="small" effect="plain" type="info">{{ item.code }}</el-tag>
              <div>
                <div class="name">{{ item.name }}</div>
                <div class="desc">{{ item.desc }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '配置角色' : '新建角色'"
      width="680px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="88px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="form.name" placeholder="如：内容编辑" :disabled="readonly" />
        </el-form-item>
        <el-form-item label="角色编码" prop="code">
          <el-input
            v-model="form.code"
            placeholder="英文标识，如 editor"
            :disabled="readonly || !!form.id"
          />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="2" :disabled="readonly" />
        </el-form-item>

        <el-form-item label="权限">
          <el-alert
            v-if="form.code === 'super'"
            type="warning"
            :closable="false"
            show-icon
            title="超级管理员固定拥有全部权限，不可修改。"
          />
          <div v-else class="perm-edit">
            <div v-for="group in groups" :key="group.group" class="perm-block">
              <div class="perm-block-head">
                <el-checkbox
                  :model-value="allChecked(group)"
                  :indeterminate="someChecked(group)"
                  :disabled="readonly"
                  @change="(v) => toggleGroup(group, v)"
                >
                  {{ group.group }}
                </el-checkbox>
              </div>
              <div class="perm-checks">
                <el-checkbox
                  v-for="item in group.items"
                  :key="item.code"
                  v-model="form.permissions"
                  :value="item.code"
                  :disabled="readonly"
                >
                  {{ item.name }}
                </el-checkbox>
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ readonly ? '关闭' : '取消' }}</el-button>
        <el-button v-if="!readonly" type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = computed(() => canWrite('roles'))

const loading = ref(false)
const saving = ref(false)
const roles = ref([])
const groups = ref([])

const dialogVisible = ref(false)
const readonly = ref(false)
const formRef = ref(null)
const form = reactive({
  id: 0,
  code: '',
  name: '',
  description: '',
  permissions: [],
  sort: 0
})

const rules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入角色编码', trigger: 'blur' },
    { pattern: /^[a-z][a-z0-9_]{1,31}$/, message: '小写字母开头，可含数字与下划线', trigger: 'blur' }
  ]
}

const permissionCount = computed(() =>
  groups.value.reduce((sum, g) => sum + g.items.length, 0)
)

const permMap = computed(() => {
  const map = {}
  groups.value.forEach((g) => g.items.forEach((i) => (map[i.code] = i.name)))
  return map
})

function permName(code) {
  return permMap.value[code] || code
}

function splitPerms(raw) {
  if (!raw) return []
  return raw.split(',').map((s) => s.trim()).filter(Boolean)
}

function allChecked(group) {
  return group.items.every((i) => form.permissions.includes(i.code))
}

function someChecked(group) {
  const n = group.items.filter((i) => form.permissions.includes(i.code)).length
  return n > 0 && n < group.items.length
}

function toggleGroup(group, checked) {
  const codes = group.items.map((i) => i.code)
  if (checked) {
    const set = new Set([...form.permissions, ...codes])
    form.permissions = Array.from(set)
  } else {
    form.permissions = form.permissions.filter((c) => !codes.includes(c))
  }
}

async function load() {
  loading.value = true
  try {
    const [roleData, permData] = await Promise.all([api.roles.list(), api.roles.permissions()])
    roles.value = roleData.list || []
    groups.value = permData.groups || []
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  readonly.value = false
  Object.assign(form, { id: 0, code: '', name: '', description: '', permissions: [], sort: roles.value.length + 1 })
  dialogVisible.value = true
}

function openEdit(row) {
  readonly.value = !canEdit.value
  Object.assign(form, {
    id: row.id,
    code: row.code,
    name: row.name,
    description: row.description,
    permissions: splitPerms(row.permissions),
    sort: row.sort
  })
  dialogVisible.value = true
}

async function submit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await api.roles.save({
        id: form.id,
        code: form.code,
        name: form.name,
        description: form.description,
        permissions: form.permissions,
        sort: form.sort
      })
      ElMessage.success('已保存')
      dialogVisible.value = false
      load()
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      saving.value = false
    }
  })
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(`确认删除角色「${row.name}」？`, '删除角色', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.roles.remove(row.id)
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
.role-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13.5px;
  font-weight: 500;
}

.perm-tag {
  margin: 0 4px 4px 0;
}

.perm-edit {
  width: 100%;
}

.perm-block {
  border: 1px solid var(--jj-line);
  border-radius: 6px;
  padding: 8px 12px 4px;
  margin-bottom: 8px;
}

.perm-block-head {
  border-bottom: 1px dashed #eef1f5;
  padding-bottom: 4px;
  margin-bottom: 6px;
}

.perm-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
}
</style>
