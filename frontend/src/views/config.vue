<template>
  <div class="flex h-full w-full overflow-hidden bg-background">
    <div class="w-56 border-r border-border flex flex-col bg-sidebar">
      <div class="p-4 border-b border-border">
        <h2 class="text-lg font-semibold text-sidebar-foreground">配置管理</h2>
      </div>
      <nav class="flex-1 overflow-y-auto p-2">
        <ul class="space-y-1">
          <li v-for="section in sections" :key="section.id">
            <button
              @click="scrollToSection(section.id)"
              :class="[
                'w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors text-left',
                activeSection === section.id
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground'
              ]"
            >
              <component :is="section.icon" class="w-4 h-4" />
              {{ section.label }}
            </button>
          </li>
        </ul>
      </nav>
    </div>

    <div class="flex-1 flex flex-col overflow-hidden">
      <div
        v-if="config"
        ref="contentRef"
        @scroll="handleScroll"
        class="flex-1 overflow-y-auto p-6"
      >
        <div class="max-w-4xl mx-auto space-y-8 pb-24">
          <section id="section-providers" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <Server class="w-5 h-5" />
              Providers
            </h3>
            <div class="space-y-4">
              <div v-for="(provider, name) in config?.providers" :key="name" class="p-4 border border-border rounded-lg bg-card">
                <div class="flex items-center justify-between mb-3 gap-4">
                  <Input
                    :modelValue="name"
                    @update:modelValue="(newName: string | number) => renameProvider(name as string, String(newName))"
                    class="font-medium"
                  />
                  <Button variant="ghost" size="icon-sm" @click="removeProvider(name as string)" class="text-destructive hover:text-destructive shrink-0">
                    <Trash2 class="w-4 h-4" />
                  </Button>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">类型</label>
                    <Select v-model="provider!.type" @update:modelValue="markChanged">
                      <SelectTrigger class="w-full">
                        <SelectValue placeholder="选择类型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="pt in providerTypes" :key="pt.value" :value="pt.value">
                          {{ pt.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">API Key</label>
                    <Input v-model="provider!.api_key" type="password" @input="markChanged" />
                  </div>
                  <div class="col-span-2 space-y-2">
                    <label class="text-sm text-muted-foreground">Base URL</label>
                    <Input v-model="provider!.base_url" @input="markChanged" />
                  </div>
                </div>
              </div>
              <Button variant="outline" @click="addProvider" class="w-full">
                <Plus class="w-4 h-4 mr-2" />
                添加 Provider
              </Button>
            </div>
          </section>

          <section id="section-agents" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <Bot class="w-5 h-5" />
              Agents
            </h3>
            <div class="space-y-4">
              <div v-for="(agent, name) in config?.agents" :key="name" class="p-4 border border-border rounded-lg bg-card">
                <div class="flex items-center justify-between mb-3 gap-4">
                  <Input
                    :modelValue="name"
                    @update:modelValue="(newName: string | number) => renameAgent(name as string, String(newName))"
                    class="font-medium"
                  />
                  <Button variant="ghost" size="icon-sm" @click="removeAgent(name as string)" class="text-destructive hover:text-destructive shrink-0">
                    <Trash2 class="w-4 h-4" />
                  </Button>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">Provider</label>
                    <Select v-model="agent!.provider" @update:modelValue="markChanged">
                      <SelectTrigger class="w-full">
                        <SelectValue placeholder="选择 Provider" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="pName in providerNames" :key="pName" :value="pName">
                          {{ pName }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">Model</label>
                    <Input v-model="agent!.model" @input="markChanged" />
                  </div>
                  <div class="col-span-2 space-y-2">
                    <label class="text-sm text-muted-foreground">Workspace</label>
                    <Input v-model="agent!.workspace" @input="markChanged" />
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">Temperature</label>
                    <Input v-model.number="agent!.temperature" type="number" step="0.1" min="0" max="2" @input="markChanged" />
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">Max Tokens</label>
                    <Input v-model.number="agent!.max_tokens" type="number" @input="markChanged" />
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">记忆整理时间</label>
                    <Input v-model="agent!.memory_organize_time" placeholder="HH:MM" @input="markChanged" />
                  </div>
                  <div class="flex items-center gap-2">
                    <Switch v-model:checked="agent!.enable_image_recognition" @update:checked="markChanged" />
                    <label class="text-sm text-muted-foreground">启用图像识别</label>
                  </div>
                </div>
              </div>
              <Button variant="outline" @click="addAgent" class="w-full">
                <Plus class="w-4 h-4 mr-2" />
                添加 Agent
              </Button>
            </div>
          </section>

          <section id="section-channels" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <MessageSquare class="w-5 h-5" />
              Channels
            </h3>
            <div class="space-y-4">
              <div v-for="(channel, name) in config?.channels" :key="name" class="p-4 border border-border rounded-lg bg-card">
                <div class="flex items-center justify-between mb-3 gap-4">
                  <Input
                    :modelValue="name"
                    @update:modelValue="(newName: string | number) => renameChannel(name as string, String(newName))"
                    class="font-medium"
                  />
                  <Button variant="ghost" size="icon-sm" @click="removeChannel(name as string)" class="text-destructive hover:text-destructive shrink-0">
                    <Trash2 class="w-4 h-4" />
                  </Button>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">类型</label>
                    <Select v-model="channel!.type" @update:modelValue="markChanged">
                      <SelectTrigger class="w-full">
                        <SelectValue placeholder="选择类型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="ct in channelTypes" :key="ct.value" :value="ct.value">
                          {{ ct.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-2">
                    <label class="text-sm text-muted-foreground">Agent</label>
                    <Select v-model="channel!.agent" @update:modelValue="markChanged">
                      <SelectTrigger class="w-full">
                        <SelectValue placeholder="选择 Agent" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="aName in agentNames" :key="aName" :value="aName">
                          {{ aName }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="col-span-2 flex items-center gap-2">
                    <Switch v-model:checked="channel!.enabled" @update:checked="markChanged" />
                    <label class="text-sm text-muted-foreground">启用</label>
                  </div>
                  <template v-if="channel?.type === 'feishu'">
                    <div class="space-y-2">
                      <label class="text-sm text-muted-foreground">App ID</label>
                      <Input v-model="channel!.app_id" @input="markChanged" />
                    </div>
                    <div class="space-y-2">
                      <label class="text-sm text-muted-foreground">App Secret</label>
                      <Input v-model="channel!.app_secret" type="password" @input="markChanged" />
                    </div>
                  </template>
                  <template v-if="channel?.type === 'web'">
                    <div class="space-y-2">
                      <label class="text-sm text-muted-foreground">监听地址</label>
                      <Input v-model="channel!.host_address" @input="markChanged" />
                    </div>
                    <div class="space-y-2">
                      <label class="text-sm text-muted-foreground">Token</label>
                      <Input v-model="channel!.token" type="password" @input="markChanged" />
                    </div>
                  </template>
                </div>
              </div>
              <Button variant="outline" @click="addChannel" class="w-full">
                <Plus class="w-4 h-4 mr-2" />
                添加 Channel
              </Button>
            </div>
          </section>

          <section id="section-skill" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <Wrench class="w-5 h-5" />
              Skill
            </h3>
            <div class="p-4 border border-border rounded-lg bg-card">
              <div class="space-y-2">
                <label class="text-sm text-muted-foreground">Skills 全局路径</label>
                <div class="flex gap-2">
                  <Input v-model="config!.skill.global_path" @input="markChanged" class="flex-1" />
                  <Button variant="outline" size="icon" @click="selectSkillFolder">
                    <FolderOpen class="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </div>
          </section>

          <section id="section-browser" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <Globe class="w-5 h-5" />
              Browser
            </h3>
            <div class="p-4 border border-border rounded-lg bg-card">
              <div class="space-y-2">
                <label class="text-sm text-muted-foreground">浏览器数据目录</label>
                <div class="flex gap-2">
                  <Input v-model="config!.browser.data_dir" @input="markChanged" class="flex-1" />
                  <Button variant="outline" size="icon" @click="selectBrowserFolder">
                    <FolderOpen class="w-4 h-4" />
                  </Button>
                </div>
                <p class="text-xs text-muted-foreground">用于持久化 cookies、登录状态等</p>
              </div>
            </div>
          </section>

          <section id="section-live2d" class="scroll-mt-6">
            <h3 class="text-xl font-semibold mb-4 flex items-center gap-2">
              <Sparkles class="w-5 h-5" />
              Live2D
            </h3>
            <div class="p-4 border border-border rounded-lg bg-card">
              <div class="grid grid-cols-2 gap-4">
                <div class="col-span-2 flex items-center gap-2">
                  <Switch v-model:checked="config!.live2d.enabled" @update:checked="markChanged" />
                  <label class="text-sm text-muted-foreground">启用 Live2D</label>
                </div>
                <div class="col-span-2 space-y-2">
                  <label class="text-sm text-muted-foreground">模型目录</label>
                  <div class="flex gap-2">
                    <Input v-model="config!.live2d.model_dir" placeholder="选择模型存放目录" @input="onModelDirChange" class="flex-1" />
                    <Button variant="outline" size="icon" @click="selectLive2DModelFolder">
                      <FolderOpen class="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                <div class="col-span-2 space-y-2">
                  <label class="text-sm text-muted-foreground">选择模型</label>
                  <Select v-model="config!.live2d.model_name" @update:modelValue="markChanged">
                    <SelectTrigger class="w-full">
                      <SelectValue placeholder="选择模型" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="model in live2dModels" :key="model" :value="model">
                        {{ model }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="col-span-2 space-y-2">
                  <label class="text-sm text-muted-foreground">缩放 (%)</label>
                  <div class="flex items-center gap-4">
                    <Slider
                      v-model="live2dScaleSlider"
                      :min="10"
                      :max="200"
                      :step="1"
                      class="flex-1"
                      @update:modelValue="onScaleSliderChange"
                    />
                    <Input
                      v-model.number="config!.live2d.scale"
                      type="number"
                      min="10"
                      max="200"
                      class="w-20"
                      @input="markChanged"
                    />
                  </div>
                </div>
                <div class="space-y-2">
                  <label class="text-sm text-muted-foreground">X 位置</label>
                  <Input v-model.number="config!.live2d.x" type="number" @input="markChanged" />
                </div>
                <div class="space-y-2">
                  <label class="text-sm text-muted-foreground">Y 位置</label>
                  <Input v-model.number="config!.live2d.y" type="number" @input="markChanged" />
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      <div class="absolute bottom-0 left-56 right-0 p-4 bg-background/95 border-t border-border backdrop-blur-sm">
        <div class="max-w-4xl mx-auto flex items-center justify-between">
          <span v-if="hasChanges" class="text-sm text-muted-foreground">有未保存的更改</span>
          <span v-else class="text-sm text-muted-foreground/50">无更改</span>
          <Button :disabled="!hasChanges" @click="saveConfig" :variant="hasChanges ? 'default' : 'outline'">
            <Save class="w-4 h-4 mr-2" />
            保存配置
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, markRaw, computed } from 'vue'
import { ConfigBundle } from '../../bindings/github.com/yockii/wangshu/internal/bundle'
import { Config, AgentConfig, ProviderConfig, ChannelConfig } from '../../bindings/github.com/yockii/wangshu/internal/config'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Server, Bot, MessageSquare, Wrench, Globe, Sparkles, Plus, Trash2, Save, FolderOpen } from '@lucide/vue'

const config = ref<Config | null>(null)
const originalConfig = ref<string>('')
const hasChanges = ref(false)
const activeSection = ref('providers')
const contentRef = ref<HTMLElement | null>(null)

const providerTypes = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'ollama', label: 'Ollama' },
]

const channelTypes = [
  { value: 'feishu', label: '飞书' },
  { value: 'web', label: 'Web' },
]

const providerNames = computed(() => {
  if (!config.value?.providers) return []
  return Object.keys(config.value.providers)
})

const agentNames = computed(() => {
  if (!config.value?.agents) return []
  return Object.keys(config.value.agents)
})

const sections = [
  { id: 'providers', label: 'Providers', icon: markRaw(Server) },
  { id: 'agents', label: 'Agents', icon: markRaw(Bot) },
  { id: 'channels', label: 'Channels', icon: markRaw(MessageSquare) },
  { id: 'skill', label: 'Skill', icon: markRaw(Wrench) },
  { id: 'browser', label: 'Browser', icon: markRaw(Globe) },
  { id: 'live2d', label: 'Live2D', icon: markRaw(Sparkles) },
]

const scrollToSection = (sectionId: string) => {
  const element = document.getElementById(`section-${sectionId}`)
  if (element && contentRef.value) {
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeSection.value = sectionId
  }
}

const handleScroll = () => {
  if (!contentRef.value) return
  const scrollTop = contentRef.value.scrollTop
  const sectionElements = sections.map(s => ({
    id: s.id,
    element: document.getElementById(`section-${s.id}`)
  })).filter(s => s.element)

  for (let i = sectionElements.length - 1; i >= 0; i--) {
    const section = sectionElements[i]
    if (section.element && section.element.offsetTop - 50 <= scrollTop) {
      activeSection.value = section.id
      break
    }
  }
}

const markChanged = () => {
  if (config.value && originalConfig.value) {
    hasChanges.value = JSON.stringify(config.value) !== originalConfig.value
  }
}

const saveConfig = async () => {
  if (!config.value) return
  try {
    await ConfigBundle.SaveConfig(config.value)
    originalConfig.value = JSON.stringify(config.value)
    hasChanges.value = false
  } catch (error) {
    console.error('Failed to save config:', error)
  }
}

const renameProvider = (oldName: string, newName: string) => {
  if (!config.value?.providers || oldName === newName || !newName.trim()) return
  const provider = config.value.providers[oldName]
  if (!provider) return
  delete config.value.providers[oldName]
  config.value.providers[newName] = provider
  markChanged()
}

const addProvider = () => {
  if (!config.value) return
  const name = `provider_${Date.now()}`
  if (!config.value.providers) {
    config.value.providers = {}
  }
  config.value.providers[name] = new ProviderConfig({
    type: 'openai',
    api_key: '',
    base_url: ''
  })
  markChanged()
}

const removeProvider = (name: string) => {
  if (!config.value?.providers) return
  delete config.value.providers[name]
  markChanged()
}

const renameAgent = (oldName: string, newName: string) => {
  if (!config.value?.agents || oldName === newName || !newName.trim()) return
  const agent = config.value.agents[oldName]
  if (!agent) return
  delete config.value.agents[oldName]
  config.value.agents[newName] = agent
  markChanged()
}

const addAgent = () => {
  if (!config.value) return
  const name = `agent_${Date.now()}`
  if (!config.value.agents) {
    config.value.agents = {}
  }
  config.value.agents[name] = new AgentConfig({
    provider: providerNames.value[0] || '',
    model: '',
    workspace: '~/.wangshu/workspace',
    temperature: 0.7,
    max_tokens: 0,
    enable_image_recognition: false,
    memory_organize_time: '00:00'
  })
  markChanged()
}

const removeAgent = (name: string) => {
  if (!config.value?.agents) return
  delete config.value.agents[name]
  markChanged()
}

const renameChannel = (oldName: string, newName: string) => {
  if (!config.value?.channels || oldName === newName || !newName.trim()) return
  const channel = config.value.channels[oldName]
  if (!channel) return
  delete config.value.channels[oldName]
  config.value.channels[newName] = channel
  markChanged()
}

const addChannel = () => {
  if (!config.value) return
  const name = `channel_${Date.now()}`
  if (!config.value.channels) {
    config.value.channels = {}
  }
  config.value.channels[name] = new ChannelConfig({
    type: 'feishu',
    enabled: false,
    agent: agentNames.value[0] || ''
  })
  markChanged()
}

const removeChannel = (name: string) => {
  if (!config.value?.channels) return
  delete config.value.channels[name]
  markChanged()
}

const selectSkillFolder = async () => {
  const folder = await ConfigBundle.SelectFolder('选择 Skills 全局路径', config.value?.skill?.global_path || '')
  if (folder && config.value) {
    config.value.skill.global_path = folder
    markChanged()
  }
}

const selectBrowserFolder = async () => {
  const folder = await ConfigBundle.SelectFolder('选择浏览器数据目录', config.value?.browser?.data_dir || '')
  if (folder && config.value) {
    config.value.browser.data_dir = folder
    markChanged()
  }
}

const live2dModels = ref<string[]>([])
const live2dScaleSlider = ref([100])

const loadLive2DModels = async () => {
  if (!config.value?.live2d?.model_dir) {
    live2dModels.value = []
    return
  }
  live2dModels.value = await ConfigBundle.GetModelList(config.value.live2d.model_dir) || []
}

const selectLive2DModelFolder = async () => {
  const folder = await ConfigBundle.SelectFolder('选择模型存放目录', config.value?.live2d?.model_dir || '')
  if (folder && config.value) {
    config.value.live2d.model_dir = folder
    await loadLive2DModels()
    markChanged()
  }
}

const onModelDirChange = async () => {
  markChanged()
  await loadLive2DModels()
}

const onScaleSliderChange = (value: number[] | undefined) => {
  if (config.value && value && value.length > 0) {
    config.value.live2d.scale = value[0]
    markChanged()
  }
}

onMounted(async () => {
  config.value = await ConfigBundle.GetConfig()
  console.log(config.value)
  if (config.value) {
    originalConfig.value = JSON.stringify(config.value)
    if (config.value.live2d?.scale) {
      live2dScaleSlider.value = [config.value.live2d.scale]
    }
    await loadLive2DModels()
  }
})
</script>
