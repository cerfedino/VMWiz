const { defineConfig } = require("@vue/cli-service");

// Makes build environment variables available to the client
process.env.VUE_APP_VMWIZ_SCHEME = process.env.VMWIZ_SCHEME;
process.env.VUE_APP_VMWIZ_HOSTNAME = process.env.VMWIZ_HOSTNAME;
process.env.VUE_APP_VMWIZ_PORT = process.env.VMWIZ_PORT;


console.log("ENV",  process.env)
module.exports = defineConfig({
  transpileDependencies: true,
  devServer: {
    // Enables/disables HTTPS based on the VMWIZ_SCHEME environment variable
    server: process.env.VUE_APP_VMWIZ_SCHEME,
  }
});
