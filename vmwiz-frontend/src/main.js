import { createApp } from "vue";
import App from "./App.vue";
import router from "./router";
import Vuex from "vuex";

// Vuetify
import "vuetify/styles";
import { createVuetify } from "vuetify";
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";

const app = createApp(App);

import { aliases, mdi } from "vuetify/iconsets/mdi-svg";

const vuetify = createVuetify({
    components: {
        ...components,
    },
    directives,
    icons: {
        defaultSet: "mdi",
        aliases,
        sets: {
            mdi,
        },
    },
});

const store = new Vuex.Store({
    state: {
        baseUrl: `https://vmwiz.vsos.ethz.ch:443`,
    },
    getters: {
        fetchBackend: (state) => (fullPath, method, headers, body) => {
            return fetch(`${state.baseUrl}${fullPath}`, {
                method: method,
                headers: headers,
                body: body,
            }).then((response) => {
                if (response.status == 401) {
                    return response.json().then((data) => {
                        if (data.redirectUrl != undefined) {
                            window.location.href = data.redirectUrl;
                        }
                    });
                } else {
                    return response;
                }
            });
        },
        fetchVMOptions: (state, getters) => () => {
            return getters.fetchBackend("/api/vmoptions", "GET", {
                "Content-Type": "application/json",
            });
        },
        fetchRequests: (state, getters) => () => {
            return getters.fetchBackend("/api/requests", "GET", {
                "Content-Type": "application/json",
            });
        },
        fetchSendVMRequest: (state, getters) => (formData) => {
            return getters.fetchBackend(
                "/api/vmrequest",
                "POST",
                {
                    "Content-Type": "application/json",
                },
                JSON.stringify(formData)
            );
        },
    },
});

app.use(router).use(store).use(vuetify).mount("#app");
