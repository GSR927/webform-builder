import { createApp } from 'vue';
import PublicForm from './PublicForm.vue';

const formId = new URLSearchParams(window.location.search).get('formId') || '3';
createApp(PublicForm, { formId }).mount('#public-app');
