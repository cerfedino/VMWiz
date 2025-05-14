<template>
    <v-dialog v-model="dialogModel" class="w-50 h-50" :persistent="persistent">
        <v-card class="w-100 h-100 ma-auto" :loading="loading">
            <template v-slot:loader="{ isActive }">
                <v-progress-linear
                    :active="isActive"
                    :color="loaderColor"
                    height="4"
                    indeterminate
                ></v-progress-linear>
            </template>
            <template v-slot:title>
                <span class="font-weight-black">{{ title }}</span>
                <v-divider v-if="title" />
            </template>

            <template v-slot:actions>
                <slot name="actions">
                    <v-btn
                        v-if="!persistent"
                        class="ms-auto"
                        text="Close"
                        @click="dialogModel = false"
                    />
                </slot>
            </template>
            <v-card-text class="pt-4">
                <i v-if="loading"> Loading ... </i>
                <slot v-else name="content">
                    <pre v-if="content">{{ content }}</pre>
                    <i v-else>
                        <i>Empty content</i>
                    </i>
                </slot>
            </v-card-text>
        </v-card>
    </v-dialog>
</template>

<script>
export default {
    name: "DialogComponent",
    data() {
        return {};
    },
    emits: ["update:open"],
    computed: {
        dialogModel: {
            get() {
                return this.open;
            },
            set(val) {
                this.$emit("update:open", val);
            },
        },
    },
    props: {
        open: { type: Boolean },
        persistent: { type: Boolean, default: false },
        loading: { type: Boolean },
        loaderColor: { type: String, default: "primary" },
        title: { type: String },
        content: { type: String },
        onClose: { type: Function, default: () => {} },
    },
    methods: {},
};
</script>
