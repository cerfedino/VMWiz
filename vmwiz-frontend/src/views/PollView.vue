<template>
    <div>
        <h1>Poll Actions</h1>
            <p>Do you still need/use your Virtual Machine?</p>
            <button @click="keep">Yes</button>
            <button @click="showConfirmation = true">No</button>
            <div v-if="showConfirmation" class="confirmation-popup">
                <p>Are you sure you want to remove the service?</p>
                <button @click="confirmRemove">Yes</button>
                <button @click="showConfirmation = false">No</button>
            </div>
            <p v-if="submitted">Your response has been submitted.</p>
            <p v-if="error">An error occurred while submitting your response. Please try again and if it keeps on not working, please just write us a mail!</p>
    </div>
</template>

<script>
import axios from 'axios';
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
                const response = await axios.post('/api/poll/set', {
                    id: this.pollId,
                    keep: true,
                });
                if (response.status === 200) {
                    this.submitted = true; 
                }
            } catch (error) {
                this.error = true;
                console.error('Error submitting request:', error);
            }
        },
        async confirmRemove() {
            showConfirmation = false; 
            try {
                const response = await axios.post('/api/poll/set', {
                    id: this.pollId,
                    keep: false,
                });
                if (response.status === 200) {
                    this.submitted = true; 
                }
            } catch (error) {
                this.error = true;
                console.error('Error submitting request:', error);
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
