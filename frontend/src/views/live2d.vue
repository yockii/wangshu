<script setup lang="ts">
import { Application, Ticker } from 'pixi.js';
import { Live2DSprite, Config } from 'easy-live2d';
import { onMounted, onUnmounted, ref, computed, watch } from 'vue';
import { Live2dBundle } from '../../bindings/github.com/yockii/wangshu/internal/bundle';
import { Events } from '@wailsio/runtime';

const canvasRef = ref<HTMLCanvasElement>()
const pixiApp = ref<Application>()
const live2DSprite = ref<Live2DSprite>()

const editMode = ref(false)
const showControlPanel = ref(false)
const windowWidth = ref(200)
const windowHeight = ref(380)
const live2dConfig = ref<{
  enabled: boolean
  model_dir: string
  model_name: string
  width: number
  height: number
  x: number
  y: number
} | null>(null)

const modelList = ref<string[]>([])

Config.MotionGroupIdle = 'Idle'
Config.MouseFollow = false

const isModelLoaded = computed(() => live2DSprite.value !== undefined)

const toggleControlPanel = () => {
  showControlPanel.value = !showControlPanel.value
}

const handleResize = (deltaX: number, deltaY: number) => {
  const newWidth = Math.max(100, windowWidth.value + deltaX)
  const newHeight = Math.max(100, windowHeight.value + deltaY)
  windowWidth.value = newWidth
  windowHeight.value = newHeight
  Live2dBundle.UpdateLive2DWindowSize(newWidth, newHeight)
}

const onWindowResize = () => {
  if (pixiApp.value && canvasRef.value) {
    const width = window.innerWidth
    const height = window.innerHeight
    windowWidth.value = width
    windowHeight.value = height
    pixiApp.value.renderer.resize(width, height)
    updateSpriteSize()
  }
}

const updateSpriteSize = () => {
  if (live2DSprite.value && canvasRef.value) {
    live2DSprite.value.width = canvasRef.value.clientWidth * window.devicePixelRatio
    live2DSprite.value.height = canvasRef.value.clientHeight * window.devicePixelRatio
  }
}

const loadModel = async (modelName?: string) => {
  if (!pixiApp.value) return
  
  const name = modelName || live2dConfig.value?.model_name
  if (!name) return

  if (live2DSprite.value) {
    pixiApp.value.stage.removeChild(live2DSprite.value)
    live2DSprite.value.destroy()
    live2DSprite.value = undefined
  }

  const modelFile = await Live2dBundle.GetModelFile()
  if (modelFile === '') {
    console.error('模型文件路径为空')
    return
  }

  live2DSprite.value = new Live2DSprite()
  live2DSprite.value.init({
    modelPath: modelFile,
    ticker: Ticker.shared
  })

  live2DSprite.value.onLive2D('hit', ({ hitAreaName, x, y }) => {
    console.log('hit', hitAreaName, x, y)
  })

  updateSpriteSize()
  pixiApp.value.stage.addChild(live2DSprite.value)

  live2DSprite.value.setExpression({
    expressionId: 'normal',
  })
}

const loadModelList = async () => {
  if (!live2dConfig.value?.model_dir) {
    modelList.value = []
    return
  }
}

const exitEditMode = async () => {
  if (live2dConfig.value) {
    live2dConfig.value.width = windowWidth.value
    live2dConfig.value.height = windowHeight.value
    await Live2dBundle.SaveLive2DConfig()
  }
  Live2dBundle.ExitEditMode()
  showControlPanel.value = false
}

const handleEditModeChange = (isEdit: boolean) => {
  editMode.value = isEdit
  if (!isEdit) {
    showControlPanel.value = false
  }
}

let resizeStartX = 0
let resizeStartY = 0
let isResizing = false

const startResize = (e: MouseEvent) => {
  if (!editMode.value) return
  isResizing = true
  resizeStartX = e.clientX
  resizeStartY = e.clientY
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
}

const onResize = (e: MouseEvent) => {
  if (!isResizing) return
  const deltaX = e.clientX - resizeStartX
  const deltaY = e.clientY - resizeStartY
  resizeStartX = e.clientX
  resizeStartY = e.clientY
  handleResize(deltaX, deltaY)
}

const stopResize = () => {
  isResizing = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
}

onMounted(async () => {
  const app = new Application()
  await app.init({
    view: document.getElementById('live2d') as HTMLCanvasElement,
    backgroundAlpha: 0,
  })
  pixiApp.value = app

  const config = await Live2dBundle.GetLive2DConfig()
  live2dConfig.value = config
  if (config) {
    windowWidth.value = config.width || 200
    windowHeight.value = config.height || 380
  }

  const isEdit = await Live2dBundle.IsEditMode()
  editMode.value = isEdit

  Events.On('live2d-edit-mode', (data: { data: boolean }) => {
    handleEditModeChange(data.data)
  })

  if (canvasRef.value) {
    await loadModel()
  }

  window.addEventListener('resize', onWindowResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onWindowResize)
  live2DSprite.value?.destroy()
  pixiApp.value?.destroy()
})
</script>

<template>
  <div class="relative overflow-hidden w-full h-full" :style="{ border: editMode ? '2px solid rgba(255, 255, 255, 1)' : 'none' }">
    <canvas id="live2d" ref="canvasRef"></canvas>
    
    <div v-if="editMode" class="absolute top-0 left-0 right-0 bottom-0 pointer-events-none">
      <button 
        class="control-toggle" 
        @click="toggleControlPanel"
        :class="{ active: showControlPanel }"
      >
        ⚙️
      </button>
      
      <div v-if="showControlPanel" class="control-panel">
        <div class="panel-header">
          <span>编辑模式</span>
          <button class="exit-btn" @click="exitEditMode">完成编辑</button>
        </div>
        <div class="panel-content">
          <div class="size-info">
            <span>{{ windowWidth }} × {{ windowHeight }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
#live2d {
  width: 100%;
  height: 100%;
}

.edit-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.control-toggle {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.3);
  background: rgba(0, 0, 0, 0.5);
  color: white;
  cursor: pointer;
  pointer-events: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all 0.2s;
}

.control-toggle:hover {
  background: rgba(0, 0, 0, 0.7);
}

.control-toggle.active {
  background: rgba(59, 130, 246, 0.8);
}

.control-panel {
  position: absolute;
  top: 48px;
  right: 8px;
  width: 180px;
  background: rgba(0, 0, 0, 0.85);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  pointer-events: auto;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(59, 130, 246, 0.3);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  font-size: 12px;
  color: white;
}

.exit-btn {
  padding: 4px 8px;
  border-radius: 4px;
  border: none;
  background: #3b82f6;
  color: white;
  font-size: 11px;
  cursor: pointer;
  transition: background 0.2s;
}

.exit-btn:hover {
  background: #2563eb;
}

.panel-content {
  padding: 12px;
}

.size-info {
  text-align: center;
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
}

</style>

<style>
body {
  background-color: rgba(0, 0, 0, 0);
  margin: 0;
  overflow: hidden;
}
</style>
