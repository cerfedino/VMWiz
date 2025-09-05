<template>
    <DialogComponent
        v-model:open="dialogModel"
        :persistent="false"
        :loading="loading"
        :loaderColor="loaderColor"
        :show-content-placeholder="false"
        title="Confirm action"
    >
        <template v-slot:actions>
            <v-container class="pb-0">
                <v-divider horizontal class="mb-2" />
                <v-row no-gutters>
                    <p>
                        Type "<b> {{ previewPayload?.confirmationToken }}</b
                        >" to confirm
                    </p></v-row
                >
                <v-row no-gutters>
                    <v-text-field
                        v-model="inputConfirmation"
                        variant="outlined"
                        density="compact"
                        persistent-placeholder
                        placeholder=""
                        class="pa-0"
                        :rules="[() => error || true]"
                        @paste.prevent
                    >
                        <template v-slot:append-inner>
                            <v-divider vertical />
                            <v-icon
                                color="primary"
                                :icon="mdiArrowRightBold"
                                @click="confirm()"
                                size="large"
                                class="ml-2"
                            />
                        </template>
                    </v-text-field>
                </v-row>
            </v-container>
        </template>
        <template v-slot:content>
            <v-expansion-panels v-model="panels">
                <v-expansion-panel>
                    <v-expansion-panel-title>
                        Request details
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                        <p class="mb-2">
                            {{ URL }}
                        </p>
                        <pre class="mb-2"
                            >{{ JSON.stringify(body, null, 2) }}
                        </pre>
                    </v-expansion-panel-text>
                </v-expansion-panel>
            </v-expansion-panels>

            <template
                v-if="method == 'POST' && URL == '/api/usagesurvey/create'"
            >
                <!-- Show stuff depending on the request -->
            </template>
        </template>
    </DialogComponent>
</template>

<script>
import { ref } from "vue";

import DialogComponent from "@/components/DialogComponent.vue";

import { mdiArrowRightBold } from "@mdi/js";

export default {
    name: "ConfirmationDialogComponent",
    data() {
        return {
            mdiArrowRightBold,

            dialogModel: false,
            panels: ref([0]),

            loading: false,
            loaderColor: "primary",

            inputConfirmation: "",
            error: null,
            previewPayload: {},

            URL: "",
            method: "",
            headers: {},
            body: {},
            onClose: () => {},
            onSubmitEnd: () => {},
        };
    },
    emits: ["update:open"],
    methods: {
        async confirm() {
            this.loading = true;

            let request = await this.$store.getters.fetchBackend(
                this.URL,
                this.method,
                { ...this.headers },
                JSON.stringify({
                    ...this.body,
                    confirmationToken: this.inputConfirmation,
                })
            );
            if (request.status >= 400) {
                this.error = request.text();
                this.loading = false;
                return;
            }

            this.dialogModel = false;
            this.loading = false;

            this.onSubmitEnd(request);
        },
        async showConfirmation(
            reqMethod,
            reqURL,
            reqHeaders,
            reqBody,
            onClose = () => {},
            onSubmitEnd = () => {}
        ) {
            this.onClose = onClose;
            this.onSubmitEnd = onSubmitEnd;
            this.URL = reqURL;
            this.method = reqMethod;
            this.headers = reqHeaders;
            this.body = reqBody;
            this.inputConfirmation = "";
            this.error = null;
            this.loading = false;
            this.dialogModel = true;

            this.previewPayload = {};

            let url = new URL(this.$store.getters.buildBackendURL(reqURL));
            // Set url query param
            url.searchParams.set("preview", "true");
            this.previewPayload = await this.$store.getters
                .fetchBackend(
                    url.pathname + url.search,
                    this.method,
                    this.headers,
                    JSON.stringify(this.body)
                )
                .then((response) => {
                    return response.json();
                });

            this.loading = false;
        },
    },
    components: {
        DialogComponent,
    },
};
</script>
