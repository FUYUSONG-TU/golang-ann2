<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">
          小程序管理
          <el-tag size="small" type="info" effect="plain">骨架阶段</el-tag>
        </h2>
        <p class="jj-page-desc">
          小程序端尚未开发。这里先固定模块清单与接口契约，等小程序开始做时，按模块逐个补齐即可。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button @click="load">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
        <el-button type="primary" @click="$router.push('/settings')">
          <el-icon><Setting /></el-icon>填写 AppID
        </el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon class="tip">
      <template #title>
        接口统一前缀 <b class="jj-mono">{{ data.apiBase }}</b>，后台账号体系与小程序用户体系相互独立，
        小程序用户表建议在接入时单独建表。
      </template>
    </el-alert>

    <div class="jj-stats">
      <div class="jj-stat" :class="data.connected ? 'ok' : 'warn'">
        <div class="jj-stat-label"><el-icon><Connection /></el-icon>接入状态</div>
        <div class="jj-stat-value small">{{ data.connected ? '已配置' : '未接入' }}</div>
        <div class="jj-stat-foot">{{ data.connected ? '已填写 AppID' : '等待小程序端开发' }}</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Iphone /></el-icon>小程序名称</div>
        <div class="jj-stat-value small">{{ data.name || '未命名' }}</div>
        <div class="jj-stat-foot">可在系统设置中修改</div>
      </div>
      <div class="jj-stat ok">
        <div class="jj-stat-label"><el-icon><CircleCheck /></el-icon>已就绪模块</div>
        <div class="jj-stat-value">{{ data.readyCount }}<span class="jj-stat-unit">/ {{ data.totalCount }}</span></div>
        <div class="jj-stat-foot">其余为规划中</div>
      </div>
      <div class="jj-stat">
        <div class="jj-stat-label"><el-icon><Postcard /></el-icon>AppID</div>
        <div class="jj-stat-value small mono">{{ data.appId || '未填写' }}</div>
        <div class="jj-stat-foot">系统设置 → 小程序配置</div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">功能模块清单</div>
        <span class="jj-sub">共 {{ data.totalCount }} 个模块</span>
      </div>
      <div class="jj-card-body">
        <div class="jj-modules">
          <div v-for="m in data.modules" :key="m.key" class="jj-module">
            <div class="jj-module-head">
              <span class="jj-module-name">{{ m.name }}</span>
              <el-tag size="small" :type="m.status === 'ready' ? 'success' : 'info'" effect="plain">
                {{ m.status === 'ready' ? '已就绪' : '待接入' }}
              </el-tag>
            </div>
            <div class="jj-module-desc">{{ m.desc }}</div>
            <span class="jj-module-api">{{ m.api }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="jj-card">
      <div class="jj-card-head">
        <div class="jj-card-title">接入路线</div>
        <span class="jj-sub">建议按顺序推进</span>
      </div>
      <div class="jj-card-body">
        <el-steps :active="1" align-center finish-status="success">
          <el-step title="后台骨架" description="账号、角色、日志、配置已完成" />
          <el-step title="小程序端开发" description="微信授权、首页、内容与预约" />
          <el-step title="接口对接" description="按模块补齐 /api/mp/* 实现" />
          <el-step title="上线运营" description="发布审核、数据看板与推送" />
        </el-steps>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const reloadTick = inject('jjReloadTick', ref(0))
const data = reactive({
  connected: false,
  name: '',
  appId: '',
  apiBase: '/api/mp',
  readyCount: 0,
  totalCount: 0,
  modules: []
})

async function load() {
  try {
    const res = await api.mp()
    Object.assign(data, res)
  } catch (e) {
    ElMessage.error(e.message)
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.tip {
  margin-bottom: 14px;
}

.jj-stat-value.small {
  font-size: 18px;
  font-weight: 600;
  margin-top: 10px;
}

.jj-stat-value.mono {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 15px;
}
</style>
