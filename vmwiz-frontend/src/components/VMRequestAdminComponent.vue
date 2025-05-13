<template>
    <div
        v-for="request in requests"
        :key="request.ID"
        :class="{
            'opacity-50 pointer-events-none':
                request.RequestStatus != 'pending',
        }"
    >
        <h1 class="text-h6 font-weight-bold mb-3">General Information</h1>

        <table>
            <tbody>
                <tr>
                    <td>Status</td>
                    <td>{{ request.RequestStatus }}</td>
                </tr>
                <tr>
                    <td>Request ID</td>
                    <td>{{ request.ID }}</td>
                </tr>
                <tr>
                    <td>Created At</td>
                    <td>{{ request.RequestCreatedAt }}</td>
                </tr>
                <tr>
                    <td>Email</td>
                    <td>{{ request.Email }}</td>
                </tr>
                <tr>
                    <td>Personal Email</td>
                    <td>{{ request.PersonalEmail }}</td>
                </tr>

                <tr v-if="request.IsOrganization">
                    <td>Organization</td>
                    <td>{{ request.OrgName }}</td>
                </tr>
            </tbody>
        </table>

        <h1 class="text-h6 font-weight-bold mt-6">VM specification</h1>

        <table>
            <tbody>
                <tr>
                    <td>OS Image</td>
                    <td>{{ request.Image }}</td>
                </tr>
                <tr>
                    <td>Hostname</td>
                    <td>
                        <input
                            v-model="request.Hostname"
                            type="text"
                            class="form-control"
                        />
                    </td>
                </tr>
                <tr>
                    <td>CPU Cores</td>
                    <td>
                        <input
                            v-model.number="request.Cores"
                            type="number"
                            class="form-control"
                        />
                    </td>
                </tr>
                <tr>
                    <td>RAM (GB)</td>
                    <td>
                        <input
                            v-model.number="request.RamGB"
                            type="number"
                            class="form-control"
                        />
                    </td>
                </tr>
                <tr>
                    <td>Disk Space (GB)</td>
                    <td>
                        <input
                            v-model.number="request.DiskGB"
                            type="number"
                            class="form-control"
                        />
                    </td>
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
            </tbody>
        </table>

        <div class="d-flex flex-column">
            <v-btn class="mt-4" @click="acceptRequest(request.ID)">
                <b>Accept request</b>
            </v-btn>
            <v-btn
                class="mt-4"
                @click="
                    editRequest(request.ID, {
                        Hostname: request.Hostname,
                        Cores: request.Cores,
                        RamGB: request.RamGB,
                        DiskGB: request.DiskGB,
                    })
                "
            >
                <b>Edit request</b>
            </v-btn>
            <v-btn class="mt-4" @click="rejectRequest(request.ID)">
                <b>Reject request</b>
            </v-btn>
        </div>
    </div>
</template>

<script>
export default {
    name: "VMRequestAdminComponent",
    data() {
        return {
            requests: [],
        };
    },
    props: {},
    methods: {
        acceptRequest(id) {
            this.$store.getters.fetchBackend(
                "/api/vmrequest/accept",
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify({
                    id: id,
                })
            );
            this.populateRequests();
        },
        rejectRequest(id) {
            this.$store.getters.fetchBackend(
                "/api/vmrequest/reject",
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify({
                    id: id,
                })
            );
            this.populateRequests();
        },
        editRequest(id, payload) {
            this.$store.getters.fetchBackend(
                "/api/vmrequest/edit",
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify({
                    id: id,
                    cores_cpu: payload.Cores,
                    ram_gb: payload.RamGB,
                    storage_gb: payload.DiskGB,
                })
            );
            this.populateRequests();
        },
        async populateRequests() {
            let data = await this.$store.getters
                .fetchRequests()
                .then((response) => response.json());
            this.$data.requests = data;
            // Sort ascending by creation date
            this.$data.requests.sort((a, b) => {
                if (
                    a.RequestStatus == "pending" &&
                    b.RequestStatus != "pending"
                ) {
                    return -1;
                } else if (
                    b.RequestStatus == "pending" &&
                    a.RequestStatus != "pending"
                ) {
                    return 1;
                } else {
                    return (
                        new Date(a.RequestCreatedAt) -
                        new Date(b.RequestCreatedAt)
                    );
                }
            });
        },
    },
    async mounted() {
        await this.populateRequests();
    },
};
</script>

<style></style>
