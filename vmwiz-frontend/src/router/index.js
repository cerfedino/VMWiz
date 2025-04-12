import { createRouter, createWebHistory } from "vue-router";

const routes = [
    {
        path: "/console",
        name: "AdminView",
        component: () =>
            import(
                /* webpackChunkName: "about" */ "../views/AdminConsoleView.vue"
            ),
    },
    {
        path: "/",
        name: "home",
        component: () =>
            import(/* webpackChunkName: "about" */ "../views/FormView.vue"),
    },
    {
        path: "/survey",
        name: "SurveyView",
        component: () => import("../views/PollView.vue"),
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

export default router;
