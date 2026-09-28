<!--
  FormRenderer.vue -- minimal, working slice of the public form renderer.

  Responsibilities proven here:
    1. Fetch the published schema for a given formId (never trusts any
       locally-cached copy of the schema across sessions -- always fetches
       fresh, relying on HTTP caching headers set server-side for perf).
    2. Render dynamic fields (text, email, select) from that schema.
    3. Evaluate conditional visibility (showIf) client-side for UX.
    4. Run lightweight client-side validation for UX ONLY -- the real
       authority is the server, which re-validates on submit regardless of
       what this component allows through.
    5. Submit and surface server-side validation errors per field.

  NOT implemented here (documented in ARCHITECTURE.md Built vs Designed):
    number/date/multiselect/checkbox rendering, drag-drop builder UI,
    CAPTCHA widget, i18n.
-->
<template>
  <form v-if="schema" @submit.prevent="onSubmit" novalidate>
    <div v-for="field in visibleFields" :key="field.key" class="field">
      <label :for="field.key">{{ field.label }}<span v-if="field.required"> *</span></label>
      <p v-if="field.helpText" class="help">{{ field.helpText }}</p>

      <input
        v-if="field.type === 'text' || field.type === 'email'"
        :id="field.key"
        :type="field.type"
        v-model="values[field.key]"
        :required="field.required"
      />

      <select v-else-if="field.type === 'select'" :id="field.key" v-model="values[field.key]" :required="field.required">
        <option disabled value="">Select...</option>
        <option v-for="opt in field.validators?.options || []" :key="opt" :value="opt">{{ opt }}</option>
      </select>

      <p v-if="errors[field.key]" class="error">{{ errors[field.key] }}</p>
    </div>

    <!-- Honeypot: visually hidden, never filled by real users -->
    <input type="text" v-model="honeypot" name="hp" autocomplete="off" style="position:absolute;left:-9999px" tabindex="-1" />

    <button type="submit" :disabled="submitting">Submit</button>
    <p v-if="submitStatus === 'success'">Thanks — your submission was received.</p>
    <p v-if="submitStatus === 'error'" class="error">Something went wrong. Please try again.</p>
  </form>
</template>

<script>
export default {
  props: { formId: { type: String, required: true } },
  data() {
    return {
      schema: null,
      versionId: null,
      values: {},
      errors: {},
      honeypot: "",
      submitting: false,
      submitStatus: null,
    };
  },
  computed: {
    visibleFields() {
      if (!this.schema) return [];
      return this.schema.fields.filter((f) => {
        if (!f.showIf) return true;
        return this.values[f.showIf.field] === f.showIf.equals;
      });
    },
  },
  async mounted() {
    const res = await fetch(`/f/${this.formId}`);
    const data = await res.json();
    this.versionId = data.versionId;
    this.schema = data.schema;
  },
  methods: {
    async onSubmit() {
      this.errors = {};
      this.submitting = true;
      this.submitStatus = null;
      try {
        const res = await fetch(`/f/${this.formId}/submit`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            versionId: this.versionId,
            payload: this.values,
            hp: this.honeypot,
          }),
        });
        if (res.status === 202) {
          this.submitStatus = "success";
          this.values = {};
        } else if (res.status === 422) {
          const body = await res.json();
          this.errors = body.errors || {};
        } else {
          this.submitStatus = "error";
        }
      } catch (e) {
        this.submitStatus = "error";
      } finally {
        this.submitting = false;
      }
    },
  },
};
</script>

<style scoped>
.field { margin-bottom: 1rem; }
.error { color: #c0392b; font-size: 0.85rem; }
.help { color: #666; font-size: 0.8rem; }
</style>
