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
  components,
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
    baseUrl: `${process.env.VUE_APP_VMWHIZ_SCHEME}://${process.env.VUE_APP_VMWHIZ_HOSTNAME}:${process.env.VUE_APP_VMWHIZ_PORT}`,
  },
  getters: {
    fetchVMOptions: (state) => () => {
      return fetch(`${state.baseUrl}/api/vmoptions`);
    },
    fetchSendVMRequest: (state) => (formData) => {
      return fetch(`${state.baseUrl}/api/vmrequest`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(formData),
      });
    },
  },
});

app.use(router).use(store).use(vuetify).mount("#app");
