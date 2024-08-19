const { defineConfig } = require("@vue/cli-service");

// Makes build environment variables available to the client
process.env.VUE_APP_VMWHIZ_SCHEME = process.env.VMWHIZ_SCHEME;
process.env.VUE_APP_VMWHIZ_HOSTNAME = process.env.VMWHIZ_HOSTNAME;
process.env.VUE_APP_VMWHIZ_PORT = process.env.VMWHIZ_PORT;


console.log("ENV",  process.env)
module.exports = defineConfig({
  transpileDependencies: true,
  devServer: {
    // Enables/disables HTTPS based on the VMWHIZ_SCHEME environment variable
    server: process.env.VUE_APP_VMWHIZ_SCHEME,
  }
});
