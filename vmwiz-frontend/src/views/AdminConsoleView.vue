<template>
    <v-dialog v-model="dialogOpen" class="w-50 h-50">
        <v-card class="w-100 h-100 ma-auto" :loading="dialogLoading">
            <template v-slot:loader="{ isActive }">
                <v-progress-linear
                    :active="isActive"
                    :color="dialogLoaderColor || 'primary'"
                    height="4"
                    indeterminate
                ></v-progress-linear>
            </template>
            <template v-slot:title>
                <span class="font-weight-black">{{ dialogTitle }}</span>
            </template>

            <template v-slot:actions>
                <v-btn
                    class="ms-auto"
                    text="Close"
                    @click="dialogOpen = false"
                ></v-btn>
            </template>
            <v-card-text class="bg-surface-light pt-4">
                <i v-if="dialogLoading"> Loading ... </i>
                <pre v-else-if="dialogContent">{{ dialogContent }}</pre>
                <i v-else> No content :P </i>
            </v-card-text>
        </v-card>
    </v-dialog>

    <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
        <h1>VM Usage surveys</h1>

        <h2 class="mt-3">Survey creation</h2>
        <v-btn @click="startSurvey">
            <b>Start Survey</b>
        </v-btn>
        <p v-if="clickCount < 3">
            Click {{ 3 - clickCount }} more time(s) to start a new survey.
        </p>
        <p v-else>Survey started!</p>

        <v-divider />

        <h2 class="mt-3">Open surveys</h2>
        <v-expansion-panels variant="accordion" multiple>
            <v-expansion-panel v-for="survey in surveys" :key="survey.surveyId">
                <v-expansion-panel-title>
                    Survey #{{ survey.surveyId }} -
                    {{
                        survey.date != undefined
                            ? new Date(survey.date).toDateString() +
                              " - " +
                              new Date(survey.date).toLocaleTimeString()
                            : "N/A"
                    }}
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                    <v-icon color="success" :icon="mdiAccountMultipleCheck" />
                    Positive responses:
                    <u
                        class="font-weight-bold cursor-grab"
                        @click="handlePositiveResponseDialog(survey.surveyId)"
                    >
                        {{
                            survey.positive != undefined
                                ? survey.positive
                                : "N/A"
                        }}
                    </u>
                    <br />
                    <v-icon color="error" :icon="mdiAccountMultipleRemove" />
                    Negative responses:
                    <u
                        class="font-weight-bold cursor-grab"
                        @click="handleNegativeResponseDialog(survey.surveyId)"
                    >
                        {{
                            survey.negative != undefined
                                ? survey.negative
                                : "N/A"
                        }}
                    </u>
                    <br />

                    <v-icon color="info" :icon="mdiAccountQuestion" />
                    Unanswered:
                    <u
                        class="font-weight-bold cursor-grab"
                        @click="handleNoneResponseDialog(survey.surveyId)"
                        >{{
                            survey.not_responded != undefined
                                ? survey.not_responded
                                : "N/A"
                        }}
                    </u>
                    <br />
                    <v-icon color="warning" :icon="mdiEmailAlert" />
                    Mails left to send:
                    <u
                        class="font-weight-bold cursor-grab"
                        @click="handleNotSentResponseDialog(survey.surveyId)"
                        >{{
                            survey.not_sent != undefined
                                ? survey.not_sent
                                : "N/A"
                        }}
                    </u>
                    <br />
                    <v-btn
                        class="mt-2"
                        color="primary"
                        variant="outlined"
                        @click="resendSurveyEmails(survey.surveyId)"
                    >
                        Resend to Unanswered & left to send
                    </v-btn>
                </v-expansion-panel-text>
            </v-expansion-panel>
        </v-expansion-panels>

        <h2 class="mt-3">VM Requests</h2>

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
import {
    mdiEmailAlert,
    mdiAccountMultipleCheck,
    mdiAccountMultipleRemove,
    mdiAccountQuestion,
} from "@mdi/js";

export default {
    name: "AdminView",
    data() {
        return {
            requests: [],
            clickCount: 0,

            surveys: [],
            surveyId: 0,
            surveyDataNeg: null,
            surveyDataNone: null,
            dialogOpen: false,
            dialogLoading: true,
            dialogLoaderColor: undefined,
            dialogContent: "Content",
            dialogTitle: "Title",

            mdiEmailAlert,
            mdiAccountMultipleCheck,
            mdiAccountMultipleRemove,
            mdiAccountQuestion,
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
                    storage_db: payload.DiskGB,
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
            this.$data.requests.sort(
                (a, b) =>
                    new Date(a.RequestCreatedAt) - new Date(b.RequestCreatedAt)
            );
        },

        // SURVEY FUNCTIONS
        async handleNegativeResponseDialog(id) {
            this.dialogLoading = true;
            this.dialogTitle = "Negative responses";
            this.dialogOpen = true;
            this.dialogLoaderColor = "success";
            this.dialogContent = (
                await this.getSurveyResponseNegative(id)
            ).join("\n");
            console.log(this.dialogContent);
            this.dialogLoading = false;
        },
        async handleNoneResponseDialog(id) {
            this.dialogLoading = true;
            this.dialogTitle = "Unanswered emails";
            this.dialogOpen = true;
            this.dialogLoaderColor = "success";
            let newcontent = (await this.getSurveyNoneResponse(id)).join("\n");
            console.log(this.dialogContent);
            this.dialogContent = newcontent;
            this.dialogLoading = false;
        },
        async handlePositiveResponseDialog(id) {
            this.dialogLoading = true;
            this.dialogTitle = "Positive responses";
            this.dialogOpen = true;
            this.dialogLoaderColor = "success";
            this.dialogContent = (
                await this.getSurveyResponsePositive(id)
            ).join("\n");
            console.log(this.dialogContent);
            this.dialogLoading = false;
        },
        async handleNotSentResponseDialog(id) {
            this.dialogLoading = true;
            this.dialogTitle = "Unsent emails";
            this.dialogOpen = true;
            this.dialogLoaderColor = "success";
            this.dialogContent = (await this.getSurveyNotSent(id)).join("\n");
            console.log(this.dialogContent);
            this.dialogLoading = false;
        },
        async populateSurveys() {
            let fetchedsurveys = [];

            let surveyIds = (await this.getAllSurveysIds()).surveyIds;
            console.log(surveyIds);
            for (let i = 0; i < surveyIds.length; i++) {
                let surveyId = surveyIds[i];
                fetchedsurveys.push(await this.getSurveyInfo(surveyId));
            }
            console.log(fetchedsurveys);
            // Sort ascending by creation date
            fetchedsurveys.sort((a, b) => new Date(a.sent) - new Date(b.sent));
        },

        startSurvey() {
            this.clickCount++;
            if (this.clickCount >= 3) {
                this.clickCount = 0;
                this.$store.getters.fetchBackend(
                    "/api/usagesurvey/start",
                    "GET"
                );
            }
        },
        getAllSurveysIds() {
            return this.$store.getters
                .fetchBackend("/api/usagesurvey/", "GET")
                .then((response) => response.json())
                .then((data) => {
                    console.log(data);
                    return data;
                });
        },
        getSurveyInfo(surveyId) {
            return this.$store.getters
                .fetchBackend(
                    "/api/usagesurvey/info?surveyId=" + surveyId,
                    "GET"
                )
                .then((response) => response.json())
                .then((data) => {
                    return data;
                });
        },

        getSurveyNoneResponse(id) {
            return this.$store.getters
                .fetchBackend(`/api/usagesurvey/responses/none?id=${id}`, "GET")
                .then((response) => response.json())
                .then((data) => {
                    return data;
                });
        },
        getSurveyResponseNegative(id) {
            return this.$store.getters
                .fetchBackend(
                    `/api/usagesurvey/responses/negative?id=${id}`,
                    "GET"
                )
                .then((response) => response.json())
                .then((data) => {
                    return data;
                });
        },
        getSurveyResponsePositive(id) {
            return this.$store.getters
                .fetchBackend(
                    `/api/usagesurvey/responses/positive?id=${id}`,
                    "GET"
                )
                .then((response) => response.json())
                .then((data) => {
                    return data;
                });
        },
        getSurveyNotSent(id) {
            return this.$store.getters
                .fetchBackend(
                    `/api/usagesurvey/responses/notsent?id=${id}`,
                    "GET"
                )
                .then((response) => response.json())
                .then((data) => {
                    return data;
                });
        },
        resendSurveyEmails(id) {
            return this.$store.getters.fetchBackend(
                `/api/usagesurvey/resend`,
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify({
                    id: id,
                })
            );
        },
    },

    async mounted() {
        await this.populateRequests();
        await this.populateSurveys();
    },
    components: {},
};
</script>
