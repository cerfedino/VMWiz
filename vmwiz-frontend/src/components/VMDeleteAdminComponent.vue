<template>
    <div class="d-flex align-center">
        <v-text-field
            v-model="deleteHostname"
            label="Hostname"
            outlined
            dense
            class="mr-4"
        ></v-text-field>
        <v-btn color="error" @click="confirmDialogOpen = true">
            <b>Delete VM</b>
        </v-btn>
        <v-checkbox
            v-model="deleteAlsoDNS"
            label="Also delete DNS entry"
        ></v-checkbox>
    </div>

    <DialogComponent
        v-model:open="confirmDialogOpen"
        :loading="dialogLoading"
        title="Confirm Deletion"
    >
        <template v-slot:content>
            Are you sure you want to delete the VM with hostname
            <b>{{ deleteHostname }}</b
            >?
        </template>
        <template v-slot:actions>
            <v-btn text @click="confirmDialogOpen = false">Cancel</v-btn>
            <v-btn color="error" text @click="deleteVM"> Confirm </v-btn>
        </template>
    </DialogComponent>
    <p v-if="deleteMessage" class="mt-2">{{ deleteMessage }}</p>
</template>

<script>
import DialogComponent from "@/components/DialogComponent.vue";

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
        DialogComponent,
    },
    methods: {
        deleteVM() {
            this.dialogLoading = true;
            this.deleteMessage = "";
            this.$store.getters
                .fetchBackend(
                    "/api/vm/deleteByName",
                    "POST",
                    {
                        "Content-Type": "application/json",
                    },
                    JSON.stringify({
                        vmName: this.deleteHostname,
                        deleteDNS: this.deleteAlsoDNS,
                    })
                )
                .then((response) => {
                    if (response.status != 200) {
                        this.deleteMessage = `Error: ${response.status}`;
                    } else {
                        this.deleteMessage = "VM deleted successfully.";
                    }
                    this.confirmDialogOpen = false;
                    this.dialogLoading = false;
                })
                .catch((error) => {
                    this.deleteMessage = `Error: ${error.message}`;
                    this.confirmDialogOpen = false;
                    this.dialogLoading = false;
                });
        },
    },
};
</script>

<style></style>
