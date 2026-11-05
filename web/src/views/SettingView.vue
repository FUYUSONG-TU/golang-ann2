<template>
  <div class="jj-page">
    <div class="jj-page-head">
      <div>
        <h2 class="jj-page-title">系统设置</h2>
        <p class="jj-page-desc">
          站点基础信息与小程序端配置，保存后立即生效。未填写的项会回退到默认值。
        </p>
      </div>
      <div class="jj-page-actions">
        <el-button v-if="canEdit" type="primary" :loading="saving" @click="save">
          <el-icon><Check /></el-icon>保存设置
        </el-button>
      </div>
    </div>

    <div v-loading="loading" class="setting-wrap">
      <div v-for="group in groups" :key="group.group" class="jj-card">
        <div class="jj-card-head">
          <div class="jj-card-title">
            {{ groupTitle(group.group) }}
            <span class="jj-card-title-sub">{{ group.items.length }} 项</span>
          </div>
        </div>
        <div class="jj-card-body">
          <el-form label-width="120px" label-position="right">
            <el-form-item v-for="item in group.items" :key="item.key" :label="item.label">
              <div class="field">
                <el-input
                  v-if="item.type === 'text'"
                  v-model="values[item.key]"
                  :disabled="!canEdit"
                  :placeholder="'请输入' + item.label"
                  style="max-width: 420px"
                />
                <el-input
                  v-else-if="item.type === 'textarea'"
                  v-model="values[item.key]"
                  type="textarea"
                  :rows="3"
                  :disabled="!canEdit"
                  :placeholder="'请输入' + item.label"
                  style="max-width: 560px"
                />
                <el-switch
                  v-else-if="item.type === 'switch'"
                  v-model="switchValues[item.key]"
                  :disabled="!canEdit"
                />
                <el-input
                  v-else
                  v-model="values[item.key]"
                  :disabled="!canEdit"
                  :placeholder="'请输入' + item.label"
                  style="max-width: 420px"
                />
                <div class="jj-sub field-tip">
                  {{ item.remark }}<span v-if="item.type === 'switch'"> · 当前为{{ switchValues[item.key] ? '开启' : '关闭' }}</span>
                </div>
              </div>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <div v-if="!groups.length && !loading" class="jj-card">
        <div class="jj-empty">暂无可配置项</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { canWrite } from '../store/auth'

const reloadTick = inject('jjReloadTick', ref(0))
const canEdit = computed(() => canWrite('configs'))

const loading = ref(false)
const saving = ref(false)
const groups = ref([])
const values = reactive({})
const switchValues = reactive({})

const groupTitles = {
  basic: '站点基础信息',
  mini: '小程序配置',
  custom: '自定义配置'
}

function groupTitle(name) {
  return groupTitles[name] || name
}

async function load() {
  loading.value = true
  try {
    const data = await api.configs.list()
    groups.value = data.groups || []
    Object.keys(values).forEach((k) => delete values[k])
    Object.keys(switchValues).forEach((k) => delete switchValues[k])

    // 只有 type 为 switch 的项才进开关状态表，文本项一律按原始字符串保存，
    // 否则保存时会把所有文本配置写成 "false"
    const switchKeys = new Set()
    groups.value.forEach((group) => {
      ;(group.items || []).forEach((item) => {
        if (item.type === 'switch') switchKeys.add(item.key)
      })
    })

    const source = data.values || {}
    switchKeys.forEach((key) => {
      switchValues[key] = source[key] === 'true' || source[key] === '1'
    })
    Object.entries(source).forEach(([k, v]) => {
      if (!switchKeys.has(k)) values[k] = v == null ? '' : String(v)
    })
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const keys = new Set([...Object.keys(values), ...Object.keys(switchValues)])
    const items = [...keys].map((key) => ({
      key,
      value: key in switchValues ? (switchValues[key] ? 'true' : 'false') : String(values[key] ?? '')
    }))
    const data = await api.configs.save(items)
    // 用后端回写的结果刷新本地态，避免前后端不一致
    const saved = data.values || {}
    Object.keys(switchValues).forEach((k) => {
      if (k in saved) switchValues[k] = saved[k] === 'true' || saved[k] === '1'
    })
    Object.keys(values).forEach((k) => {
      if (k in saved) values[k] = saved[k] == null ? '' : String(saved[k])
    })
    ElMessage.success(`已保存 ${data.saved} 项设置`)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

watch(reloadTick, load)
onMounted(load)
</script>

<style scoped>
.setting-wrap {
  max-width: 880px;
}

.field {
  width: 100%;
}

.field-tip {
  margin-top: 4px;
  line-height: 1.5;
}
</style>
