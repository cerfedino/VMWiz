const { defineConfig } = require("@vue/cli-service");

process.env.VUE_APP_VMWIZ_BASE_URL = `${process.env.VMWIZ_SCHEME}://${process.env.VMWIZ_HOSTNAME}:${process.env.VMWIZ_PORT}`;

console.log("ENV", process.env);
module.exports = defineConfig({
    transpileDependencies: true,
    devServer: {
        host: "0.0.0.0",
        allowedHosts: "all",
        client: {
            webSocketURL: "auto://0.0.0.0:0/ws",
        },
    },
});
