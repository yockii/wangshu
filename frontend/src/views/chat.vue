<template>
  <div class="w-full h-full flex flex-col justify-between px-4 py-2 wails-draggable">
    <div class="w-50 absolute top-4 left-1/2 -translate-y-1/2">
      <OpacitySlider />
    </div>

    <div ref="chatContainer" class="flex-1 overflow-y-auto p-4 space-y-4">
      <MessageItem v-for="msg in messages" :key="msg.id" :content="msg.content" :is-user="msg.isUser" />
    </div>

    <div class="mt-2">
      <InputGroup class="wails-nodraggable">
        <InputGroupTextarea placeholder="和我聊聊吧" v-model="msgContent" :disabled="inputDisabled"
          @keydown="handleInputKeydown" />
        <InputGroupAddon align="block-end">
          <!-- <InputGroupButton variant="outline" class="rounded-full" size="icon-xs">
            <PlusIcon class="size-4" />
          </InputGroupButton> -->
          <InputGroupText class="ml-auto">
            {{ sessionPercent }}% used
          </InputGroupText>
          <Separator orientation="vertical" class="!h-4" />
          <InputGroupButton variant="default" class="rounded-full cursor-pointer" size="icon-xs"
            :disabled="inputDisabled" @click="sendMessage">
            <ArrowUpIcon class="size-4" />
            <span class="sr-only">Send</span>
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import OpacitySlider from '@/components/OpacitySlider.vue'
import { ArrowUpIcon, PlusIcon } from '@lucide/vue'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput, InputGroupText, InputGroupTextarea } from '@/components/ui/input-group'
import { Separator } from '@/components/ui/separator'
import { nextTick, onMounted, ref, shallowRef } from 'vue';
import type { Message } from '@/types/message'
import MessageItem from '@/components/MessageItem.vue'
import { ChatBundle, WindowBundle } from '../../bindings/github.com/yockii/wangshu/internal/bundle';
import { Events, Window } from "@wailsio/runtime";
import type { Message as BusMessage } from '../../bindings/github.com/yockii/wangshu/pkg/bus';

// 使用 shallowRef 优化性能，避免深度监听整个消息数组
const messages = shallowRef<Message[]>([])
const sessionPercent = ref(0)
const chatContainer = ref<HTMLDivElement>()

const msgContent = ref('')
const inputDisabled = ref(false)
const sendMessage = async () => {
  if (!msgContent.value) {
    return
  }
  inputDisabled.value = true
  try {
    await ChatBundle.HandleMessage(msgContent.value)
    messages.value = [...messages.value, { id: Date.now(), content: msgContent.value, isUser: true }]
    msgContent.value = ''
  } finally {
    inputDisabled.value = false
  }
  // 滚动到最新消息
  scrollToBottom()
}

const handleInputKeydown = (e: KeyboardEvent) => {
  const isEnterKey = e.key === 'Enter'
  const isShortcutKey = e.ctrlKey || e.metaKey || e.altKey || e.shiftKey
  if (isEnterKey) {
    if (isShortcutKey) {
      // 获取光标位置
      const ce = e.target as HTMLTextAreaElement
      const cursorPosition = ce?.selectionStart || 0
      const textPrefix = ce?.value.slice(0, cursorPosition) || ''
      const textSuffix = ce?.value.slice(cursorPosition) || ''
      const newText = textPrefix + '\n' + textSuffix
      ce.value = newText
      nextTick(() => {
        ce.setSelectionRange(cursorPosition + 1, cursorPosition + 1)
      })
    } else {
      sendMessage()
    }
    return
  }
}

let streamingMessage: Message | null = null
let contentBuffer = ''
let isThrottling = false

// 模拟从大模型接收数据流
function handleStreamChunk(chunk) {
  contentBuffer += chunk // 1. 累积内容到缓冲区

  // 2. 使用 requestAnimationFrame 进行节流
  if (!isThrottling) {
    isThrottling = true
    requestAnimationFrame(() => {
      // 3. 在一帧内，用缓冲区的完整内容更新消息
      if (streamingMessage) {
        streamingMessage.content = contentBuffer
      }
      // 4. 手动触发 shallowRef 的更新
      messages.value = [...messages.value]
      isThrottling = false
    })
  }
}

async function startChat() {
  const userMessage = { id: Date.now(), content: '你好', isUser: true }
  const aiMessage = { id: Date.now() + 1, content: '', isUser: false }

  messages.value = [...messages.value, userMessage, aiMessage]
  streamingMessage = aiMessage
  contentBuffer = ''

  // 模拟接收流式数据
  const mockStream = ['你好', '，我是', '**', 'AI', '**', '助手。']
  for (const chunk of mockStream) {
    handleStreamChunk(chunk)
    await new Promise(r => setTimeout(r, 100)) // 模拟网络延迟
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

const updateLocation = async () => {
  if ("geolocation" in navigator) {
    navigator.geolocation.getCurrentPosition((position) => {
      const coords = position.coords
      WindowBundle.UpdateGeoLocation(
        coords.latitude + '',
        coords.longitude + '',
        coords.altitude + '',
        coords.accuracy + '',
        coords.altitudeAccuracy + '',
        coords.heading + '',
        coords.speed + '',
      )
    })
  }
}

onMounted(async () => {
  updateLocation()
  Events.On('chat-message', (msg: { data: BusMessage }) => {
    messages.value = [...messages.value, { id: Date.now(), content: msg.data.content, isUser: false }]
    inputDisabled.value = false
    sessionPercent.value = (msg.data.metadata.session_percent || 0) * 100

    // 滚动到最新消息
    scrollToBottom()
  });

  // 获取历史消息
  const msgs = await ChatBundle.GetHistoryMessages(0, 100)
  for (const msg of msgs) {
    messages.value = [...messages.value, { id: Math.random(), content: msg.Content, isUser: msg.Role == 'user' }]
  }
  scrollToBottom()
})
</script>

<style>
#app {
  border: 1px solid oklch(0 0 0 / 50%);
}
</style>