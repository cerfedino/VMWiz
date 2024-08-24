<template>
  <!-- <nav>
    <router-link to="/">Home</router-link> |
    <router-link to="/about">About</router-link>
  </nav> -->
  <div
    class="w-screen py-2 d-flex flex-column align-center"
    style="height: 14vh"
  >
    <img style="height: 75%; max-width: 75%" src="@/assets/SOSETH_Logo.svg" />
    <h1 class="text-h5 h-auto">VMWiz</h1>
  </div>
  <div style="min-height: 86vh">
    <router-view />
  </div>

  <footer class="bg-white pa-6 d-flex flex-column align-center">
    Made with ❤️ by VSOS
    <div class="w-25 d-flex mt-4 justify-space-evenly">
      <v-btn :icon="mdiGithub" density="compact" variant="plain" />
      <v-btn :icon="mdiWeb" density="compact" variant="plain" />
      <v-btn :icon="mdiAccountKey" density="compact" variant="plain" />
    </div>
  </footer>
</template>

<script setup>
import { mdiGithub, mdiAccountKey, mdiWeb } from "@mdi/js";
</script>

<style scoped>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

nav {
  padding: 30px;
}

nav a {
  font-weight: bold;
  color: #2c3e50;
}

nav a.router-link-exact-active {
  color: #42b983;
}
</style>
