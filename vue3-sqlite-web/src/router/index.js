import { createRouter, createWebHistory } from 'vue-router';
import Layout from "@/views/Layout.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'root',
      component: Layout,
      redirect: { name: "home" },
      children: [
        {
          name: "home",
          path: '/',
          component: () => import('@/views/home/Home.vue'),
          redirect: { name: "database" },
          meta: {},
          children: [
            {
              name: "database",
              path: "/database",
              component: () => import('@/views/database/Database.vue'),
            },
            {
              name: "query",
              path: "/query",
              component: () => import("@/views/query/Query.vue"),
            },
            {
              name: "table",
              path: "/:table/:action?",
              component: () => import("@/views/table/Table.vue"),
            },
          ]
        },
      ],
    }
  ],
});

export default router;
