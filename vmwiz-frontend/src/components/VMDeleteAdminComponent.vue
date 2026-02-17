<template>
    <ConfirmationDialogComponent ref="confirmationDialog" />

    <div class="d-flex align-center">
        <v-text-field
            v-model="deleteHostname"
            label="Hostname"
            outlined
            dense
            class="mr-4"
        ></v-text-field>
        <v-btn color="error" @click="deleteVM()">
            <b>Delete VM</b>
        </v-btn>
        <v-checkbox
            v-model="deleteAlsoDNS"
            label="Also delete DNS entry"
        ></v-checkbox>
    </div>
    <p v-if="deleteMessage" class="mt-2">{{ deleteMessage }}</p>
</template>

<script>
import ConfirmationDialogComponent from "@/components/ConfirmationDialogComponent.vue";

export default {
    name: "VMDeleteAdminComponent",
    data() {
        return {
            deleteHostname: "",
            deleteAlsoDNS: true,
            confirmDialogOpen: false,
            deleteMessage: "",

            dialogLoading: false,
        };
    },
    components: {
        ConfirmationDialogComponent,
    },
    methods: {
        deleteVM() {
            this.dialogLoading = true;
            this.confirmDialogOpen = false;
            this.deleteMessage = "";
            this.$refs.confirmationDialog.showConfirmation(
                "POST",
                "/api/vm/deleteByName",
                {
                    "Content-Type": "application/json",
                },
                {
                    vmName: this.deleteHostname,
                    deleteDNS: this.deleteAlsoDNS,
                },
                () => {},
                async (response) => {
                    if (response.status != 200) {
                        this.deleteMessage = `Error: ${response.status}`;
                    } else {
                        this.deleteMessage = "VM deleted successfully.";
                    }
                },
            );
        },
    },
};
</script>

<style></style>
