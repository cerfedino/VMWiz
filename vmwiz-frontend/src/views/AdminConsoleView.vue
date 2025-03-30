<template>
    <!-- <div class="h-screen d-flex flex-column justify-center"> -->
    <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
        <h1 class="text-h4 text-center font-weight-bold mb-3">VM Requests</h1>

        <div v-for="request in requests" :key="request.ID">
            <h1 class="text-h6 font-weight-bold mb-3">General Information</h1>

            <table>
                <tr>
                    <td>Request ID</td>
                    <td>{{ request.ID }}</td>
                </tr>
                <tr>
                    <td>Created</td>
                    <td>{{ request.CreatedAt }}</td>
                </tr>
                <tr>
                    <td>Email</td>
                    <td>{{ request.Email }}</td>
                </tr>

                <tr v-if="request.IsOrganization">
                    <td>Organization</td>
                    <td>{{ request.OrgName }}</td>
                </tr>
            </table>

            <h1 class="text-h6 font-weight-bold mt-4">VM specification</h1>

            <table>
                <tr>
                    <td>Hostname</td>
                    <td>{{ request.Hostname }}</td>
                </tr>
                <tr>
                    <td>OS Image</td>
                    <td>{{ request.Image }}</td>
                </tr>
                <tr>
                    <td>CPU Cores</td>
                    <td>{{ request.Cores }}</td>
                </tr>
                <tr>
                    <td>RAM (GB)</td>
                    <td>{{ request.RamGB }}</td>
                </tr>
                <tr>
                    <td>Disk Space (GB)</td>
                    <td>{{ request.DiskGB }}</td>
                </tr>
                <tr>
                    <td>SSH Public Key(s)</td>
                    <td>
                        <div
                            v-for="(key, index) in request.SshPubkeys"
                            :key="index"
                        >
                            {{ key }}
                        </div>
                    </td>
                </tr>
                <tr>
                    <td>Comments</td>
                    <td>{{ request.Comments }}</td>
                </tr>
            </table>

            <div class="d-flex flex-column">
                <v-btn class="mt-4" :color="submit_color" @click="submit(request.ID)">
                    <b>Accept request</b>
                </v-btn>
            </div>
        </div>
    </div>
    <!-- </div> -->
</template>

<script>

function submit(id) {
    fetch("/api/accept", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ id }),
    })
        .then((response) => {
            if (response.ok) {
                console.log("Request accepted");
            } else {
                console.error("Error accepting request");
            }
        })
        .catch((error) => {
            console.error("Error:", error);
        });
}
export default {
    name: "AdminView",
    data() {
        return {
            requests: [],
        };
    },
    methods: {},
    mounted() {
        this.$store.getters
            .fetchRequests()
            .then((response) => response.json())
            .then((data) => {
                this.$data.requests = data;

                console.log(data);

                // for (const [key, value] of Object.entries(data)) {
                //   this.requests[key] = value;
                // }
            });
    },
    components: {},
};
</script>
