import Vuex from "vuex";

const store = new Vuex.Store({
    state: {
        baseUrl: `${process.env.VUE_APP_VMWIZ_SCHEME}://${process.env.VUE_APP_VMWIZ_HOSTNAME}:${process.env.VUE_APP_VMWIZ_PORT}`,
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
            return getters.fetchBackend("/api/vmrequest/options", "GET", {
                "Content-Type": "application/json",
            });
        },
        fetchRequests: (state, getters) => () => {
            return getters.fetchBackend("/api/vmrequest", "GET", {
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

export default store;
