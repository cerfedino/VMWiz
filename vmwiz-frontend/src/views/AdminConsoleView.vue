<template>
    <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
        <h1 class="mt-3">Delete VM</h1>
        <VMDeleteAdminComponent />

        <h1>VM Usage surveys</h1>
        <SurveyAdminComponent />

        <h1 class="mt-3">VM Requests</h1>
        <VMRequestAdminComponent />

        <v-divider />
    </div>
</template>

<script>
import SurveyAdminComponent from "@/components/SurveyAdminComponent.vue";
import VMRequestAdminComponent from "@/components/VMRequestAdminComponent.vue";
import VMDeleteAdminComponent from "@/components/VMDeleteAdminComponent.vue";

export default {
    name: "AdminView",
    components: {
        SurveyAdminComponent,
        VMRequestAdminComponent,
        VMDeleteAdminComponent,
    },
    data() {
        return {};
    },
    methods: {},
};
</script>
