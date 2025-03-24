<template>
  <!-- <div class="h-screen d-flex flex-column justify-center"> -->
  <div class="w-75 pa-6 ma-auto border-t-sm" style="max-width: 700px">
    <h1 class="text-h4 text-center font-weight-bold mb-3">VM Requests</h1>

    <div v-for="(vm, index) in form_values" :key="index">
      <h1 class="text-h6 font-weight-bold mb-3">General Information</h1>

      <h1 class="text-subtitle-1">Request ID</h1>
      {{ vm.id }}
      <h1 class="text-subtitle-1">Created</h1>
      {{ vm.created }}
      <h1 class="text-subtitle-1">Email</h1>
      {{ vm.email }}
      <h1 class="text-subtitle-1">Non ETH email</h1>
      {{ vm.personalEmail }}
      <div v-if="vm.isOrganization">
        <h1 class="text-subtitle-1">Organization</h1>
        {{ vm.orgName }}
      </div>

      <h1 class="text-h6 font-weight-bold mt-4">VM specification</h1>
      <h1 class="text-subtitle-1 mt-3">Hostname</h1>
      {{ vm.hostname }}
      <h1 class="text-subtitle-1">OS Image</h1>
      {{ vm.image }}

      <h1 class="text-subtitle-1">CPU Cores</h1>
      {{ vm.cpuCores }}
      <h1 class="text-subtitle-1">RAM (GB)</h1>
      {{ vm.ramGB }}

      <h1 class="text-subtitle-1">Disk Space (GB)</h1>
      {{ vm.diskGB }}

      <h1 class="text-subtitle-1 pb-3">SSH Public Key(s)</h1>
      <div v-for="(key, index) in vm.sshPubkey" :key="index">
        {{ key }}
      </div>

      <h1 class="text-subtitle-1">Comments</h1>
      <v-textarea
        variant="outlined"
        density="compact"
        v-model="vm.comments"
        persistent-placeholder
        placeholder=""
      />

      <div class="d-flex flex-column">
        <v-btn class="mt-4" :color="submit_color" @click="submit">
          <b>Accept request</b>
        </v-btn>
      </div>
    </div>
  </div>
  <!-- </div> -->
</template>

<script>
const vmreq = {
  id: 0,
  created: "",
  email: "",
  personalEmail: "",
  isOrganization: false,
  orgName: "",
  hostname: "",
  image: "",
  cores: 0,
  ramGB: 0,
  diskGB: 0,
  sshPubkeys: [],
  comments: "",
};
export default {
  name: "AdminView",
  data() {
    return {
      form_values: [vmreq],
    };
  },
  methods: {},
  mounted() {
    // Fetches allowed slider ranges and select options from the backend
    this.$store.getters
      .fetchRequests()
      .then((response) => response.json())
      .then((data) => {
        console.log(data);
        for (const [key, value] of Object.entries(data)) {
          this.form_values[key] = value;
        }
      });
  },
  components: {},
};
</script>
