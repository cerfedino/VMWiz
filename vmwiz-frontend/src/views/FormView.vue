<template>
  <!-- <div class="h-screen d-flex flex-column justify-center"> -->
  <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
    <h1 class="text-h4 text-center font-weight-bold mb-3">VM Request Form</h1>
    <p class="text-center">
      <v-icon
        :class="isFormModified() ? 'opacity-100' : 'opacity-0'"
        color="error"
        :icon="mdiBackspace"
        @click="resetForm"
      />
    </p>

    <v-form ref="form">
      <h1 class="text-h6 font-weight-bold mb-3">General Information</h1>
      <v-text-field
        class="mb-3"
        :prepend-inner-icon="mdiEmailOutline"
        label="ETH E-Mail address"
        type="email"
        variant="outlined"
        density="compact"
        v-model="form_values.current.email"
        :rules="[() => form_values.validation_errors.email || true]"
      />

      <v-text-field
        :prepend-inner-icon="mdiEmailOutline"
        label="Non-ETH E-Mail address"
        type="email"
        variant="outlined"
        density="compact"
        v-model="form_values.current.personalEmail"
        :rules="[() => form_values.validation_errors.personalEmail || true]"
      />

      <v-checkbox
        v-model="form_values.current.isOrganization"
        variant="outlined"
        density="compact"
        hide-details="true"
        label="Are you requesting this VM on behalf of an organization?"
      />

      <v-text-field
        v-show="form_values.current.isOrganization"
        :prepend-inner-icon="mdiOfficeBuildingOutline"
        label="Organization Name"
        variant="outlined"
        density="compact"
        v-model="form_values.current.orgName"
        :rules="[() => form_values.validation_errors.orgName || true]"
      />

      <h1 class="text-h6 font-weight-bold mt-4">VM specification</h1>
      <h1 class="text-subtitle-1 mt-3">Hostname</h1>
      <v-text-field
        persistent-placeholder
        placeholder="my-vm"
        :prepend-inner-icon="mdiLink"
        suffix=".vsos.ethz.ch"
        variant="outlined"
        density="compact"
        v-model="form_values.current.hostname"
        :rules="[() => form_values.validation_errors.hostname || true]"
      />
      <h1 class="text-subtitle-1">OS Image</h1>
      <v-select
        variant="outlined"
        density="compact"
        :items="form_values.allowed.image"
        v-model="form_values.current.image"
        :rules="[() => form_values.validation_errors.image || true]"
      >
        <template v-slot:item="{ props }">
          <v-list-item v-bind="props">
            <template v-slot:prepend>
              <v-icon class="mr-2" :icon="mdiPenguin" />
            </template>
          </v-list-item>
        </template>
        <template v-slot:selection="{ item }">
          <v-icon class="mr-2" :icon="mdiPenguin" />
          {{ item.title }}
        </template>
      </v-select>

      <h1 class="text-subtitle-1">CPU Cores</h1>
      <v-slider
        show-ticks="always"
        step="1"
        thumb-label
        thumb-size="16"
        tick-size="4"
        variant="outlined"
        density="compact"
        color="error"
        v-model="form_values.current.cores"
        :max="form_values.allowed.cores.max"
        :min="form_values.allowed.cores.min"
        :rules="[() => form_values.validation_errors.cores || true]"
      >
        <template v-slot:prepend>
          <v-text-field
            style="width: 80px"
            type="number"
            hide-details
            single-line
            variant="outlined"
            density="compact"
            v-model="form_values.current.cores" /></template
      ></v-slider>

      <h1 class="text-subtitle-1">RAM (GB)</h1>
      <v-slider
        show-ticks="always"
        step="1"
        thumb-label
        thumb-size="16"
        tick-size="4"
        variant="outlined"
        density="compact"
        color="warning"
        v-model="form_values.current.ramGB"
        :max="form_values.allowed.ramGB.max"
        :min="form_values.allowed.ramGB.min"
        :rules="[() => form_values.validation_errors.ramGB || true]"
      >
        <template v-slot:prepend>
          <v-text-field
            style="width: 80px"
            type="number"
            hide-details
            single-line
            variant="outlined"
            density="compact"
            v-model="form_values.current.ramGB"
          />
        </template>
      </v-slider>

      <h1 class="text-subtitle-1">Disk Space (GB)</h1>
      <v-slider
        step="1"
        thumb-label
        thumb-size="16"
        tick-size="4"
        variant="outlined"
        density="compact"
        color="success"
        v-model="form_values.current.diskGB"
        :max="form_values.allowed.diskGB.max"
        :min="form_values.allowed.diskGB.min"
        :rules="[() => form_values.validation_errors.diskGB || true]"
      >
        <template v-slot:prepend>
          <v-text-field
            style="width: 80px"
            type="number"
            hide-details
            variant="outlined"
            density="compact"
            v-model="form_values.current.diskGB"
          />
        </template>
      </v-slider>

      <h1 class="text-subtitle-1 pb-3">
        SSH Public Key(s)
        <!-- <v-tooltip text="Tooltip">
          <template v-slot:activator="{ props }">
            <v-icon v-bind="props" :icon="mdiInformationOutline" />
          </template>
          {{ form_values.tooltips.sshPubkey }}
        </v-tooltip> -->
        <v-icon
          :icon="mdiPlusBoxOutline"
          @click="form_values.current.sshPubkey.push('')"
        />
        <p class="text-caption text-error">
          {{
            form_values.current.sshPubkey.length != 0
              ? Array.isArray(form_values.validation_errors.sshPubkey)
                ? form_values.validation_errors.sshPubkey.join("\n")
                : form_values.validation_errors.sshPubkey
              : ""
          }}
        </p>
      </h1>
      <div v-for="(key, index) in form_values.current.sshPubkey" :key="index">
        <v-text-field
          v-model="form_values.current.sshPubkey[index]"
          variant="outlined"
          density="compact"
          persistent-placeholder
          placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCq..."
          :rules="[
            () => form_values.validation_errors.sshPubkey[index] || true,
          ]"
        >
          <template v-slot:prepend>
            <v-icon
              :icon="mdiMinusBoxOutline"
              :disabled="form_values.current.sshPubkey.length === 1"
              @click="
                () => {
                  form_values.current.sshPubkey.splice(index, 1);
                  if (form_values.validation_errors.sshPubkey.length > index)
                    form_values.validation_errors.sshPubkey.splice(index, 1);
                }
              "
            />
          </template>
        </v-text-field>
      </div>

      <h1 class="text-subtitle-1">Comments</h1>
      <v-textarea
        variant="outlined"
        density="compact"
        v-model="form_values.current.comments"
        persistent-placeholder
        placeholder="Do you have any special wishes or requirements?"
      />

      <v-checkbox
        v-model="form_values.current.accept_terms"
        :rules="[() => form_values.validation_errors.accept_terms || true]"
      >
        <template v-slot:label>
          <p>
            I have read and understood the
            <a href="https://rechtssammlung.sp.ethz.ch/Dokumente/203.21.pdf">
              Benutzungsordnung für Informations- und Kommunikationstechnologie
              an der ETH Zürich (BOT)
            </a>
          </p>
        </template>
      </v-checkbox>

      <div class="d-flex flex-column">
        <v-btn
          class="mt-4"
          :color="submit_color"
          block
          :disabled="submit_disable"
          @click="submit"
        >
          <b>Submit request</b>
        </v-btn>
      </div>
    </v-form>
  </div>
  <!-- </div> -->
</template>

<script>
import {
  mdiEmailOutline,
  mdiInformationOutline,
  mdiPlusBoxOutline,
  mdiMinusBoxOutline,
  mdiLink,
  mdiOfficeBuildingOutline,
  mdiPenguin,
  mdiBackspace,
} from "@mdi/js";
// @ is an alias to /src
export default {
  name: "HomeView",
  watch: {
    "form_values.current": {
      handler() {
        this.storeFormState();
      },
      deep: true,
    },
  },
  data() {
    return {
      mdiEmailOutline,
      mdiLink,
      mdiInformationOutline,
      mdiPlusBoxOutline,
      mdiMinusBoxOutline,
      mdiOfficeBuildingOutline,
      mdiPenguin,
      mdiBackspace,

      submit_color: "primary",
      submit_disable: false,

      form_values: {
        initial: {},
        current: {
          email: "",
          personalEmail: "",
          isOrganization: false,
          orgName: "",

          hostname: "",
          image: "Debian",
          cores: 2,
          ramGB: 2,
          diskGB: 15,

          sshPubkey: [""],

          comments: "",
          accept_terms: false,
        },
        // These values are fetched from the backend on beforeMount
        allowed: {
          image: ["Ubuntu", "Debian"],
          cores: { min: 1, max: 8 },
          ramGB: { min: 2, max: 16 },
          diskGB: { min: 15, max: 500 },
        },
        // These values are received from the backend after submitting the form
        validation_errors: {
          email: "",
          personalEmail: "",
          hostname: "",
          image: "",
          cores: "",
          ramGB: "",
          diskGB: "",
          sshPubkey: "",
          accept_terms: "",
        },
      },
    };
  },
  methods: {
    resetForm() {
      for (const [key, value] of Object.entries(this.form_values.initial)) {
        if (Array.isArray(value)) this.form_values.current[key] = [...value];
        else this.form_values.current[key] = value;
      }
      this.storeFormState();
    },
    isFormModified() {
      return Object.keys(this.form_values.current).some((key) => {
        if (Array.isArray(this.form_values.current[key]))
          return this.form_values.current[key].some(
            (x, i) => x != this.form_values.initial[key][i]
          );
        else
          return this.form_values.current[key] != this.form_values.initial[key];
      });
    },
    // Stores the form state in the current URL
    storeFormState() {
      let newquery = {};

      Object.keys(this.form_values.current).forEach((key) => {
        let currVal = this.form_values.current[key];
        let initVal = this.form_values.initial[key];

        // Remove query params that are the same as the initial values

        // If the current field is an array, check if any of the values are different
        if (Array.isArray(currVal)) {
          currVal = Object.values(this.form_values.current[key]);
          initVal = Object.values(this.form_values.initial[key]);
          // console.log(currVal, initVal);
          if (
            currVal.filter((x) => x != "" && !initVal.includes(x)).length == 0
          ) {
            return;
          } else {
            // Serialize the array to a comma separated string
            newquery[key] = currVal
              .filter((x) => x != "" && !initVal.includes(x))
              .join(",");
          }
        } else {
          if (currVal == initVal) return;
          newquery[key] = currVal;
        }
      });

      this.$router.replace({ query: newquery });
    },

    // Restores the form state from the current URL
    restoreFormState() {
      // Read query params
      let query = this.$route.query;
      for (const [key, value] of Object.entries(query)) {
        if (!Object.keys(this.form_values.current).includes(key)) continue;

        if (value == "true") this.form_values.current[key] = true;
        else if (value == "false") this.form_values.current[key] = false;
        else if (key == "sshPubkey")
          this.form_values.current[key] = value.split(",").some((x) => x != "")
            ? value.split(",").filter((x) => x != "")
            : [""];
        else {
          this.form_values.current[key] = !isNaN(Number(value))
            ? Number(value)
            : value;
        }
      }
    },

    async submit() {
      let response = await this.$store.getters.fetchSendVMRequest(
        this.form_values.current
      );
      if (response.status >= 200 && response.status < 300) {
        this.submit_color = "success";
        this.submit_disable = true;
        setTimeout(() => {
          this.submit_color = "primary";
          this.submit_disable = false;
        }, 2500);
        return;
      }

      this.submit_color = "error";
      this.submit_disable = true;
      setTimeout(() => {
        this.submit_color = "primary";
        this.submit_disable = false;
      }, 2500);

      response.json().then((data) => {
        // console.log(data);
        for (const [key, value] of Object.entries(data)) {
          this.form_values.validation_errors[key] = value;
        }
        this.$refs.form.validate();
      });
    },
  },
  mounted() {
    // Fetches allowed slider ranges and select options from the backend
    this.$store.getters
      .fetchVMOptions()
      .then((response) => response.json())
      .then((data) => {
        console.log(data);
        for (const [key, value] of Object.entries(data)) {
          this.form_values.allowed[key] = value;
        }
      });

    // Saves initial form values aside
    for (const [key, value] of Object.entries(this.form_values.current)) {
      if (Array.isArray(value)) this.form_values.initial[key] = [...value];
      else this.form_values.initial[key] = value;
    }

    this.restoreFormState();
  },
  components: {},
};
</script>
