<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">个人中心</h2>
        <p class="jj-page-desc">查看自己的账号信息并修改登录密码。</p>
      </div>
    </div>

    <div class="profile-grid">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">账号信息</div>
          <el-tag size="small" effect="plain" :type="user?.status === 1 ? 'success' : 'danger'">
            {{ user?.status === 1 ? '已启用' : '已禁用' }}
          </el-tag>
        </div>
        <div class="jj-card-body">
          <div class="profile-head">
            <el-avatar :size="54" class="profile-avatar">{{ avatarText }}</el-avatar>
            <div>
              <div class="profile-name">{{ user?.nickname || user?.username }}</div>
              <div class="jj-sub">{{ auth.role?.name || '未分配角色' }}</div>
            </div>
          </div>

          <dl class="jj-kv profile-kv">
            <dt>登录账号</dt>
            <dd>{{ user?.username }}</dd>
            <dt>账号 ID</dt>
            <dd>#{{ user?.id }}</dd>
            <dt>手机号</dt>
            <dd>{{ user?.phone || '未填写' }}</dd>
            <dt>邮箱</dt>
            <dd>{{ user?.email || '未填写' }}</dd>
            <dt>最后登录</dt>
            <dd>{{ fmt(user?.lastLoginAt) }}<span v-if="user?.lastLoginIp" class="jj-sub"> · {{ user.lastLoginIp }}</span></dd>
            <dt>创建时间</dt>
            <dd>{{ fmt(user?.createdAt) }}</dd>
            <dt>拥有权限</dt>
            <dd>
              <template v-if="auth.permissions.includes('*')">
                <el-tag size="small" type="danger" effect="plain">全部权限</el-tag>
              </template>
              <template v-else>
                <el-tag
                  v-for="p in auth.permissions"
                  :key="p"
                  size="small"
                  effect="plain"
                  class="perm-tag"
                >
                  {{ p }}
                </el-tag>
                <span v-if="!auth.permissions.length" class="jj-sub">未分配</span>
              </template>
            </dd>
          </dl>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">修改密码</div>
        </div>
        <div class="jj-card-body">
          <el-alert
            v-if="user?.mustChangePwd"
            type="warning"
            :closable="false"
            show-icon
            title="当前仍在使用初始密码，建议立即修改。"
            class="mb"
          />
          <el-form ref="formRef" :model="form" :rules="rules" label-width="88px">
            <el-form-item label="原密码" prop="oldPassword">
              <el-input v-model="form.oldPassword" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="form.newPassword" type="password" show-password />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="form.confirmPassword" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="submit">保存并重新登录</el-button>
            </el-form-item>
          </el-form>
          <p class="jj-hint">
            密码使用 PBKDF2-SHA256 加盐存储，修改后当前登录状态会失效，需要用新密码重新登录。
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { auth, clearSession } from '../store/auth'

const router = useRouter()
const user = computed(() => auth.user)
const avatarText = computed(() => (user.value?.nickname || user.value?.username || '我').slice(0, 1))

const formRef = ref(null)
const saving = ref(false)
const form = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const rules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '至少 6 位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule, value, cb) => {
        if (value !== form.newPassword) cb(new Error('两次输入的密码不一致'))
        else cb()
      },
      trigger: 'blur'
    }
  ]
}

function fmt(v) {
  if (!v) return '-'
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function submit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await api.changePassword(form.oldPassword, form.newPassword)
      ElMessage.success('密码已更新，请重新登录')
      clearSession()
      router.replace('/login')
    } catch (e) {
      ElMessage.error(e.message)
    } finally {
      saving.value = false
    }
  })
}
</script>

<style scoped>
.profile-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.profile-grid .jj-card + .jj-card {
  margin-top: 0;
}

.profile-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px dashed #eef1f5;
}

.profile-avatar {
  background: linear-gradient(135deg, #f08a4b, #e8722c);
  font-size: 22px;
  font-weight: 600;
}

.profile-name {
  font-size: 17px;
  font-weight: 600;
  margin-bottom: 2px;
}

.profile-kv {
  grid-template-columns: 90px 1fr;
}

.perm-tag {
  margin: 0 4px 4px 0;
}

.mb {
  margin-bottom: 16px;
}

@media (max-width: 1000px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
