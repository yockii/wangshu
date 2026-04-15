import { createRouter, createWebHashHistory } from "vue-router";

const routes = [
  {
    path: "/",
    redirect: "/chat",
    component: () => import("@/layout/main.vue"),
    children: [
      {
        path: "/chat",
        name: "chat",
        meta: {
          title: "聊天",
        },
        component: () => import("@/views/chat.vue"),
      },
      {
        path: "/config",
        name: "config",
        meta: {
          title: "配置",
        },
        component: () => import("@/views/config.vue"),
      },
      {
        path: "/emotion",
        name: "emotion",
        meta: {
          title: "情感映射",
        },
        component: () => import("@/views/emotion.vue"),
      }
    ]
  },
  {
    path: "/live2d",
    name: "live2d",
    meta: {
      title: "桌宠",
    },
    component: () => import("@/views/live2d.vue"),
  },
  {
    path: "/qrcode",
    name: "qrcode",
    meta: {
      title: "微信登录",
    },
    component: () => import("@/views/qrcode.vue"),
  }
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;