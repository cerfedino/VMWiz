import { createRouter, createWebHashHistory } from "vue-router";
import FormView from "@/views/FormView.vue";
import AdminConsoleView from "@/views/AdminConsoleView.ue";

const routes = [
  {
    path: "/",
    name: "home",
    component: FormView,
  },
  {
    path: "/console/",
    name: "admin",
    component: AdminConsoleView,
  },
  // {
  //   path: "/about",
  //   name: "about",
  //   // route level code-splitting
  //   // this generates a separate chunk (about.[hash].js) for this route
  //   // which is lazy-loaded when the route is visited.
  //   component: () =>
  //     import(/* webpackChunkName: "about" */ "../views/AboutView.vue"),
  // },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
