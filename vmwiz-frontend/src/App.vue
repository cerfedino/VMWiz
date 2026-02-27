<template>
    <div
        class="w-screen py-2 d-flex flex-column align-center"
        style="min-height: 14vh"
    >
        <img
            style="height: 8vh; max-width: 20vw"
            src="@/assets/SOSETH_Logo.svg"
        />
        <h1>VMWiz</h1>
    </div>
    <v-divider />
    <div style="min-height: 86vh">
        <router-view />
    </div>

    <footer class="bg-white pa-6 d-flex flex-column align-center">
        Made with ❤️ by VSOS
        <div class="w-25 d-flex mt-4 ga-8 justify-center">
            <a href="https://vsos.ethz.ch">
                <v-btn :icon="mdiWeb" density="compact" variant="plain" />
            </a>
            <a href="/console">
                <v-btn
                    :icon="mdiAccountKey"
                    density="compact"
                    variant="plain"
                />
            </a>
        </div>
    </footer>
</template>

<script setup>
import { mdiAccountKey, mdiWeb } from "@mdi/js";
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
