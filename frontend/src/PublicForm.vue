<template>
  <main class="page"><div class="card"><div class="cover"></div><form class="body" @submit.prevent="submit" novalidate><small>FORM #{{ formId }} · PUBLISHED V{{ versionNumber || '—' }}</small><h1>{{ title }}</h1><p class="intro">Complete the fields below. Your submission will be saved against this exact published form version.</p>
    <div v-if="loading" role="status">Loading form…</div><p v-else-if="loadError" class="error">{{ loadError }}</p>
    <div v-for="field in visibleFields" :key="field.key" class="field"><label :for="`public-${field.key}`">{{ field.label }} <em v-if="field.required">*</em></label><p v-if="field.helpText" class="help">{{ field.helpText }}</p>
      <input v-if="['text','email','number','date'].includes(field.type)" :id="`public-${field.key}`" :type="field.type" :value="values[field.key] ?? ''" @input="setValue(field.key, $event.target.value)" />
      <select v-else-if="field.type === 'select'" :id="`public-${field.key}`" :value="values[field.key] ?? ''" @change="setValue(field.key, $event.target.value)"><option value="">Choose an option</option><option v-for="option in field.validators?.options || []" :key="option" :value="option">{{ option }}</option></select>
      <div v-else-if="field.type === 'radio'" class="choices"><label v-for="option in field.validators?.options || []" :key="option"><input type="radio" :name="`public-${field.key}`" :value="option" :checked="values[field.key] === option" @change="setValue(field.key, option)" /> {{ option }}</label></div>
      <div v-else-if="field.type === 'checkbox'" class="choices"><label v-for="option in field.validators?.options || []" :key="option"><input type="checkbox" :checked="Array.isArray(values[field.key]) && values[field.key].includes(option)" @change="toggleOption(field.key, option, $event.target.checked)" /> {{ option }}</label></div>
      <select v-else-if="field.type === 'multiselect'" multiple class="multi" :id="`public-${field.key}`" :value="values[field.key] || []" @change="setValue(field.key, Array.from($event.target.selectedOptions).map(option => option.value))"><option v-for="option in field.validators?.options || []" :key="option" :value="option">{{ option }}</option></select>
      <span v-if="errors[field.key]" class="error">{{ errors[field.key] }}</span>
    </div>
    <input v-model="honeypot" class="hp" tabindex="-1" aria-hidden="true" autocomplete="off" />
    <button type="submit" :disabled="loading || submitting || !!loadError">{{ submitting ? 'Submitting…' : 'Submit form →' }}</button><p v-if="message" role="status" :class="messageKind">{{ message }}</p><p class="foot">This page renders the published form—not the editor draft.</p>
  </form></div></main>
</template>

<script>
export default {
  name: 'PublicForm',
  props: { formId: { type: String, required: true } },
  data() {
    return { schema: null, versionId: null, versionNumber: null, title: 'Published form', values: {}, errors: {}, honeypot: '', loading: true, loadError: '', submitting: false, message: '', messageKind: 'success' };
  },
  computed: {
    visibleFields() {
      return (this.schema?.fields || []).filter(field => !field.showIf || String(this.values[field.showIf.field] ?? '') === field.showIf.equals);
    }
  },
  async mounted() {
    try {
      const response = await fetch(`/f/${this.formId}`, { cache: 'no-store' });
      if (!response.ok) throw new Error(`Form unavailable (HTTP ${response.status}).`);
      const data = await response.json();
      this.schema = data.schema;
      this.versionId = data.versionId;
      this.versionNumber = data.versionNumber;
    } catch (error) { this.loadError = error.message; }
    finally { this.loading = false; }
  },
  methods: {
    setValue(key, value) { this.values = { ...this.values, [key]: value }; delete this.errors[key]; },
    toggleOption(key, option, checked) {
      const values = new Set(Array.isArray(this.values[key]) ? this.values[key] : []);
      if (checked) values.add(option); else values.delete(option);
      this.setValue(key, [...values]);
    },
    async submit() {
      this.errors = {}; this.message = '';
      const payload = {};
      for (const field of this.visibleFields) {
        const value = this.values[field.key];
        const empty = value === undefined || value === null || value === '' || (Array.isArray(value) && !value.length);
        if (field.required && empty) this.errors[field.key] = 'This field is required.';
        if (!empty) payload[field.key] = field.type === 'number' ? Number(value) : value;
      }
      if (Object.keys(this.errors).length) { this.message = 'Please correct the highlighted fields.'; this.messageKind = 'error'; return; }
      this.submitting = true;
      try {
        const response = await fetch(`/f/${this.formId}/submit`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ versionId: this.versionId, payload, hp: this.honeypot })
        });
        if (response.status === 422) {
          const body = await response.json(); this.errors = body.errors || {};
          this.message = 'Server validation rejected the submission.'; this.messageKind = 'error'; return;
        }
        if (!response.ok) throw new Error(`Submission failed (HTTP ${response.status}).`);
        const body = await response.json();
        this.values = {}; this.errors = {}; this.message = `Submission accepted: ${body.submissionId}`; this.messageKind = 'success';
      } catch (error) { this.message = error.message; this.messageKind = 'error'; }
      finally { this.submitting = false; }
    }
  }
};
</script>

<style scoped>
:global(*){box-sizing:border-box}:global(body){margin:0;font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;background:#f5f5fb;color:#26263d}:global(input),:global(select),:global(button){font:inherit}.page{padding:48px 16px;min-height:100vh}.card{max-width:620px;margin:auto;background:#fff;border-radius:15px;overflow:hidden;box-shadow:0 18px 55px #2525541b}.cover{height:140px;background:radial-gradient(circle at 77% 15%,#b193ff,transparent 30%),linear-gradient(110deg,#392e81,#6656ef 55%,#b272dc)}.body{padding:38px 45px 42px}.body>small{color:#6656ef;font-size:10px;font-weight:800;letter-spacing:.13em}.body h1{font-size:32px;letter-spacing:-.04em;margin:10px 0}.intro{color:#9192a2;font-size:13px;line-height:1.5;margin-bottom:27px}.field{margin:19px 0}.field>label{display:block;font-size:13px;font-weight:700;margin-bottom:8px;color:#505169}.field em{font-style:normal;color:#d66}.help{font-size:11px;color:#9798a7;margin:0 0 8px}.field>input,.field>select{height:43px;border:1px solid #dedfea;border-radius:8px;padding:9px 11px;width:100%;background:#fff;color:#313245}.field>.multi{height:110px}.choices{display:grid;gap:9px}.choices label{display:flex;gap:8px;align-items:center;font-size:12px;color:#616277}.choices input{accent-color:#6656ef}.body button{width:100%;border:0;border-radius:8px;background:#6656ef;color:#fff;font-weight:700;padding:13px;margin-top:20px;cursor:pointer}.body button:disabled{opacity:.55}.hp{position:absolute;left:-9999px}.error{color:#b74457;font-size:11px;display:block;margin-top:6px}.success{color:#24885f;font-size:12px;margin-top:15px}.foot{text-align:center;color:#adaebb;font-size:10px;margin:16px 0 0}@media(max-width:600px){.body{padding:27px}}
</style>