import { createRouter, createWebHistory } from "vue-router";

const routes = [
    {
        meta: { title: "Admin Console - VMWiz" },
        path: "/console",
        name: "AdminView",
        component: () =>
            import(
                /* webpackChunkName: "about" */ "../views/AdminConsoleView.vue"
            ),
    },
    {
        meta: { title: "Request VM - VMWiz" },
        path: "/",
        name: "home",
        component: () =>
            import(/* webpackChunkName: "about" */ "../views/FormView.vue"),
    },
    {
        meta: { title: "VM Usage survey - VMWiz" },
        path: "/survey",
        name: "SurveyView",
        component: () => import("../views/PollView.vue"),
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach((to) => {
    document.title = to.meta.title || "VMWiz";
});

export default router;
