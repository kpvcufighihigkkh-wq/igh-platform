<template>
  <div>
    <n-tabs type="line" v-model:value="tab">
      <n-tab-pane name="bindings" tab="工位绑定">
        <n-space justify="space-between" style="margin-bottom:12px">
          <span>通过IP地址自动识别工位类型，免登录显示对应操作界面</span>
          <n-button type="primary" size="small" @click="openBindForm()">新增绑定</n-button>
        </n-space>
        <n-data-table :columns="bindCols" :data="bindings" :bordered="false" size="small" />
      </n-tab-pane>
      <n-tab-pane name="productTypes" tab="产品类型">
        <n-space justify="space-between" style="margin-bottom:12px">
          <span>管理可生产的产品类型，产线可多选</span>
          <n-button type="primary" size="small" @click="openPtForm()">新增类型</n-button>
        </n-space>
        <n-data-table :columns="ptCols" :data="productTypes" :bordered="false" size="small" />
      </n-tab-pane>
      <n-tab-pane name="modules" tab="模块配置">
        <p style="margin-bottom:12px;color:var(--steel)">切换模块开关后立即生效，无需重启服务。关闭模块后对应API将返回403，管理菜单置灰。</p>
        <n-space vertical size="large">
          <div v-for="m in moduleList" :key="m.key" style="display:flex;align-items:flex-start;gap:12px">
            <n-switch :value="modules[m.key]" :disabled="m.key==='Winding'" @update:value="(val: boolean) => toggleModule(m.key, val)" style="margin-top:2px" />
            <div>
              <div style="font-weight:bold">{{ m.label }}<span v-if="m.key==='Winding'" style="font-size:11px;color:var(--ok);margin-left:6px">（核心模块，不可关闭）</span></div>
              <div style="font-size:12px;color:var(--steel)">{{ m.desc }}</div>
            </div>
          </div>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="ha" tab="高可用">
        <div style="display:flex;align-items:center;gap:12px;margin-bottom:16px">
          <n-switch :value="ha.mode!=='standalone'" @update:value="(val: boolean) => { ha.mode = val ? 'active-standby' : 'standalone' }" />
          <span style="font-weight:bold">{{ ha.mode === 'standalone' ? '未启用高可用' : '已启用高可用（主备模式）' }}</span>
        </div>
        <template v-if="ha.mode!=='standalone'">
          <n-form :model="ha" label-placement="left" label-width="90" style="max-width:400px">
            <n-form-item label="虚拟IP"><n-input v-model:value="ha.vip" placeholder="如 192.168.1.100" /></n-form-item>
            <n-form-item label="对端节点"><n-input v-model:value="ha.peer" placeholder="对端IP地址" /></n-form-item>
            <n-form-item label="心跳网卡"><n-input v-model:value="ha.heartbeat_interface" placeholder="eth0" /></n-form-item>
          </n-form>
        </template>
        <n-space style="margin-top:12px">
          <n-button type="primary" size="small" @click="saveHA">保存配置</n-button>
          <n-popconfirm @positive-click="restartService">
            <template #trigger><n-button type="warning" size="small">重启服务生效</n-button></template>
            确认重启服务？重启期间服务将短暂不可用。
          </n-popconfirm>
        </n-space>
        <n-alert type="info" style="margin-top:12px">保存配置后需点击"重启服务生效"，服务将自动重启并加载新配置。</n-alert>
      </n-tab-pane>
      <n-tab-pane name="dict" tab="数据字典">
        <n-space style="margin-bottom:12px" justify="space-between">
          <n-select v-model:value="dictCategory" :options="dictCategoryOpts" style="width:160px" @update:value="fetchDict" />
          <n-button type="primary" size="small" @click="openDictForm()">新增字典项</n-button>
        </n-space>
        <n-data-table :columns="dictCols" :data="dictItems" :bordered="false" size="small" />
      </n-tab-pane>
      <n-tab-pane name="users" tab="用户管理">
        <n-space justify="space-between" style="margin-bottom:12px">
          <n-input v-model:value="userKeyword" placeholder="搜索用户名或姓名" clearable style="width:240px" @update:value="fetchUsers" />
          <n-button type="primary" size="small" @click="openUserForm()">新增用户</n-button>
        </n-space>
        <n-data-table :columns="userCols" :data="users" :bordered="false" size="small" />
        <n-alert v-if="legacyHashCount > 0" type="warning" style="margin-top:12px">
          还有 {{ legacyHashCount }} 个账号使用旧口令哈希，用户下次登录时会自动升级为加盐哈希。
        </n-alert>
      </n-tab-pane>
      <n-tab-pane name="logs" tab="系统日志">
        <n-button size="small" @click="fetchLogs" style="margin-bottom:8px">刷新</n-button>
        <div class="log-box"><pre>{{ logLines.join('\n') }}</pre></div>
      </n-tab-pane>
    </n-tabs>

    <!-- 工位绑定表单 -->
    <n-modal v-model:show="showBindForm" preset="card" :title="bindForm.id?'编辑工位绑定':'新增工位绑定'" style="width:400px">
      <n-form :model="bindForm" label-placement="left" label-width="70">
        <n-form-item label="IP地址"><n-input v-model:value="bindForm.ip" placeholder="192.168.1.x" /></n-form-item>
        <n-form-item label="工位类型"><n-select v-model:value="bindForm.station_type" :options="stationOpts" /></n-form-item>
        <n-form-item label="产线">
          <n-select v-model:value="bindForm.line_code" :options="lineOpts" placeholder="请选择产线" />
        </n-form-item>
        <n-form-item label="备注"><n-input v-model:value="bindForm.description" /></n-form-item>
        <n-button type="primary" block @click="saveBinding">保存</n-button>
      </n-form>
    </n-modal>


    <!-- 产品类型表单 -->
    <n-modal v-model:show="showPtForm" preset="card" :title="ptForm.id?'编辑产品类型':'新增产品类型'" style="width:360px">
      <n-form :model="ptForm" label-placement="left" label-width="50">
        <n-form-item label="代码"><n-input v-model:value="ptForm.code" placeholder="FDY" /></n-form-item>
        <n-form-item label="名称"><n-input v-model:value="ptForm.name" placeholder="全拉伸丝" /></n-form-item>
        <n-button type="primary" block @click="savePt">保存</n-button>
      </n-form>
    </n-modal>

    <!-- 字典表单 -->
    <n-modal v-model:show="showDictForm" preset="card" :title="dictForm.id?'编辑字典项':'新增字典项'" style="width:360px">
      <n-form :model="dictForm" label-placement="left" label-width="50">
        <n-form-item label="代码"><n-input v-model:value="dictForm.code" placeholder="code" /></n-form-item>
        <n-form-item label="名称"><n-input v-model:value="dictForm.name" placeholder="显示名称" /></n-form-item>
        <n-form-item label="排序"><n-input-number v-model:value="dictForm.sort" :min="0" /></n-form-item>
        <n-button type="primary" block @click="saveDict">保存</n-button>
      </n-form>
    </n-modal>
    <!-- 用户表单 -->
    <n-modal v-model:show="showUserForm" preset="card" :title="userForm.id?'编辑用户':'新增用户'" style="width:420px">
      <n-form :model="userForm" label-placement="left" label-width="70">
        <n-form-item label="用户名">
          <n-input v-model:value="userForm.username" :disabled="!!userForm.id" placeholder="登录名，创建后不可修改" />
        </n-form-item>
        <n-form-item v-if="!userForm.id" label="初始口令">
          <n-input v-model:value="userForm.password" type="password" show-password-on="click" placeholder="至少 8 位" />
        </n-form-item>
        <n-form-item label="姓名"><n-input v-model:value="userForm.name" /></n-form-item>
        <n-form-item label="电话"><n-input v-model:value="userForm.phone" placeholder="选填" /></n-form-item>
        <n-form-item label="角色"><n-select v-model:value="userForm.role_id" :options="roleOpts" /></n-form-item>
        <n-form-item v-if="userForm.id" label="状态">
          <n-select v-model:value="userForm.status" :options="[{label:'启用',value:1},{label:'停用',value:0}]" />
        </n-form-item>
        <n-button type="primary" block @click="saveUser">保存</n-button>
      </n-form>
    </n-modal>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { NButton, NSpace, NTag } from 'naive-ui'
import http from '../../api/http'
import { tStation, tStatus } from '../../utils/status'

const tab = ref('bindings')
const bindings = ref<any[]>([])
const lines = ref<any[]>([])
const productTypes = ref<any[]>([])
const modules = ref<any>({})
const ha = ref<any>({})
const logLines = ref<string[]>([])
const showBindForm = ref(false)
const showPtForm = ref(false)
const showDictForm = ref(false)
const bindForm = ref({ id: 0, ip: '', station_type: 'winding', line_code: '', description: '' })
const ptForm = ref({ id: 0, code: '', name: '' })
const dictItems = ref<any[]>([])
const dictCategory = ref('device_type')
const dictForm = ref({ id: 0, category: '', code: '', name: '', sort: 0 })
const dictCategories = ref<any[]>([])
const dictCategoryOpts = computed(() => dictCategories.value.map(d => ({ label: d.name, value: d.code })))

// --- 用户管理 ---
const users = ref<any[]>([])
const roles = ref<any[]>([])
const userKeyword = ref('')
const showUserForm = ref(false)
const userForm = ref({ id: 0, username: '', password: '', name: '', phone: '', role_id: 0, status: 1 })
const legacyHashCount = ref(0)
const roleOpts = computed(() => roles.value.map((r: any) => ({ label: r.name, value: r.id })))

const stationOpts = [
  { label: '卷绕工位', value: 'winding' },
  { label: '分拣工位', value: 'sorting' },
  { label: '包装工位', value: 'packaging' },
  { label: '仓库工位', value: 'warehouse' },
]
const lineOpts = computed(() => lines.value
  .filter(l => l.station_type === bindForm.value.station_type)
  .map(l => ({ label: `${l.code} - ${l.name}`, value: l.code }))
)

// --- PLACEHOLDER_SCRIPT_CONTINUE ---

const bindCols = [
  { title: 'IP', key: 'ip', width: 140 },
  { title: '工位类型', key: 'station_type', width: 100, render: (r: any) => tStation(r.station_type) },
  { title: '产线', key: 'line_code', width: 80 },
  { title: '备注', key: 'description' },
  { title: '操作', key: 'a', width: 120, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openBindForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', type: 'error', onClick: () => delBinding(r.id) }, () => '删除'),
  ]) },
]
const ptCols = [
  { title: '代码', key: 'code', width: 100 },
  { title: '名称', key: 'name' },
  { title: '操作', key: 'a', width: 120, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openPtForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', type: 'error', onClick: () => delPt(r.id) }, () => '删除'),
  ]) },
]
const dictCols = [
  { title: '代码', key: 'code', width: 140 },
  { title: '名称', key: 'name', width: 140 },
  { title: '排序', key: 'sort', width: 60 },
  { title: '操作', key: 'a', width: 120, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openDictForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', type: 'error', onClick: () => delDict(r.id) }, () => '删除'),
  ]) },
]
const userCols = [
  { title: '用户名', key: 'username', width: 120 },
  { title: '姓名', key: 'name', width: 100 },
  { title: '角色', key: 'role_name', width: 110 },
  { title: '电话', key: 'phone', width: 130 },
  { title: '状态', key: 'status', width: 80, render: (r: any) =>
    h(NTag, { size: 'small', type: r.status === 1 ? 'success' : 'default' },
      () => (r.status === 1 ? '启用' : '停用')) },
  { title: '操作', key: 'a', width: 200, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openUserForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', onClick: () => resetPassword(r) }, () => '重置口令'),
    // 停用的账号允许重新启用；停用与删除都不物理删记录
    h(NButton, {
      size: 'tiny',
      type: r.status === 1 ? 'error' : 'primary',
      onClick: () => toggleUserStatus(r),
    }, () => (r.status === 1 ? '停用' : '启用')),
  ]) },
]

const moduleList = [  { key: 'Winding', label: '卷绕区', desc: '批次管理、落纱记录、丝饼数据、料车状态、卷绕工位页面' },
  { key: 'Quality', label: '质检区', desc: '等级定义、称重规则、分拣结果录入、视觉检测、分拣工位页面' },
  { key: 'Warehouse', label: '仓库区', desc: '库位管理、入库/出库操作、仓库看板、仓库工位页面' },
  { key: 'Packaging', label: '包装区', desc: '拣选订单、打包订单、托盘管理、包装工位页面' },
  { key: 'ERP', label: 'ERP对接', desc: 'ERP同步队列、数据推送/拉取、失败重试、对接状态监控' },
  { key: 'Maintenance', label: '维保管理', desc: '设备台账、维保计划、维保记录、逾期预警' },
  { key: 'Vehicle', label: '载具管理', desc: '丝车、吊车、天轨小车、AGV、穿梭车的统一管理与调度' },
]

function moduleLabel(k: string) {
  const m: Record<string, string> = { Winding: '卷绕区', Quality: '质检区', Warehouse: '仓库区', Packaging: '包装区', ERP: 'ERP对接', Maintenance: '维保管理' }
  return m[k] || k
}

function openBindForm(row?: any) {
  if (row) {
    bindForm.value = { id: row.id, ip: row.ip, station_type: row.station_type, line_code: row.line_code, description: row.description || '' }
  } else {
    bindForm.value = { id: 0, ip: '', station_type: 'winding', line_code: '', description: '' }
  }
  showBindForm.value = true
}
function openPtForm(row?: any) {
  if (row) {
    ptForm.value = { id: row.id, code: row.code, name: row.name }
  } else {
    ptForm.value = { id: 0, code: '', name: '' }
  }
  showPtForm.value = true
}
function openDictForm(row?: any) {
  if (row) {
    dictForm.value = { id: row.id, category: row.category, code: row.code, name: row.name, sort: row.sort }
  } else {
    dictForm.value = { id: 0, category: dictCategory.value, code: '', name: '', sort: 0 }
  }
  showDictForm.value = true
}
// --- PLACEHOLDER_FETCH ---

async function fetchBindings() { bindings.value = ((await http.get('/admin/station-bindings')) as any).data || [] }
async function fetchLines() { lines.value = ((await http.get('/admin/production-lines')) as any).data || [] }
async function fetchProductTypes() { productTypes.value = ((await http.get('/admin/product-types')) as any).data || [] }
async function fetchModules() { modules.value = ((await http.get('/admin/modules')) as any).data || {} }
async function fetchHA() { ha.value = ((await http.get('/admin/ha-status')) as any).data || {} }
async function fetchLogs() { logLines.value = ((await http.get('/admin/logs?lines=50')) as any).data?.lines || [] }
async function fetchDictCategories() { dictCategories.value = ((await http.get('/admin/dict-items?category=dict_category')) as any).data || [] }
async function fetchDict() { dictItems.value = ((await http.get(`/admin/dict-items?category=${dictCategory.value}`)) as any).data || [] }

// --- 用户管理 ---
async function fetchUsers() {
  const res: any = await http.get(`/admin/users?keyword=${encodeURIComponent(userKeyword.value)}`)
  users.value = res.data?.items || []
  try {
    const hc: any = await http.get('/admin/users/legacy-hash-count')
    legacyHashCount.value = hc.data?.count || 0
  } catch { legacyHashCount.value = 0 }
}
async function fetchRoles() { roles.value = ((await http.get('/admin/roles')) as any).data?.items || [] }

function openUserForm(row?: any) {
  if (row) {
    userForm.value = {
      id: row.id, username: row.username, password: '',
      name: row.name, phone: row.phone || '', role_id: row.role_id, status: row.status,
    }
  } else {
    userForm.value = {
      id: 0, username: '', password: '', name: '', phone: '',
      role_id: roles.value.length ? roles.value[0].id : 0, status: 1,
    }
  }
  showUserForm.value = true
}

async function saveUser() {
  try {
    if (userForm.value.id) {
      // 编辑时不提交 username/password：用户名不可改，口令走独立的改动流程
      await http.put(`/admin/users/${userForm.value.id}`, {
        name: userForm.value.name,
        phone: userForm.value.phone,
        role_id: userForm.value.role_id,
        status: userForm.value.status,
      })
    } else {
      await http.post('/admin/users', userForm.value)
    }
    showUserForm.value = false
    fetchUsers()
  } catch (e: any) {
    // 服务端用 409 区分"用户名已存在"这类可预期错误，原样提示比"保存失败"有用
    alert(e?.response?.data?.message || '保存失败')
  }
}

async function toggleUserStatus(row: any) {
  const next = row.status === 1 ? 0 : 1
  if (next === 0 && !confirm(`确认停用账号 ${row.username}？停用后无法登录，但历史记录保留。`)) return
  try {
    await http.put(`/admin/users/${row.id}`, { status: next })
    fetchUsers()
  } catch (e: any) {
    alert(e?.response?.data?.message || '操作失败')
  }
}

async function resetPassword(row: any) {
  const pwd = prompt(`为 ${row.username} 设置新口令（至少 8 位）`)
  if (!pwd) return
  try {
    await http.post(`/admin/users/${row.id}/reset-password`, { password: pwd })
    alert('口令已重置，请通知本人尽快修改')
  } catch (e: any) {
    alert(e?.response?.data?.message || '重置失败')
  }
}

async function saveDict() {
  const payload = { ...dictForm.value, category: dictForm.value.category || dictCategory.value }
  if (dictForm.value.id) {
    await http.put(`/admin/dict-items/${dictForm.value.id}`, payload)
  } else {
    await http.post('/admin/dict-items', payload)
  }
  showDictForm.value = false; fetchDict(); if (dictCategory.value === 'dict_category') fetchDictCategories()
}
async function delDict(id: number) { await http.delete(`/admin/dict-items/${id}`); fetchDict(); if (dictCategory.value === 'dict_category') fetchDictCategories() }

async function saveBinding() {
  if (bindForm.value.id) {
    await http.put(`/admin/station-bindings/${bindForm.value.id}`, bindForm.value)
  } else {
    await http.post('/admin/station-bindings', bindForm.value)
  }
  showBindForm.value = false; fetchBindings()
}
async function delBinding(id: number) { await http.delete(`/admin/station-bindings/${id}`); fetchBindings() }


async function savePt() {
  if (ptForm.value.id) {
    await http.put(`/admin/product-types/${ptForm.value.id}`, ptForm.value)
  } else {
    await http.post('/admin/product-types', ptForm.value)
  }
  showPtForm.value = false; fetchProductTypes()
}
async function delPt(id: number) { await http.delete(`/admin/product-types/${id}`); fetchProductTypes() }

async function toggleModule(key: string, val: boolean) {
  modules.value[key] = val
  await http.put('/admin/modules', modules.value)
}
async function saveHA() {
  await http.put('/admin/ha-status', ha.value)
}
async function restartService() {
  try { await http.post('/admin/restart') } catch {}
}

onMounted(() => { fetchBindings(); fetchLines(); fetchProductTypes(); fetchModules(); fetchHA(); fetchDictCategories(); fetchDict(); fetchRoles(); fetchUsers() })
</script>

<style scoped>
.log-box { background: #0a0a1a; border-radius: 4px; padding: 12px; max-height: 400px; overflow: auto; }
.log-box pre { font-size: 11px; color: #aaa; white-space: pre-wrap; word-break: break-all; margin: 0; }
</style>
