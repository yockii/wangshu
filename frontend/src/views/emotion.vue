<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { Live2dBundle } from '../../bindings/github.com/yockii/wangshu/internal/bundle'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sparkles, Save, Play } from '@lucide/vue'
import { EmotionAction, EmotionMapping } from '../../bindings/github.com/yockii/wangshu/internal/types'

const EMOTIONS = [
  { key: 'happy', label: '开心', emoji: '😊' },
  { key: 'sad', label: '悲伤', emoji: '😢' },
  { key: 'angry', label: '愤怒', emoji: '😠' },
  { key: 'neutral', label: '中性', emoji: '😐' },
  { key: 'excited', label: '兴奋', emoji: '🤩' },
]

const modelName = ref('')
const motions = ref<EmotionAction[]>([])
const expressions = ref<string[]>([])
const mapping = ref<EmotionMapping>({ id: '', mappings: {} })
const hasChanges = ref(false)

const getMotionLabel = (motion: EmotionAction) => {
  const name = `动作${motion.motion_no || 0}`
  return `${motion.motion_group || ''} - ${name}`
}

const getMotionKey = (motion: EmotionAction) => {
  return `${motion.motion_group || ''}::${motion.motion_no || 0}`
}

const parseMotionKey = (key: string): { group: string; no: number } | null => {
  const parts = key.split('::')
  if (parts.length !== 2) return null
  return { group: parts[0], no: parseInt(parts[1], 10) }
}

const getSelectedMotionKey = (emotion: string) => {
  const action = mapping.value.mappings[emotion]
  if (!action?.motion_group) return ''
  return `${action.motion_group}::${action.motion_no || 0}`
}

const ensureMappingEntry = (emotion: string) => {
  if (!mapping.value.mappings) {
    mapping.value.mappings = {}
  }
  if (!mapping.value.mappings[emotion]) {
    mapping.value.mappings[emotion] = {}
  }
}

const updateMotionMapping = (emotion: string, motionKey: string) => {
  if (motionKey === '') {
    const entry = mapping.value.mappings[emotion]
    if (entry) {
      delete entry.motion_group
      delete entry.motion_no
      if (!entry.expression_id) {
        delete mapping.value.mappings[emotion]
      }
    }
  } else {
    const parsed = parseMotionKey(motionKey)
    if (!parsed) return
    ensureMappingEntry(emotion)
    mapping.value.mappings[emotion]!.motion_group = parsed.group
    mapping.value.mappings[emotion]!.motion_no = parsed.no
  }
  hasChanges.value = true
}

const updateExpressionMapping = (emotion: string, expressionId: string) => {
  if (expressionId === '') {
    const entry = mapping.value.mappings[emotion]
    if (entry) {
      delete entry.expression_id
      if (!entry.motion_group) {
        delete mapping.value.mappings[emotion]
      }
    }
  } else {
    ensureMappingEntry(emotion)
    mapping.value.mappings[emotion]!.expression_id = expressionId
  }
  hasChanges.value = true
}

const previewMotion = (group: string, no: number) => {
  Live2dBundle.PreviewMotion(group, no)
}

const previewExpression = (id: string) => {
  Live2dBundle.PreviewExpression(id)
}

const previewEmotionMotion = (emotion: string) => {
  const action = mapping.value.mappings[emotion]
  if (action?.motion_group) {
    previewMotion(action.motion_group, action.motion_no || 0)
  }
}

const previewEmotionExpression = (emotion: string) => {
  const action = mapping.value.mappings[emotion]
  if (action?.expression_id) {
    previewExpression(action.expression_id)
  }
}

const saveMapping = async () => {
  try {
    await Live2dBundle.SaveEmotionMapping(mapping.value)
    hasChanges.value = false
  } catch (error) {
    console.error('Failed to save emotion mapping:', error)
  }
}

onMounted(async () => {
  modelName.value = await Live2dBundle.GetCurrentModelName()
  const live2dMotions = await Live2dBundle.GetMotions()
  motions.value = live2dMotions.map((motion) => ({
    motion_group: motion.group,
    motion_no: motion.no,
  }))
  expressions.value = await Live2dBundle.GetExpressions()
  if (modelName.value) {
    const mappingVal = await Live2dBundle.GetEmotionMapping(modelName.value)
    if (mappingVal) {
        mapping.value = mappingVal
    } else {
        mapping.value = { id: modelName.value, mappings: {} }
    }
  }
})
</script>

<template>
  <div class="w-full h-full flex flex-col bg-background">
    <div class="flex-1 overflow-y-auto p-6">
      <div class="max-w-3xl mx-auto space-y-6">
        <div class="flex items-center gap-3">
          <Sparkles class="w-6 h-6 text-primary" />
          <h2 class="text-2xl font-bold">情感映射配置</h2>
        </div>

        <p class="text-sm text-muted-foreground">
          将系统定义的情感映射到 Live2D 精灵的动作和表情，以便通过情感驱动精灵动画。每个模型独立存储映射关系，切换模型后可直接复用。
        </p>

        <div v-if="!modelName" class="p-6 border border-border rounded-lg bg-muted/30 text-center">
          <p class="text-muted-foreground">尚未选择 Live2D 模型，请先在配置页面中选择模型。</p>
        </div>

        <div v-else class="space-y-4">
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <span>当前模型:</span>
            <span class="font-medium text-foreground">{{ modelName }}</span>
          </div>

          <div v-if="motions.length === 0 && expressions.length === 0" class="p-6 border border-border rounded-lg bg-muted/30 text-center">
            <p class="text-muted-foreground">当前模型尚未加载动作和表情数据。请先启动桌宠窗口加载模型。</p>
          </div>

          <div
            v-for="emotion in EMOTIONS"
            :key="emotion.key"
            class="p-4 border border-border rounded-lg bg-card"
          >
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xl">{{ emotion.emoji }}</span>
              <span class="font-medium">{{ emotion.label }}</span>
              <span class="text-xs text-muted-foreground">({{ emotion.key }})</span>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <div class="flex items-center justify-between">
                  <label class="text-sm text-muted-foreground">动作</label>
                  <Button
                    v-if="mapping.mappings[emotion.key]?.motion_group"
                    variant="ghost"
                    size="icon-sm"
                    @click="previewEmotionMotion(emotion.key)"
                  >
                    <Play class="w-3 h-3" />
                  </Button>
                </div>
                <Select
                  :modelValue="getSelectedMotionKey(emotion.key)"
                  @update:modelValue="(v: any) => updateMotionMapping(emotion.key, String(v))"
                >
                  <SelectTrigger class="w-full">
                    <SelectValue placeholder="未配置" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="motion in motions"
                      :key="getMotionKey(motion)"
                      :value="getMotionKey(motion)"
                    >
                      {{ getMotionLabel(motion) }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div class="space-y-2">
                <div class="flex items-center justify-between">
                  <label class="text-sm text-muted-foreground">表情</label>
                  <Button
                    v-if="mapping.mappings[emotion.key]?.expression_id"
                    variant="ghost"
                    size="icon-sm"
                    @click="previewEmotionExpression(emotion.key)"
                  >
                    <Play class="w-3 h-3" />
                  </Button>
                </div>
                <Select
                  :modelValue="mapping.mappings[emotion.key]?.expression_id || ''"
                  @update:modelValue="(v: any) => updateExpressionMapping(emotion.key, String(v))"
                >
                  <SelectTrigger class="w-full">
                    <SelectValue placeholder="未配置" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="expr in expressions"
                      :key="expr"
                      :value="expr"
                    >
                      {{ expr }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <div v-if="motions.length > 0" class="p-4 border border-border rounded-lg bg-muted/30">
            <h4 class="text-sm font-medium mb-3">动作预览</h4>
            <div class="flex flex-wrap gap-2">
              <Button
                v-for="motion in motions"
                :key="getMotionKey(motion)"
                variant="outline"
                size="sm"
                @click="previewMotion(motion.motion_group || '', motion.motion_no || 0)"
              >
                <Play class="w-3 h-3 mr-1" />
                {{ getMotionLabel(motion) }}
              </Button>
            </div>
          </div>

          <div v-if="expressions.length > 0" class="p-4 border border-border rounded-lg bg-muted/30">
            <h4 class="text-sm font-medium mb-3">表情预览</h4>
            <div class="flex flex-wrap gap-2">
              <Button
                v-for="expr in expressions"
                :key="expr"
                variant="outline"
                size="sm"
                @click="previewExpression(expr)"
              >
                <Play class="w-3 h-3 mr-1" />
                {{ expr }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="border-t border-border p-4 bg-background/95 backdrop-blur-sm">
      <div class="max-w-3xl mx-auto flex items-center justify-between">
        <span v-if="hasChanges" class="text-sm text-muted-foreground">有未保存的更改</span>
        <span v-else class="text-sm text-muted-foreground/50">无更改</span>
        <Button :disabled="!hasChanges" @click="saveMapping" :variant="hasChanges ? 'default' : 'outline'">
          <Save class="w-4 h-4 mr-2" />
          保存映射
        </Button>
      </div>
    </div>
  </div>
</template>
