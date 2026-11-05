<template>
  <div class="jj-login">
    <aside class="jj-login-side">
      <div class="jj-login-brand">
        <div class="jj-login-mark">匠</div>
        <div>
          <div class="jj-login-brand-name">机匠学徒</div>
          <div class="jj-login-brand-sub">ADMIN CONSOLE</div>
        </div>
      </div>

      <div class="jj-login-copy">
        <h1>把手艺沉进系统里，<br />让每一次交付都有据可查。</h1>
        <p>
          账号、角色、日志、数据看板与小程序端接口骨架都已就位。<br />
          这里是机匠学徒的统一后台入口。
        </p>
      </div>

      <div class="jj-login-points">
        <span>· 多角色权限</span>
        <span>· 登录与操作留痕</span>
        <span>· 小程序接口预留</span>
      </div>
    </aside>

    <main class="jj-login-main">
      <div class="jj-login-form">
        <h2>后台登录</h2>
        <p class="desc">请使用管理员账号登录，未授权访问将被记录。</p>

        <el-alert
          v-if="errorMsg"
          :title="errorMsg"
          type="error"
          :closable="false"
          show-icon
          class="mb"
        />

        <el-form ref="formRef" :model="form" :rules="rules" size="large" @submit.prevent>
          <el-form-item prop="username">
            <el-input v-model="form.username" placeholder="账号" clearable @keyup.enter="submit">
              <template #prefix><el-icon><User /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码"
              show-password
              @keyup.enter="submit"
            >
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>

          <div class="row">
            <el-checkbox v-model="form.remember">7 天内保持登录</el-checkbox>
            <span class="forget" @click="showHelp">登录遇到问题？</span>
          </div>

          <el-button
            type="primary"
            size="large"
            class="submit"
            :loading="loading"
            @click="submit"
          >
            登录
          </el-button>
        </el-form>

        <div class="jj-login-tip">
          首次部署内置账号 <b>admin</b>，初始密码 <b>admin123</b>。登录后请立即到「个人中心」修改，
          或通过环境变量 <code>ADMIN_INIT_PASSWORD</code> 指定初始密码。
        </div>

        <div class="jj-login-foot">{{ copyright }}</div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { auth, setSession } from '../store/auth'

const route = useRoute()
const router = useRouter()

const formRef = ref(null)
const loading = ref(false)
const errorMsg = ref('')
const copyright = ref('© 2026 机匠学徒')

const form = reactive({ username: '', password: '', remember: true })
const rules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function submit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    errorMsg.value = ''
    try {
      const data = await api.login(form.username.trim(), form.password, form.remember)
      setSession({
        token: data.token,
        user: data.user,
        role: data.role,
        permissions: data.permissions
      })
      if (data.user?.mustChangePwd) {
        ElMessage.warning('当前账号仍在使用初始密码，请尽快到「个人中心」修改')
      } else {
        ElMessage.success('登录成功')
      }
      const redirect = route.query.redirect
      router.replace(typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/')
    } catch (e) {
      errorMsg.value = e.message || '登录失败'
    } finally {
      loading.value = false
    }
  })
}

function showHelp() {
  ElMessageBox.alert(
    '1. 账号或密码错误会记录在登录日志中；\n2. 忘记密码请联系超级管理员在「账号管理」中重置；\n3. 若刚部署，使用初始账号 admin / admin123 登录。',
    '登录帮助',
    { confirmButtonText: '知道了' }
  )
}

onMounted(() => {
  if (auth.token) router.replace('/')
})
</script>

<style scoped>
.mb {
  margin-bottom: 16px;
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: -4px 0 18px;
}

.forget {
  font-size: 12.5px;
  color: #9aa4b2;
  cursor: pointer;
}

.forget:hover {
  color: var(--jj-orange);
}

.submit {
  width: 100%;
  letter-spacing: 4px;
}

code {
  background: rgba(154, 90, 44, 0.1);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11.5px;
}
</style>
