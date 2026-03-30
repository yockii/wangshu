<script setup lang="ts">
import { Application, Ticker } from 'pixi.js';
import { Live2DSprite, Config, Priority } from 'easy-live2d';
import { onMounted, onUnmounted, ref } from 'vue';
import { Live2dBundle } from '../../../bindings/github.com/yockii/wangshu/internal/bundle';

const canvasRef = ref<HTMLCanvasElement>()

Config.MotionGroupIdle = 'Idle';// 设置默认的空闲动作组
Config.MouseFollow = false; // 禁用鼠标跟随

// 创建 Live2D 精灵
const live2DSprite = ref<Live2DSprite>()

onMounted(async () => {
  const app = new Application()
  await app.init({
    view: document.getElementById('live2d') as HTMLCanvasElement,
    backgroundAlpha: 0, // 透明
  })

  if (canvasRef.value) {
    // 获取模型文件路径
    const modelFile = await Live2dBundle.GetModelFile()
    if (modelFile == "") {
      console.error("模型文件路径为空")
      return
    }
    live2DSprite.value = new Live2DSprite()

    live2DSprite.value.init({
      modelPath: modelFile,
      ticker: Ticker.shared
    })

    // 监听点击事件
    live2DSprite.value.onLive2D('hit', ({ hitAreaName, x, y }) => {
      console.log('hit', hitAreaName, x, y)
    })

    live2DSprite.value.width = canvasRef.value.clientWidth * window.devicePixelRatio
    live2DSprite.value.height = canvasRef.value.clientHeight * window.devicePixelRatio

    app.stage.addChild(live2DSprite.value)

    live2DSprite.value.setExpression({
      expressionId: 'normal',
    })

    // 播放声音
    live2DSprite.value.playVoice({
      // 当前音嘴同步 仅支持wav格式
      voicePath: '/Resources/Hiyori/sounds/test3.wav',
    })

    // 停止声音
    // live2DSprite.stopVoice()

    setTimeout(() => {
      // 播放声音
      live2DSprite.value?.playVoice({
        voicePath: '/Resources/Hiyori/sounds/test.wav',
        immediate: true // 是否立即播放: 默认为true，会把当前正在播放的声音停止并立即播放新的声音
      })
    }, 10000)

    // live2DSprite.startMotion({
    //   group: 'test',
    //   no: 0,
    //   priority: 3,
    // })
  }
})

onUnmounted(() => {
  live2DSprite.value?.destroy()
})
</script>

<template>
  <canvas id="live2d" ref="canvasRef"></canvas>
</template>

<style scoped>
#live2d {
  width: 100vw;
  height: 100vh;
}
</style>

<style >
body {
  background-color: rgba(0, 0, 0, 0);
  margin: 0;
}
</style>