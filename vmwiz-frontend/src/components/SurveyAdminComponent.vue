<template>
    <DialogComponent
        v-model:open="dialogOpen"
        :loading="dialogLoading"
        :loaderColor="dialogLoaderColor"
        :content="dialogContent"
        :title="dialogTitle"
    />

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
                    @click="handleSurveyPositiveResponseDialog(survey.surveyId)"
                >
                    {{ survey.positive != undefined ? survey.positive : "N/A" }}
                </u>
                <br />
                <v-icon color="error" :icon="mdiAccountMultipleRemove" />
                Negative responses:
                <u
                    class="font-weight-bold cursor-grab"
                    @click="handleSurveyNegativeResponseDialog(survey.surveyId)"
                >
                    {{ survey.negative != undefined ? survey.negative : "N/A" }}
                </u>
                <br />

                <v-icon color="info" :icon="mdiAccountQuestion" />
                Unanswered:
                <u
                    class="font-weight-bold cursor-grab"
                    @click="handleSurveyNoneResponseDialog(survey.surveyId)"
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
                    @click="handleSurveyNotSentResponseDialog(survey.surveyId)"
                    >{{
                        survey.not_sent != undefined ? survey.not_sent : "N/A"
                    }}
                </u>
                <br />
                <v-btn
                    class="mt-2"
                    color="primary"
                    variant="outlined"
                    @click="retryUnsentEmails(survey.surveyId)"
                >
                    Retry mails left to send
                </v-btn>
                <v-btn
                    class="mt-2"
                    color="primary"
                    variant="outlined"
                    @click="sendReminderEmail(survey.surveyId)"
                >
                    Send reminder to unanswered emails
                </v-btn>
            </v-expansion-panel-text>
        </v-expansion-panel>
    </v-expansion-panels>

    <v-divider />
</template>

<script>
import {
    mdiEmailAlert,
    mdiAccountMultipleCheck,
    mdiAccountMultipleRemove,
    mdiAccountQuestion,
} from "@mdi/js";

import DialogComponent from "@/components/DialogComponent.vue";

export default {
    name: "SurveyAdminComponent",
    data() {
        return {
            mdiEmailAlert,
            mdiAccountMultipleCheck,
            mdiAccountMultipleRemove,
            mdiAccountQuestion,

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
        };
    },
    props: {},
    components: {
        DialogComponent,
    },
    methods: {
        // SURVEY FUNCTIONS
        async handleSurveyNegativeResponseDialog(id) {
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
        async handleSurveyNoneResponseDialog(id) {
            this.dialogLoading = true;
            this.dialogTitle = "Unanswered emails";
            this.dialogOpen = true;
            this.dialogLoaderColor = "success";
            let newcontent = (await this.getSurveyNoneResponse(id)).join("\n");
            this.dialogContent = newcontent;
            console.log(this.dialogContent);
            this.dialogLoading = false;
        },
        async handleSurveyPositiveResponseDialog(id) {
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
        async handleSurveyNotSentResponseDialog(id) {
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
            this.surveys = fetchedsurveys;
        },

        startSurvey() {
            this.clickCount++;
            if (this.clickCount >= 3) {
                this.clickCount = 0;
                this.$store.getters.fetchBackend(
                    "/api/usagesurvey/create",
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
        retryUnsentEmails(id) {
            return this.$store.getters.fetchBackend(
                `/api/usagesurvey/resend/unsent`,
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify({
                    id: id,
                })
            );
        },
        sendReminderEmail(id) {
            return this.$store.getters.fetchBackend(
                `/api/usagesurvey/resend/unanswered`,
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
        await this.populateSurveys();
    },
};
</script>

<style></style>
