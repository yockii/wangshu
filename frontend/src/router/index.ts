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
      }
    ]
  },
  {
    path: "/live2d",
    name: "live2d",
    component: () => import("@/views/live2d.vue"),
  }
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;