<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">账号管理</h2>
        <p class="jj-page-desc">维护后台登录账号、所属角色与启用状态。当前共 {{ total }} 个账号。</p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建账号
        </el-button>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-toolbar">
        <el-input
          v-model="query.keyword"
          placeholder="搜索账号 / 昵称 / 手机号"
          clearable
          style="width: 230px"
          @keyup.enter="search"
          @clear="search"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.role" placeholder="全部角色" clearable style="width: 150px" @change="search">
          <el-option v-for="r in roles" :key="r.code" :label="r.name" :value="r.code" />
        </el-select>
        <el-select v-model="query.status" style="width: 130px" @change="search">
          <el-option label="全部状态" :value="-1" />
          <el-option label="已启用" :value="1" />
          <el-option label="已禁用" :value="0" />
        </el-select>
        <el-button type="primary" plain @click="search">查询</el-button>
        <el-button @click="reset">重置</el-button>
        <span class="grow" />
        <el-button text :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%" row-key="id">
        <el-table-column label="账号" min-width="180">
          <template #default="{ row }">
            <div class="cell-user">
              <el-avatar :size="30" class="cell-avatar">{{ (row.nickname || row.username).slice(0, 1) }}</el-avatar>
              <div>
                <div class="cell-name">
                  {{ row.nickname || row.username }}
                  <el-tag v-if="row.id === auth.user?.id" size="small" effect="plain" type="warning">本人</el-tag>
                </div>
                <div class="jj-sub">{{ row.username }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="角色" width="130">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.roleName || row.roleCode }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="联系方式" min-width="180">
          <template #default="{ row }">
            <div class="jj-sub">{{ row.phone || '未填写手机号' }}</div>
            <div class="jj-sub">{{ row.email || '未填写邮箱' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-switch
              :model-value="row.status === 1"
              :disabled="!canEdit || row.id === auth.user?.id"
              @change="(v) => toggle(row, v)"
            />
          </template>
        </el-table-column>

        <el-table-column label="最后登录" width="190">
          <template #default="{ row }">
            <template v-if="row.lastLoginAt">
              <div>{{ fmt(row.lastLoginAt) }}</div>
              <div class="jj-sub">{{ row.lastLoginIp }}</div>
            </template>
            <span v-else class="jj-sub">从未登录</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="120">
          <template #default="{ row }">
            <span class="jj-sub">{{ fmt(row.createdAt, true) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <template v-if="canEdit">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link type="primary" @click="openReset(row)">重置密码</el-button>
              <el-button
                link
                type="danger"
                :disabled="row.id === auth.user?.id"
                @click="remove(row)"
              >
                删除
              </el-button>
            </template>
            <span v-else class="jj-sub">只读</span>
          </template>
        </el-table-column>

        <template #empty>
          <div class="jj-empty">没有符合条件的账号</div>
        </template>
      </el-table>

      <div class="jj-pager">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.size"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="load"
          @current-change="load"
        />
      </div>
    </div>

    <!-- 新建 / 编辑 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editing ? '编辑账号' : '新建账号'"
      width="520px"
      :close-on-click-modal="false"
      @closed="clearValidate"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="88px">
        <el-form-item label="账号" prop="username">
          <el-input v-model="form.username" :disabled="editing" placeholder="3-32 位，登录用" />
        </el-form-item>
        <el-form-item v-if="!editing" label="初始密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" placeholder="显示名称" />
        </el-form-item>
        <el-form-item label="角色" prop="roleCode">
          <el-select v-model="form.roleCode" style="width: 100%" :disabled="editing && form.id === auth.user?.id">
            <el-option v-for="r in roles" :key="r.code" :label="r.name" :value="r.code" />
          </el-select>
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="form.phone" placeholder="选填" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="选填" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="选填" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码 -->
    <el-dialog v-model="resetVisible" title="重置密码" width="420px" :close-on-click-modal="false">
      <p class="jj-hint">
        正在为账号 <b>{{ resetTarget?.username }}</b> 重置密码，重置后对方登录时需重新设置。
      </p>
      <el-form ref="resetFormRef" :model="resetForm" :rules="resetRules" label-width="80px">
        <el-form-item label="新密码" prop="password">
          <el-input v-model="resetForm.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitReset">确认重置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { auth, canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = computed(() => canWrite('users'))

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const roles = ref([])
const total = ref(0)

const query = reactive({ keyword: '', role: '', status: -1, page: 1, size: 10 })

const dialogVisible = ref(false)
const editing = ref(false)
const formRef = ref(null)
const form = reactive({
  id: 0,
  username: '',
  password: '',
  nickname: '',
  roleCode: 'operator',
  phone: '',
  email: '',
  remark: ''
})

const rules = {
  username: [
    { required: true, message: '请输入账号', trigger: 'blur' },
    { min: 3, max: 32, message: '长度 3-32 位', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入初始密码', trigger: 'blur' },
    { min: 6, message: '至少 6 位', trigger: 'blur' }
  ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  roleCode: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const resetVisible = ref(false)
const resetFormRef = ref(null)
const resetTarget = ref(null)
const resetForm = reactive({ password: '' })
const resetRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '至少 6 位', trigger: 'blur' }
  ]
}

function fmt(v, dateOnly = false) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  if (dateOnly) return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await api.users.list({
      page: query.page,
      size: query.size,
      keyword: query.keyword,
      role: query.role,
      status: query.status
    })
    list.value = data.list || []
    total.value = data.total || 0
    if (data.roles) roles.value = data.roles
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function search() {
  query.page = 1
  load()
}

function reset() {
  query.keyword = ''
  query.role = ''
  query.status = -1
  search()
}

function openCreate() {
  editing.value = false
  Object.assign(form, {
    id: 0,
    username: '',
    password: '',
    nickname: '',
    roleCode: roles.value[1]?.code || roles.value[0]?.code || 'operator',
    phone: '',
    email: '',
    remark: ''
  })
  dialogVisible.value = true
}

function openEdit(row) {
  editing.value = true
  Object.assign(form, {
    id: row.id,
    username: row.username,
    password: '',
    nickname: row.nickname,
    roleCode: row.roleCode,
    phone: row.phone,
    email: row.email,
    remark: row.remark
  })
  dialogVisible.value = true
}

function clearValidate() {
  formRef.value?.clearValidate()
}

async function submit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editing.value) {
        await api.users.update({
          id: form.id,
          nickname: form.nickname,
          roleCode: form.roleCode,
          phone: form.phone,
          email: form.email,
          remark: form.remark
        })
        ElMessage.success('已保存')
      } else {
        await api.users.create({
          username: form.username,
          password: form.password,
          nickname: form.nickname,
          roleCode: form.roleCode,
          phone: form.phone,
          email: form.email,
          remark: form.remark
        })
        ElMessage.success('账号已创建')
      }
      dialogVisible.value = false
      load()
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      saving.value = false
    }
  })
}

function openReset(row) {
  resetTarget.value = row
  resetForm.password = ''
  resetVisible.value = true
}

async function submitReset() {
  if (!resetFormRef.value) return
  await resetFormRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await api.users.resetPassword(resetTarget.value.id, resetForm.password)
      ElMessage.success('密码已重置')
      resetVisible.value = false
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      saving.value = false
    }
  })
}

async function toggle(row, value) {
  try {
    await api.users.toggle(row.id, value ? 1 : 0)
    row.status = value ? 1 : 0
    ElMessage.success(value ? '账号已启用' : '账号已禁用')
  } catch (e) {
    ElMessage.error(e.message)
    load()
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除账号「${row.username}」？该操作不可恢复，历史日志会保留。`,
      '删除账号',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.users.remove(row.id)
    ElMessage.success('已删除')
    if (list.value.length === 1 && query.page > 1) query.page -= 1
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.cell-user {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cell-avatar {
  background: linear-gradient(135deg, #f3a06a, #e8722c);
  font-size: 13px;
  flex-shrink: 0;
}

.cell-name {
  font-size: 13.5px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
  line-height: 1.4;
}
</style>
