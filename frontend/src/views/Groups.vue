<script setup>
import { ref, computed, onMounted } from 'vue'
import { useGroupsStore } from '@/stores/groups'
import { useContainersStore } from '@/stores/containers'

const groupsStore = useGroupsStore()
const containersStore = useContainersStore()

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showRuleModal = ref(false)
const showAssignModal = ref(false)
const showHistoryModal = ref(false)
const selectedGroup = ref(null)
const editingGroup = ref(null)
const operatingIds = ref(new Set())

// 群组类型选项
const groupTypes = [
  { value: 'container', label: '容器', icon: 'container', color: 'blue' },
  { value: 'project', label: 'Compose 项目', icon: 'project', color: 'purple' },
  { value: 'image', label: '镜像', icon: 'image', color: 'green' }
]

// 新建群组表单
const newGroup = ref({
  name: '',
  groupType: 'container',
  cronExpr: '',
  autoUpdate: false,
  checkUpdate: true,
  priority: 100,
  enabled: true
})

// 新建规则表单
const newRule = ref({
  ruleType: 'name_prefix',
  pattern: ''
})

// 规则预览
const previewContainers = ref([])
const previewLoading = ref(false)

// 规则类型选项
const ruleTypes = [
  { value: 'name_prefix', label: '容器名称前缀' },
  { value: 'name_suffix', label: '容器名称后缀' },
  { value: 'name_contains', label: '容器名称包含' },
  { value: 'name_regex', label: '容器名称正则' },
  { value: 'image_prefix', label: '镜像名称前缀' },
  { value: 'image_regex', label: '镜像名称正则' },
  { value: 'label_key', label: '标签键存在' },
  { value: 'label_value', label: '标签键值匹配' }
]

// Cron 预设
const cronPresets = [
  { value: '', label: '不设置定时' },
  { value: '0 0 * * *', label: '每天凌晨' },
  { value: '0 3 * * *', label: '每天凌晨3点' },
  { value: '0 0 * * 0', label: '每周日凌晨' },
  { value: '0 0 1 * *', label: '每月1日凌晨' }
]

const stats = computed(() => {
  return {
    total: groupsStore.groups.length,
    enabled: groupsStore.groups.filter(g => g.enabled).length,
    disabled: groupsStore.groups.filter(g => !g.enabled).length
  }
})

onMounted(async () => {
  await groupsStore.fetchGroups()
  await containersStore.fetchContainers()
})

function openCreateModal() {
  newGroup.value = {
    name: '',
    groupType: 'container',
    cronExpr: '',
    autoUpdate: false,
    checkUpdate: true,
    priority: 100,
    enabled: true
  }
  showCreateModal.value = true
}

function getGroupTypeLabel(type) {
  const found = groupTypes.find(t => t.value === type)
  return found?.label || type
}

function getGroupTypeColor(type) {
  const colors = {
    container: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    project: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
    image: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
  }
  return colors[type] || colors.container
}

async function createGroup() {
  if (!newGroup.value.name) {
    alert('请输入群组名称')
    return
  }
  const result = await groupsStore.createGroup(newGroup.value)
  if (result.success) {
    showCreateModal.value = false
  } else {
    alert(result.message || '创建失败')
  }
}

function openEditModal(group) {
  editingGroup.value = { ...group }
  showEditModal.value = true
}

async function updateGroup() {
  const result = await groupsStore.updateGroup(editingGroup.value.id, editingGroup.value)
  if (result.success) {
    showEditModal.value = false
  } else {
    alert(result.message || '更新失败')
  }
}

async function deleteGroup(group) {
  if (!confirm(`确定要删除群组 "${group.name}" 吗？`)) return
  const result = await groupsStore.deleteGroup(group.id)
  if (!result.success) {
    alert(result.message || '删除失败')
  }
}

async function toggleGroup(group) {
  const result = await groupsStore.updateGroup(group.id, { enabled: !group.enabled })
  if (!result.success) {
    alert(result.message || '操作失败')
  }
}

async function checkGroup(group) {
  operatingIds.value.add(group.id)
  try {
    const result = await groupsStore.checkGroup(group.id)
    alert(result.message || '检查任务已触发')
  } finally {
    operatingIds.value.delete(group.id)
  }
}

async function triggerUpdate(group) {
  if (!confirm(`确定要更新群组 "${group.name}" 中的所有容器吗？`)) return
  operatingIds.value.add(group.id)
  try {
    const result = await groupsStore.triggerGroupUpdate(group.id)
    alert(result.message || '更新任务已触发')
  } finally {
    operatingIds.value.delete(group.id)
  }
}

async function openRuleModal(group) {
  selectedGroup.value = group
  await groupsStore.fetchGroup(group.id)
  newRule.value = { ruleType: 'name_prefix', pattern: '' }
  previewContainers.value = []
  showRuleModal.value = true
}

async function previewRule() {
  if (!newRule.value.pattern) {
    previewContainers.value = []
    return
  }
  previewLoading.value = true
  try {
    const result = await groupsStore.previewRule(newRule.value.ruleType, newRule.value.pattern)
    if (result.success) {
      previewContainers.value = result.data
    }
  } finally {
    previewLoading.value = false
  }
}

async function addRule() {
  if (!newRule.value.pattern) {
    alert('请输入匹配模式')
    return
  }
  const result = await groupsStore.createRule({
    groupId: selectedGroup.value.id,
    ruleType: newRule.value.ruleType,
    pattern: newRule.value.pattern
  })
  if (result.success) {
    newRule.value = { ruleType: 'name_prefix', pattern: '' }
    previewContainers.value = []
  } else {
    alert(result.message || '添加规则失败')
  }
}

async function removeRule(ruleId) {
  if (!confirm('确定要删除此规则吗？')) return
  const result = await groupsStore.deleteRule(ruleId, selectedGroup.value.id)
  if (!result.success) {
    alert(result.message || '删除规则失败')
  }
}

async function openAssignModal(group) {
  selectedGroup.value = group
  await groupsStore.fetchGroup(group.id)
  showAssignModal.value = true
}

async function assignContainer(container) {
  const result = await groupsStore.assignContainer({
    groupId: selectedGroup.value.id,
    containerId: container.id,
    containerName: container.name
  })
  if (!result.success) {
    alert(result.message || '分配失败')
  }
}

async function unassignContainer(assignId) {
  if (!confirm('确定要移除此容器吗？')) return
  const result = await groupsStore.unassignContainer(assignId, selectedGroup.value.id)
  if (!result.success) {
    alert(result.message || '移除失败')
  }
}

async function openHistoryModal() {
  await groupsStore.fetchHistory({ page: 1, size: 50 })
  showHistoryModal.value = true
}

function getRuleTypeLabel(type) {
  const found = ruleTypes.find(t => t.value === type)
  return found?.label || type
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 获取可分配的容器列表
const availableContainers = computed(() => {
  if (!groupsStore.currentGroup) return []
  const assignedIds = new Set(groupsStore.currentGroup.containers?.map(c => c.containerId) || [])
  return containersStore.containers
    .filter(c => !assignedIds.has(c.id))
    .map(c => ({
      id: c.id,
      name: c.name || c.Names?.[0]?.replace(/^\//, '') || 'unknown',
      image: c.usingImage || c.Image
    }))
})
</script>

<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
            <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">总群组</p>
          </div>
        </div>
      </div>
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
            <svg class="w-5 h-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.enabled }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">已启用</p>
          </div>
        </div>
      </div>
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.disabled }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">已禁用</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center justify-between gap-4">
      <button @click="openCreateModal" class="btn-primary">
        <svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        新建群组
      </button>
      <button @click="openHistoryModal" class="btn-secondary">
        <svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        更新历史
      </button>
    </div>

    <!-- 群组列表 -->
    <div v-if="groupsStore.loading" class="text-center py-12">
      <div class="loading-spinner mx-auto mb-4"></div>
      <p class="text-gray-500 dark:text-gray-400">加载中...</p>
    </div>

    <div v-else-if="groupsStore.groups.length === 0" class="text-center py-12">
      <svg class="w-16 h-16 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400 mb-4">暂无群组</p>
      <button @click="openCreateModal" class="btn-primary">创建第一个群组</button>
    </div>

    <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div v-for="group in groupsStore.groups" :key="group.id" class="card p-5">
        <div class="flex items-start justify-between mb-4">
          <div>
            <div class="flex items-center gap-2 mb-1">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ group.name }}</h3>
              <span :class="['px-2 py-0.5 text-xs font-medium rounded', getGroupTypeColor(group.groupType || 'container')]">
                {{ getGroupTypeLabel(group.groupType || 'container') }}
              </span>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              优先级: {{ group.priority }} ·
              <span v-if="group.cronExpr">定时: {{ group.cronExpr }}</span>
              <span v-else>无定时任务</span>
            </p>
          </div>
          <span :class="[
            'px-2.5 py-1 text-xs font-medium rounded-full',
            group.enabled
              ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
              : 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400'
          ]">
            {{ group.enabled ? '已启用' : '已禁用' }}
          </span>
        </div>

        <!-- 群组配置信息 -->
        <div class="flex flex-wrap gap-2 mb-4">
          <span v-if="group.checkUpdate" class="px-2 py-1 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 rounded">
            检查更新
          </span>
          <span v-if="group.autoUpdate" class="px-2 py-1 text-xs bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400 rounded">
            自动更新
          </span>
        </div>

        <!-- 操作按钮 -->
        <div class="flex flex-wrap gap-2">
          <button
            @click="openRuleModal(group)"
            class="btn-sm btn-secondary"
          >
            规则管理
          </button>
          <button
            @click="openAssignModal(group)"
            class="btn-sm btn-secondary"
          >
            手动分配
          </button>
          <button
            @click="checkGroup(group)"
            :disabled="operatingIds.has(group.id)"
            class="btn-sm btn-secondary"
          >
            检查更新
          </button>
          <button
            @click="triggerUpdate(group)"
            :disabled="operatingIds.has(group.id)"
            class="btn-sm btn-warning"
          >
            立即更新
          </button>
          <button @click="openEditModal(group)" class="btn-sm btn-secondary">编辑</button>
          <button @click="toggleGroup(group)" class="btn-sm" :class="group.enabled ? 'btn-gray' : 'btn-success'">
            {{ group.enabled ? '禁用' : '启用' }}
          </button>
          <button @click="deleteGroup(group)" class="btn-sm btn-danger">删除</button>
        </div>
      </div>
    </div>

    <!-- 创建群组弹窗 -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal-content">
        <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">新建群组</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">群组名称</label>
            <input v-model="newGroup.name" type="text" class="input" placeholder="输入群组名称">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">群组类型</label>
            <div class="flex gap-2">
              <button
                v-for="type in groupTypes"
                :key="type.value"
                @click="newGroup.groupType = type.value"
                :class="[
                  'flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-all border-2',
                  newGroup.groupType === type.value
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                ]"
              >
                {{ type.label }}
              </button>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">定时任务 (Cron)</label>
            <select v-model="newGroup.cronExpr" class="input">
              <option v-for="preset in cronPresets" :key="preset.value" :value="preset.value">
                {{ preset.label }}
              </option>
            </select>
            <input v-model="newGroup.cronExpr" type="text" class="input mt-2" placeholder="或输入自定义 Cron 表达式">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">优先级 (越小越高)</label>
            <input v-model.number="newGroup.priority" type="number" class="input" min="1">
          </div>
          <div class="flex items-center gap-4">
            <label class="flex items-center gap-2">
              <input v-model="newGroup.checkUpdate" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">检查更新</span>
            </label>
            <label class="flex items-center gap-2">
              <input v-model="newGroup.autoUpdate" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">自动更新</span>
            </label>
            <label class="flex items-center gap-2">
              <input v-model="newGroup.enabled" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">启用</span>
            </label>
          </div>
        </div>
        <div class="flex justify-end gap-3 mt-6">
          <button @click="showCreateModal = false" class="btn-secondary">取消</button>
          <button @click="createGroup" class="btn-primary">创建</button>
        </div>
      </div>
    </div>

    <!-- 编辑群组弹窗 -->
    <div v-if="showEditModal && editingGroup" class="modal-overlay" @click.self="showEditModal = false">
      <div class="modal-content">
        <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">编辑群组</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">群组名称</label>
            <input v-model="editingGroup.name" type="text" class="input">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">群组类型</label>
            <div class="flex gap-2">
              <button
                v-for="type in groupTypes"
                :key="type.value"
                @click="editingGroup.groupType = type.value"
                :class="[
                  'flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-all border-2',
                  editingGroup.groupType === type.value
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                ]"
              >
                {{ type.label }}
              </button>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">定时任务 (Cron)</label>
            <select v-model="editingGroup.cronExpr" class="input">
              <option v-for="preset in cronPresets" :key="preset.value" :value="preset.value">
                {{ preset.label }}
              </option>
            </select>
            <input v-model="editingGroup.cronExpr" type="text" class="input mt-2" placeholder="或输入自定义 Cron 表达式">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">优先级</label>
            <input v-model.number="editingGroup.priority" type="number" class="input" min="1">
          </div>
          <div class="flex items-center gap-4">
            <label class="flex items-center gap-2">
              <input v-model="editingGroup.checkUpdate" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">检查更新</span>
            </label>
            <label class="flex items-center gap-2">
              <input v-model="editingGroup.autoUpdate" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">自动更新</span>
            </label>
            <label class="flex items-center gap-2">
              <input v-model="editingGroup.enabled" type="checkbox" class="checkbox">
              <span class="text-sm text-gray-700 dark:text-gray-300">启用</span>
            </label>
          </div>
        </div>
        <div class="flex justify-end gap-3 mt-6">
          <button @click="showEditModal = false" class="btn-secondary">取消</button>
          <button @click="updateGroup" class="btn-primary">保存</button>
        </div>
      </div>
    </div>

    <!-- 规则管理弹窗 -->
    <div v-if="showRuleModal && selectedGroup" class="modal-overlay" @click.self="showRuleModal = false">
      <div class="modal-content max-w-2xl">
        <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">
          规则管理 - {{ selectedGroup.name }}
        </h3>

        <!-- 现有规则 -->
        <div class="mb-6">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">现有规则</h4>
          <div v-if="groupsStore.currentGroup?.rules?.length" class="space-y-2">
            <div v-for="rule in groupsStore.currentGroup.rules" :key="rule.id"
              class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <div>
                <span class="text-sm font-medium text-gray-900 dark:text-white">{{ getRuleTypeLabel(rule.ruleType) }}</span>
                <span class="text-sm text-gray-500 dark:text-gray-400 ml-2">{{ rule.pattern }}</span>
              </div>
              <button @click="removeRule(rule.id)" class="text-red-600 hover:text-red-700">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 dark:text-gray-400">暂无规则</p>
        </div>

        <!-- 添加规则 -->
        <div class="border-t border-gray-200 dark:border-gray-600 pt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">添加规则</h4>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-3">
            <select v-model="newRule.ruleType" class="input">
              <option v-for="type in ruleTypes" :key="type.value" :value="type.value">
                {{ type.label }}
              </option>
            </select>
            <input v-model="newRule.pattern" @input="previewRule" type="text" class="input" placeholder="匹配模式">
          </div>
          <div class="flex gap-2 mb-4">
            <button @click="addRule" class="btn-primary btn-sm">添加规则</button>
            <button @click="previewRule" class="btn-secondary btn-sm">预览匹配</button>
          </div>

          <!-- 预览结果 -->
          <div v-if="previewContainers.length" class="mt-4">
            <h5 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              匹配到 {{ previewContainers.length }} 个容器
            </h5>
            <div class="max-h-40 overflow-y-auto space-y-1">
              <div v-for="c in previewContainers" :key="c.id"
                class="text-sm text-gray-600 dark:text-gray-400 p-2 bg-gray-50 dark:bg-gray-700 rounded">
                {{ c.name }} ({{ c.image }})
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end mt-6">
          <button @click="showRuleModal = false" class="btn-secondary">关闭</button>
        </div>
      </div>
    </div>

    <!-- 手动分配弹窗 -->
    <div v-if="showAssignModal && selectedGroup" class="modal-overlay" @click.self="showAssignModal = false">
      <div class="modal-content max-w-2xl">
        <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">
          手动分配容器 - {{ selectedGroup.name }}
        </h3>

        <!-- 已分配容器 -->
        <div class="mb-6">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">已分配容器</h4>
          <div v-if="groupsStore.currentGroup?.containers?.length" class="space-y-2">
            <div v-for="c in groupsStore.currentGroup.containers" :key="c.id"
              class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <span class="text-sm text-gray-900 dark:text-white">{{ c.containerName }}</span>
              <button @click="unassignContainer(c.id)" class="text-red-600 hover:text-red-700">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 dark:text-gray-400">暂无手动分配的容器</p>
        </div>

        <!-- 可分配容器 -->
        <div class="border-t border-gray-200 dark:border-gray-600 pt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">可分配容器</h4>
          <div v-if="availableContainers.length" class="max-h-60 overflow-y-auto space-y-2">
            <div v-for="c in availableContainers" :key="c.id"
              class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <div>
                <span class="text-sm font-medium text-gray-900 dark:text-white">{{ c.name }}</span>
                <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">{{ c.image }}</span>
              </div>
              <button @click="assignContainer(c)" class="btn-sm btn-primary">分配</button>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 dark:text-gray-400">没有可分配的容器</p>
        </div>

        <div class="flex justify-end mt-6">
          <button @click="showAssignModal = false" class="btn-secondary">关闭</button>
        </div>
      </div>
    </div>

    <!-- 更新历史弹窗 -->
    <div v-if="showHistoryModal" class="modal-overlay" @click.self="showHistoryModal = false">
      <div class="modal-content max-w-3xl">
        <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">更新历史</h3>

        <div v-if="groupsStore.history.length" class="max-h-96 overflow-y-auto">
          <table class="w-full">
            <thead class="sticky top-0 bg-gray-50 dark:bg-gray-700">
              <tr>
                <th class="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2">容器</th>
                <th class="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2">状态</th>
                <th class="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2">时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-600">
              <tr v-for="h in groupsStore.history" :key="h.id">
                <td class="px-3 py-2 text-sm text-gray-900 dark:text-white">{{ h.containerName }}</td>
                <td class="px-3 py-2">
                  <span :class="[
                    'px-2 py-1 text-xs font-medium rounded',
                    h.status === 'success' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                    h.status === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                    'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400'
                  ]">
                    {{ h.status === 'success' ? '成功' : h.status === 'failed' ? '失败' : h.status }}
                  </span>
                </td>
                <td class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">{{ formatDate(h.createdAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="text-center text-gray-500 dark:text-gray-400 py-8">暂无更新历史</p>

        <div class="flex justify-end mt-6">
          <button @click="showHistoryModal = false" class="btn-secondary">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  @apply bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700;
}

.btn-primary {
  @apply inline-flex items-center px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium;
}

.btn-secondary {
  @apply inline-flex items-center px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors font-medium;
}

.btn-warning {
  @apply inline-flex items-center px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition-colors font-medium;
}

.btn-danger {
  @apply inline-flex items-center px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-medium;
}

.btn-success {
  @apply inline-flex items-center px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors font-medium;
}

.btn-gray {
  @apply inline-flex items-center px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors font-medium;
}

.btn-sm {
  @apply px-3 py-1.5 text-sm;
}

.input {
  @apply w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent;
}

.checkbox {
  @apply w-4 h-4 text-primary-600 border-gray-300 dark:border-gray-600 rounded focus:ring-primary-500;
}

.modal-overlay {
  @apply fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4;
}

.modal-content {
  @apply bg-white dark:bg-gray-800 rounded-xl shadow-xl p-6 w-full max-w-md max-h-[90vh] overflow-y-auto;
}

.loading-spinner {
  @apply w-8 h-8 border-4 border-primary-200 dark:border-primary-800 border-t-primary-600 rounded-full animate-spin;
}
</style>
