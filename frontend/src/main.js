import { createApp } from "vue";
import FormBuilder from "./FormBuilder.vue";

const params = new URLSearchParams(window.location.search);
const formId = params.get("formId") || "3";
createApp(FormBuilder, { formId }).mount("#app");
