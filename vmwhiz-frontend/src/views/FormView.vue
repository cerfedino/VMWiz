<template>
  <!-- <div class="h-screen d-flex flex-column justify-center"> -->
  <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
    <h1 class="text-h4 text-center font-weight-bold mb-3">VM Request Form</h1>
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
        v-model="form_values.current.personal_email"
        :rules="[() => form_values.validation_errors.personal_email || true]"
      />

      <v-checkbox
        v-model="form_values.current.isOrganization"
        variant="outlined"
        density="compact"
        hide-details="true"
        label="Are you requesting this VM on behalf of an organization?"
      />

      <v-text-field
        v-if="form_values.current.isOrganization"
        :prepend-inner-icon="mdiOfficeBuildingOutline"
        label="Organization Name"
        variant="outlined"
        density="compact"
        hide-details="true"
        v-model="form_values.current.orgName"
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
              <v-list-item-icon>
                <v-icon class="mr-2" :icon="mdiPenguin" />
              </v-list-item-icon>
            </template>
          </v-list-item>
        </template>
        <template v-slot:selection="{ item }">
          <v-list-item-icon>
            <v-icon class="mr-2" :icon="mdiPenguin" />
          </v-list-item-icon>
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
        v-model="form_values.current.ram_gb"
        :max="form_values.allowed.ram_gb.max"
        :min="form_values.allowed.ram_gb.min"
        :rules="[() => form_values.validation_errors.ram_gb || true]"
      >
        <template v-slot:prepend>
          <v-text-field
            style="width: 80px"
            type="number"
            hide-details
            single-line
            variant="outlined"
            density="compact"
            v-model="form_values.current.ram_gb"
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
        v-model="form_values.current.disk_gb"
        :max="form_values.allowed.disk_gb.max"
        :min="form_values.allowed.disk_gb.min"
        :rules="[() => form_values.validation_errors.disk_gb || true]"
      >
        <template v-slot:prepend>
          <v-text-field
            style="width: 80px"
            type="number"
            hide-details
            variant="outlined"
            density="compact"
            v-model="form_values.current.disk_gb"
          />
        </template>
      </v-slider>

      <h1 class="text-subtitle-1 pb-3">
        SSH Public Key(s)
        <v-tooltip text="Tooltip">
          <template v-slot:activator="{ props }">
            <v-icon v-bind="props" :icon="mdiInformationOutline" />
          </template>
          {{ form_values.tooltips.ssh_pubkey }}
        </v-tooltip>
        <v-icon
          :icon="mdiPlusBoxOutline"
          @click="form_values.current.ssh_pubkey.push('')"
        />
        <p class="text-caption text-error">
          {{
            form_values.current.ssh_pubkey.lenght == 0
              ? form_values.validation_errors.ssh_pubkey[0]
              : ""
          }}
        </p>
      </h1>
      <div v-for="(key, index) in form_values.current.ssh_pubkey" :key="index">
        <v-text-field
          v-model="form_values.current.ssh_pubkey[index]"
          variant="outlined"
          density="compact"
          persistent-placeholder
          placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCq..."
          :rules="[
            () => form_values.validation_errors.ssh_pubkey[index] || true,
          ]"
        >
          <template v-slot:prepend>
            <v-icon
              :icon="mdiMinusBoxOutline"
              :disabled="form_values.current.ssh_pubkey.length === 1"
              @click="
                () => {
                  form_values.current.ssh_pubkey.splice(index, 1);
                  form_values.validation_errors.ssh_pubkey.splice(index, 1);
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
} from "@mdi/js";
// @ is an alias to /src
export default {
  name: "HomeView",
  data() {
    return {
      mdiEmailOutline,
      mdiLink,
      mdiInformationOutline,
      mdiPlusBoxOutline,
      mdiMinusBoxOutline,
      mdiOfficeBuildingOutline,
      mdiPenguin,

      submit_color: "primary",
      submit_disable: false,

      form_values: {
        current: {
          email: "",
          personal_email: "",
          isOrganization: false,
          orgName: "",

          hostname: "",
          image: "Debian",
          cores: 2,
          ram_gb: 2,
          disk_gb: 15,

          ssh_pubkey: [""],

          comments: "",
          accept_terms: false,
        },
        // These values are fetched from the backend on beforeMount
        allowed: {
          image: ["Ubuntu", "Debian"],
          cores: { min: 1, max: 8 },
          ram_gb: { min: 2, max: 16 },
          disk_gb: { min: 15, max: 500 },
        },
        tooltips: {
          ssh_pubkey: "Please provide your SSH public key",
        },
        // These values are received from the backend after submitting the form
        validation_errors: {
          email: "",
          personal_email: "",
          hostname: "",
          image: "",
          cores: "",
          ram_gb: "",
          disk_gb: "",
          ssh_pubkey: "",
          accept_terms: "",
        },
      },
    };
  },
  methods: {
    async submit() {
      let response = await fetch("http://localhost:8081/vmrequest", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(this.form_values.current),
      });
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
        console.log(data);
        for (const [key, value] of Object.entries(data)) {
          this.form_values.validation_errors[key] = value;
        }
        this.$refs.form.validate();
      });
    },
  },
  beforeMount() {
    // Fetches allowed slider ranges and select options from the backend
    fetch("http://localhost:8081/formconfig")
      .then((response) => response.json())
      .then((data) => {
        console.log(data);
        for (const [key, value] of Object.entries(data)) {
          this.form_values.allowed[key] = value;
        }
      });
  },
  components: {},
};
</script>
