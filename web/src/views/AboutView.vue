<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h1 class="jj-page-title">系统信息</h1>
        <p class="jj-page-desc">技术栈、接口清单与部署说明</p>
      </div>
    </div>

    <div class="jj-grid">
      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">项目概况</div>
        </div>
        <div class="jj-card-body">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="项目名称">机匠学徒 · 后台管理系统</el-descriptions-item>
            <el-descriptions-item label="前端">Vue 3 + Vite + Element Plus</el-descriptions-item>
            <el-descriptions-item label="后端">Go net/http + GORM</el-descriptions-item>
            <el-descriptions-item label="数据库">MySQL 5.7（golang_demo.Counters）</el-descriptions-item>
            <el-descriptions-item label="部署">微信云托管，单容器，前端嵌入 Go 二进制</el-descriptions-item>
            <el-descriptions-item label="访问入口">
              <span class="jj-mono">/</span> 后台首页 ·
              <span class="jj-mono">/login</span> 登录
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </div>

      <div class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">接口清单</div>
        </div>
        <div class="jj-card-body">
          <el-table :data="apis" size="small" border>
            <el-table-column prop="method" label="方法" width="80">
              <template #default="{ row }">
                <span class="jj-method" :class="row.method.toLowerCase()">{{ row.method }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="path" label="路径" min-width="200">
              <template #default="{ row }">
                <span class="jj-mono">{{ row.path }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="desc" label="说明" min-width="120" />
          </el-table>
        </div>
      </div>
    </div>

    <div class="jj-card" style="margin-top: 16px">
      <div class="jj-card-head">
        <div class="jj-card-title">部署与安全</div>
      </div>
      <div class="jj-card-body">
        <el-alert
          v-if="!authRequired"
          type="warning"
          :closable="false"
          show-icon
          title="服务未启用访问控制"
          description="在云托管「服务设置 → 环境变量」添加 ADMIN_TOKEN 并重新部署后，所有 /api/admin 接口将要求携带 X-Admin-Token 请求头。"
          style="margin-bottom: 14px"
        />
        <el-alert
          v-else
          type="success"
          :closable="false"
          show-icon
          title="访问控制已启用"
          description="接口已要求 X-Admin-Token 请求头，密钥在服务端环境变量 ADMIN_TOKEN 中配置。"
          style="margin-bottom: 14px"
        />
        <ul class="jj-tips">
          <li>前端构建产物通过 <code>go:embed</code> 嵌入二进制，容器内无需额外静态目录</li>
          <li>SPA 路由使用 history 模式，非 API 路径统一回退到 <code>index.html</code></li>
          <li>数据库连接信息全部来自环境变量，不写入代码与仓库</li>
          <li>建议在合规模块下操作数据，删除记录不可恢复</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { auth } from '../store/auth'

const authRequired = computed(() => auth.authRequired === true)

const apis = [
  { method: 'GET', path: '/api/admin/overview', desc: '运行概览' },
  { method: 'GET', path: '/api/admin/counters', desc: '记录列表' },
  { method: 'POST', path: '/api/admin/counters', desc: '新建记录' },
  { method: 'POST', path: '/api/admin/counters/incr', desc: '按步长自增' },
  { method: 'POST', path: '/api/admin/counters/set', desc: '设为指定值' },
  { method: 'POST', path: '/api/admin/counters/delete', desc: '删除记录' },
  { method: 'GET', path: '/api/count', desc: '原模板计数接口' }
]
</script>

<style scoped>
.jj-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.jj-tips {
  margin: 0;
  padding-left: 18px;
  color: #5c6675;
  font-size: 13px;
  line-height: 2;
}

.jj-tips code {
  background: #eceef2;
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 12px;
}

.jj-method {
  display: inline-block;
  min-width: 48px;
  text-align: center;
  font-size: 11.5px;
  font-weight: 600;
  line-height: 1.5;
  padding: 1px 6px;
  border-radius: 4px;
  border: 1px solid transparent;
  letter-spacing: 0.4px;
}

.jj-method.get {
  color: #2f8f5b;
  background: #eefaf2;
  border-color: #b7e0c8;
}

.jj-method.post {
  color: #bf6a15;
  background: #fdf5ea;
  border-color: #f3d3ac;
}

@media (max-width: 1100px) {
  .jj-grid {
    grid-template-columns: 1fr;
  }
}
</style>
