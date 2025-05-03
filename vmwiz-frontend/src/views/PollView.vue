<template>
    <DialogComponent
        v-model:open="showSubmitted"
        :persistent="submitted && !error"
        :loading="submitLoading"
        :title="error ? 'Error' : submitted ? 'Thank you!' : undefined"
    >
        <template v-slot:content>
            <template v-if="submitted">
                <p class="text-center">
                    Your response has been submitted. You can close this window.
                </p>
            </template>
            <template v-if="error">
                <p class="text-center">
                    An error occurred while submitting your response.<br />Please
                    try again later. If the problem persists, please write us an
                    email !
                </p>
            </template>
        </template>
    </DialogComponent>

    <DialogComponent
        v-model:open="showConfirmation"
        :persistent="false"
        title="VM Deletion Confirmation"
    >
        <template v-slot:content>
            Are you sure you want to lose access to the following VM ?<br />
            <p class="text-center font-weight-bold mt-3">
                {{ hostname }}
            </p>
        </template>
        <template v-slot:actions>
            <v-btn text="Yes" @click="() => submitChoice(false)"></v-btn>
            <v-btn text="No" @click="showConfirmation = false"></v-btn>
        </template>
    </DialogComponent>

    <div class="w-75 pa-6 ma-auto align" style="max-width: 500px">
        <h1 class="text-center">VSOS VM Usage Survey</h1>
        <div class="text-center mt-8">
            You currently have the following VM with us:
            <p class="text-center font-weight-bold">{{ hostname }}</p>
            <br /><br />
            Do you still need/use your Virtual Machine ?
        </div>
        <div class="w-100 d-flex flex-row justify-center mt-4">
            <v-btn
                class="ma-3"
                variant="outlined"
                @click="() => (showConfirmation = true)"
            >
                <b>Yes</b>
            </v-btn>
            <v-btn
                class="ma-3"
                variant="outlined"
                @click="showConfirmation = true"
            >
                <b>No</b>
            </v-btn>
        </div>
    </div>
</template>

<script>
import DialogComponent from "@/components/DialogComponent.vue";

export default {
    data() {
        return {
            showConfirmation: false,
            showSubmitted: false,
            submitted: false,
            submitLoading: false,
            error: false,
            pollId: this.$route.query.id, //todo: show error if id is not set
            hostname: this.$route.query.hostname,
        };
    },
    components: {
        DialogComponent,
    },
    methods: {
        async submitChoice(keep) {
            this.showConfirmation = false;
            this.submitLoading = true;
            this.showSubmitted = true;
            try {
                const response = await this.$store.getters.fetchBackend(
                    "/api/usagesurvey/set",
                    "POST",
                    {},
                    JSON.stringify({
                        id: this.pollId,
                        keep: keep,
                    })
                );
                if (response.status < 200 || response.status >= 300) {
                    this.error = true;
                }
                if (response.status === 200) {
                    this.submitted = true;
                }
            } catch (error) {
                this.error = true;
                console.error("Error submitting request:", error);
            } finally {
                this.submitLoading = false;
            }
        },
    },
};
</script>
