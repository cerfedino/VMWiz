import pluginVue from "eslint-plugin-vue";
import eslintConfigPrettier from "eslint-config-prettier";
import eslintPluginPrettier from "eslint-plugin-prettier";
import js from "@eslint/js";

export default [
    js.configs.recommended,
    ...pluginVue.configs["flat/essential"],
    eslintConfigPrettier,
    {
        plugins: {
            prettier: eslintPluginPrettier,
        },
        rules: {
            "prettier/prettier": "error",
            "arrow-body-style": "off",
            "prefer-arrow-callback": "off",
            "no-console":
                process.env.NODE_ENV === "production" ? "warn" : "off",
            "no-debugger":
                process.env.NODE_ENV === "production" ? "warn" : "off",
            "max-len": "off",
            "vue/max-attributes-per-line": "off",
        },
    },
];
