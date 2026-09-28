import { createApp } from "vue";
import SubmissionAccess from "./SubmissionAccess.vue";
createApp(SubmissionAccess, { formId: new URLSearchParams(location.search).get("formId") || "3" }).mount("#app");
