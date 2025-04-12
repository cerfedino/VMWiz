<template>
    <div class="w-75 pa-6 ma-auto" style="max-width: 500px">
        <h1>Poll Actions</h1>
        <p>Do you still need/use your Virtual Machine?</p>
        <v-btn class="mt-4" @click="keep">
            <b>Yes</b>
        </v-btn>
        <v-btn class="mt-4" @click="showConfirmation = true">
            <b>No</b>
        </v-btn>
        <div v-if="showConfirmation" class="confirmation-popup ma-4">
            <p>Are you sure you want to remove the service?</p>
            <v-btn class="mt-4" @click="confirmRemove">
                <b>Yes</b>
            </v-btn>
            <v-btn class="mt-4" @click="showConfirmation = false">
                <b>No</b>
            </v-btn>
        </div>
        <p v-if="submitted">Your response has been submitted.</p>
        <p v-if="error">
            An error occurred while submitting your response. Please try again
            and if it keeps on not working, please just write us a mail!
        </p>
    </div>
</template>

<script>
import axios from "axios";
export default {
    data() {
        return {
            showConfirmation: false,
            submitted: false,
            error: false,
            pollId: this.$route.query.id, //todo: show error if id is not set
        };
    },
    methods: {
        async keep() {
            try {
                const response = await axios.post("/api/poll/set", {
                    id: this.pollId,
                    keep: true,
                });
                if (response.status === 200) {
                    this.submitted = true;
                }
            } catch (error) {
                this.error = true;
                console.error("Error submitting request:", error);
            }
        },
        async confirmRemove() {
            this.showConfirmation = false;
            try {
                const response = await axios.post("/api/poll/set", {
                    id: this.pollId,
                    keep: false,
                });
                if (response.status === 200) {
                    this.submitted = true;
                }
            } catch (error) {
                this.error = true;
                console.error("Error submitting request:", error);
            }
        },
        remove() {
            this.showConfirmation = true;
        },
    },
};
</script>

<style scoped>
.confirmation-popup {
    background: rgba(0, 0, 0, 0.5);
    padding: 20px;
    border-radius: 5px;
    color: white;
}
</style>
