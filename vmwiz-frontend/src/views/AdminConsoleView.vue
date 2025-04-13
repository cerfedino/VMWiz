<template>
    <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
        <h1 class="text-h4 text-center font-weight-bold mb-3">VM Requests</h1>

        <v-btn @click="startSurvey">
            <b>Start Survey</b>
        </v-btn>
        <p v-if="clickCount < 3">
            Click {{ 3 - clickCount }} more time(s) to start the survey.
        </p>
        <p v-else>Survey started!</p>

        <div class="ma-auto pa-2 mb-4">
            <div>
                <v-text-field
                    class="ma-auto pa-2"
                    label="Last Survey ID:"
                    v-model="surveyId"
                    outlined
                ></v-text-field>
            </div>
            <v-btn @click="getSurveyNoneResponse(surveyId)">
                <b>Get None Responses</b>
            </v-btn>
            <v-btn @click="getSurveyResponseNegative(surveyId)">
                <b>Get Negative Responses</b>
            </v-btn>
        </div>

        <div v-for="request in requests" :key="request.ID">
            <h1 class="text-h6 font-weight-bold mb-3">General Information</h1>

            <table>
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
            </table>

            <h1 class="text-h6 font-weight-bold mt-6">VM specification</h1>

            <table>
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
            </table>

            <div class="d-flex flex-column">
                <v-btn
                    class="mt-4"
                    :color="submit_color"
                    @click="acceptRequest(request.ID)"
                >
                    <b>Accept request</b>
                </v-btn>
                <v-btn
                    class="mt-4"
                    :color="edit_color"
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
                <v-btn
                    class="mt-4"
                    :color="reject_color"
                    @click="rejectRequest(request.ID)"
                >
                    <b>Reject request</b>
                </v-btn>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    name: "AdminView",
    data() {
        return {
            requests: [],
            clickCount: 0,
            surveyId: 0,
        };
    },
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
                    storage_db: payload.DiskGB,
                })
            );
        },
        startSurvey() {
            this.clickCount++;
            if (this.clickCount >= 3) {
                this.$store.getters.fetchBackend("/api/poll/start", "GET");
            }
        },
        getLastSurveyId() {
            this.$store.getters
                .fetchBackend("/api/poll/lastsurvey", "GET")
                .then((response) => response.json())
                .then((data) => {
                    this.surveyId = data;
                });
        },
        getSurveyNoneResponse(id) {
            this.$store.getters
                .fetchBackend(`/api/poll/responses/none?id=${id}`, "GET")
                .then((response) => response.json())
                .then((data) => {
                    console.log(data);
                    //todo: display data in a table
                });
        },
        getSurveyResponseNegative(id) {
            this.$store.getters
                .fetchBackend(`/api/poll/responses/negative?id=${id}`, "GET")
                .then((response) => response.json())
                .then((data) => {
                    console.log(data);
                    //todo: display data in a table
                });
        },
    },

    mounted() {
        this.$store.getters
            .fetchRequests()
            .then((response) => response.json())
            .then((data) => {
                this.$data.requests = data;
                console.log(data);
            });
        this.getLastSurveyId();
    },
    components: {},
};
</script>
