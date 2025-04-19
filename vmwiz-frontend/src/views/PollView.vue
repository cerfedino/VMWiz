<template>
    <v-dialog :persistent="submitted" v-model="showSubmitted" max-width="50%">
        <v-card class="w-100 h-100 ma-auto">
            <v-card-text>
                <template v-if="submitted">
                    <h2 class="text-center">Thank you!</h2>
                    <p class="text-center mt-8 mb-8">
                        Your response has been submitted. You can close this
                        window.
                    </p>
                </template>
                <template v-if="error">
                    <h1 class="text-center">Error</h1>
                    <p class="text-center mt-8 mb-8">
                        An error occurred while submitting your response.<br />Please
                        try again later. If the problem persists, please write
                        us an email !
                    </p>
                </template>
            </v-card-text>
            <template v-slot:actions v-if="error">
                <v-btn
                    class="mt-4"
                    variant="outlined"
                    @click="showSubmitted = false"
                >
                    Close
                </v-btn>
            </template>
        </v-card>
    </v-dialog>
    <v-dialog v-model="showConfirmation" max-width="50%">
        <v-card :loading="dialogLoading" class="w-100 h-100 ma-auto">
            <template v-slot:loader="{ isActive }">
                <v-progress-linear
                    :active="isActive"
                    :color="dialogLoaderColor || 'primary'"
                    height="4"
                    indeterminate
                ></v-progress-linear>
            </template>
            <template v-slot:title>
                <span class="font-weight-black">VM Deletion Confirmation</span>
            </template>

            <template v-slot:actions>
                <div class="w-100 d-flex flex-row justify-center">
                    <v-btn
                        text="Yes"
                        @click="() => submitChoice(false)"
                    ></v-btn>
                    <v-btn text="No" @click="showConfirmation = false"></v-btn>
                </div>
            </template>
            <v-card-text>
                <div class="mt-4">
                    Are you sure you want to lose access to the following VM
                    ?<br />
                    <p class="text-center font-weight-bold mt-3">
                        {{ hostname }}
                    </p>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>

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
                @click="() => submitChoice(true)"
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
export default {
    data() {
        return {
            showConfirmation: false,
            showSubmitted: false,
            submitted: false,
            error: false,
            pollId: this.$route.query.id, //todo: show error if id is not set
            hostname: this.$route.query.hostname,
        };
    },
    methods: {
        async submitChoice(keep) {
            this.showConfirmation = false;
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
                    this.showSubmitted = true;
                }
                if (response.status === 200) {
                    this.submitted = true;
                    this.showSubmitted = true;
                }
            } catch (error) {
                this.error = true;
                this.showSubmitted = true;
                console.error("Error submitting request:", error);
            }
        },
        remove() {
            this.showConfirmation = true;
        },
    },
};
</script>
