<template>
  <div class="jj-login">
    <div class="jj-login-panel">
      <div class="jj-login-brand">
        <div class="jj-login-logo">匠</div>
        <div class="jj-login-title">机匠学徒</div>
        <div class="jj-login-slogan">手艺传下去，数据管起来</div>
      </div>
      <ul class="jj-login-points">
        <li><el-icon><Cpu /></el-icon> Go + Vue3 一体化部署</li>
        <li><el-icon><DataLine /></el-icon> 运行状态实时可见</li>
        <li><el-icon><Lock /></el-icon> 密钥保护管理入口</li>
      </ul>
      <div class="jj-login-foot">Powered by 微信云托管</div>
    </div>

    <div class="jj-login-form-wrap">
      <div class="jj-login-form">
        <h2>管理员登录</h2>
        <p class="jj-login-hint">
          {{ probing ? '正在检测服务状态…' : auth.authRequired ? '请输入服务端配置的管理密钥' : '服务未启用访问控制，可直接进入' }}
        </p>

        <el-skeleton v-if="probing" :rows="2" animated />

        <template v-else-if="!auth.authRequired">
          <el-alert type="warning" :closable="false" show-icon
            title="当前未配置 ADMIN_TOKEN，任何人都能访问后台"
            description="建议到云托管「服务设置 → 环境变量」添加 ADMIN_TOKEN 后重新部署。" />
          <el-button type="primary" size="large" class="jj-login-btn" @click="go">
            进入控制台
          </el-button>
        </template>

        <template v-else>
          <el-input
            v-model="token"
            size="large"
            type="password"
            show-password
            placeholder="请输入 ADMIN_TOKEN"
            prefix-icon="Key"
            @keyup.enter="submit"
          />
          <el-button
            type="primary"
            size="large"
            class="jj-login-btn"
            :loading="loading"
            @click="submit"
          >
            验证并进入
          </el-button>
        </template>

        <div class="jj-login-tip">
          密钥在云托管控制台配置，名称 <code>ADMIN_TOKEN</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { auth, setToken } from '../store/auth'

const route = useRoute()
const router = useRouter()

const token = ref(auth.token || '')
const loading = ref(false)
const probing = ref(true)

onMounted(async () => {
  try {
    await api.probe()
    auth.authRequired = false
  } catch {
    auth.authRequired = true
  } finally {
    auth.checked = true
    probing.value = false
  }
})

function go() {
  const r = route.query.redirect
  router.replace(typeof r === 'string' && r ? r : '/')
}

async function submit() {
  const val = token.value.trim()
  if (!val) {
    ElMessage.warning('请输入管理密钥')
    return
  }
  loading.value = true
  try {
    setToken(val)
    await api.overview()
    auth.authRequired = true
    auth.checked = true
    ElMessage.success('验证通过')
    go()
  } catch (e) {
    setToken('')
    ElMessage.error(e.message || '密钥验证失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.jj-login {
  height: 100vh;
  display: flex;
  background: #f5f6f8;
}

/* 左侧品牌区 */
.jj-login-panel {
  width: 44%;
  max-width: 520px;
  background: radial-gradient(circle at 20% 15%, #2c3852 0%, #1d2531 55%, #161c26 100%);
  color: #fff;
  padding: 54px 48px;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.jj-login-panel::after {
  content: '';
  position: absolute;
  right: -120px;
  bottom: -120px;
  width: 320px;
  height: 320px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(232, 114, 44, 0.28), transparent 65%);
}

.jj-login-brand {
  position: relative;
  z-index: 1;
}

.jj-login-logo {
  width: 52px;
  height: 52px;
  border-radius: 13px;
  background: linear-gradient(135deg, #f08a4b, #e8722c);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  box-shadow: 0 8px 24px rgba(232, 114, 44, 0.4);
}

.jj-login-title {
  font-size: 30px;
  font-weight: 700;
  letter-spacing: 4px;
  margin-top: 22px;
}

.jj-login-slogan {
  color: #8b97a9;
  font-size: 14px;
  letter-spacing: 1px;
  margin-top: 8px;
}

.jj-login-points {
  list-style: none;
  padding: 0;
  margin: 44px 0 0;
  position: relative;
  z-index: 1;
  flex: 1;
}

.jj-login-points li {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #b3bccb;
  font-size: 13.5px;
  padding: 9px 0;
}

.jj-login-points li .el-icon {
  color: var(--jj-orange);
  font-size: 15px;
}

.jj-login-foot {
  color: #5c6675;
  font-size: 12px;
  position: relative;
  z-index: 1;
}

/* 右侧表单 */
.jj-login-form-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.jj-login-form {
  width: 100%;
  max-width: 360px;
}

.jj-login-form h2 {
  font-size: 22px;
  margin: 0 0 8px;
  color: var(--jj-ink);
}

.jj-login-hint {
  color: #8a94a6;
  font-size: 13px;
  margin: 0 0 20px;
  min-height: 20px;
}

.jj-login-btn {
  width: 100%;
  margin-top: 16px;
  letter-spacing: 2px;
}

.jj-login-tip {
  margin-top: 22px;
  font-size: 12px;
  color: #a5adba;
  text-align: center;
}

.jj-login-tip code {
  background: #eceef2;
  padding: 2px 6px;
  border-radius: 4px;
  color: #5c6675;
}

@media (max-width: 860px) {
  .jj-login-panel {
    display: none;
  }
}
</style>
