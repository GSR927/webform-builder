<template>
  <div class="studio">
    <header class="topbar">
      <div class="brand"><span class="brand-icon">✦</span><span>Formly</span><small>WEBFORM STUDIO</small></div>
      <div class="status"><span class="status-dot"></span> Form #{{ formId }} <span class="separator">·</span> {{ statusText }}</div>
      <div class="actions">
        <button class="quiet" @click="enterPreview">{{ mode === 'build' ? 'Preview' : 'Back to editor' }}</button>
        <button class="primary" :disabled="busy" @click="publish">{{ busy ? 'Working…' : 'Publish form ↗' }}</button>
      </div>
    <a class="customer-link" :href="`/submissions.html?formId=${formId}`">View submissions ↗</a></header>

    <div v-if="mode === 'build'" class="workspace">
      <aside class="palette">
        <h2>Build your form</h2><p class="muted">Add fields and configure them on the right</p>
        <input v-model="search" class="search" placeholder="⌕  Search fields" aria-label="Search fields" />
        <div v-for="group in filteredGroups" :key="group.title" class="palette-group">
          <h3>{{ group.title }}</h3>
          <button v-for="item in group.items" :key="item.type" class="palette-item" @click="addField(item.type)">
            <span class="type-icon">{{ item.icon }}</span>{{ item.label }}<span class="plus">+</span>
          </button>
        </div>
        <div class="tip">✦ <span><strong>Tip</strong><br />Select a field on the canvas to change its rules and visibility.</span></div>
      </aside>

      <main class="canvas-area">
        <div class="canvas-heading"><div><small>FORM CANVAS</small><h1>{{ formTitle || 'Untitled form' }}</h1></div><span class="pill">{{ dirty ? '● Unsaved changes' : '✓ Saved draft' }}</span></div>
        <div class="form-card">
          <div class="cover"></div>
          <div class="form-inner">
            <small class="overline">YOUR FORM</small><h2>{{ formTitle }}</h2><p class="description">{{ formDescription }}</p>
            <p v-if="!fields.length" class="empty">Choose a field from the left to get started.</p>
            <div v-for="(field, index) in fields" :key="field.localId" class="canvas-field" :class="{ selected: selectedId === field.localId }" @click="selectedId = field.localId">
              <span class="ordinal">{{ String(index + 1).padStart(2, '0') }}</span>
              <div class="field-body">
                <label>{{ field.label || field.key }} <em v-if="field.required">*</em></label><small v-if="field.helpText">{{ field.helpText }}</small>
                <div v-if="['text', 'email', 'number', 'date'].includes(field.type)" class="mock-input">{{ field.placeholder || inputHint(field) }}</div>
                <div v-else-if="field.type === 'select'" class="mock-input">▾ &nbsp; Choose an option</div>
                <div v-else-if="field.type === 'multiselect'" class="mock-multiple"><span v-for="option in field.options" :key="option">{{ option }}</span><small>Use Ctrl/Cmd to choose more than one option</small></div>
                <div v-else class="choice-preview"><label v-for="option in field.options" :key="option"><span>{{ field.type === 'radio' ? '◯' : '□' }}</span>{{ option }}</label></div>
                <div v-if="field.conditional" class="condition-tag">Shown when {{ field.showIf.field || 'another field' }} equals {{ field.showIf.equals || '…' }}</div>
              </div>
              <button class="delete" :aria-label="`Remove ${field.label}`" @click.stop="removeField(field.localId)">×</button>
            </div>
            <button type="button" class="submit-look" @click="enterPreview">Preview form <span>→</span></button>
            <p class="footnote">Your visitors will see the published version—not your unsaved edits.</p>
          </div>
        </div>
      </main>

      <aside class="inspector">
        <div class="tabs"><button :class="{ active: tab === 'field' }" @click="tab = 'field'">Field settings</button><button :class="{ active: tab === 'style' }" @click="tab = 'style'">Form content</button></div>
        <div v-if="tab === 'style'" class="inspector-body"><h3>FORM CONTENT</h3><label>Title<input v-model="formTitle" @input="dirty = true" maxlength="120" /></label><label>Description<textarea v-model="formDescription" @input="dirty = true" rows="4" maxlength="500"></textarea></label><p class="hint">Title and description are editor-only in this working slice; the persisted schema stores fields and validation rules.</p></div>
        <div v-else-if="selected" class="inspector-body">
          <div class="selected-heading"><span class="type-icon">{{ iconFor(selected.type) }}</span><div><strong>{{ selected.label }}</strong><small>{{ selected.type }}</small></div></div>
          <h3>CONTENT</h3><label>Field key<input v-model.trim="selected.key" @input="dirty = true" placeholder="unique_field_key" /></label><label>Label<input v-model="selected.label" @input="dirty = true" maxlength="120" /></label><label>Placeholder<input v-model="selected.placeholder" @input="dirty = true" maxlength="120" /></label><label>Help text<input v-model="selected.helpText" @input="dirty = true" maxlength="300" /></label><label class="check"><input type="checkbox" v-model="selected.required" @change="dirty = true" /> Required field</label>
          <template v-if="isChoice(selected)"><h3>OPTIONS</h3><div v-for="(option, i) in selected.options" :key="i" class="option-row"><input v-model="selected.options[i]" @input="dirty = true" :aria-label="`Option ${i + 1}`" /><button @click="selected.options.splice(i, 1); dirty = true">×</button></div><button class="text-button" @click="selected.options.push('New option'); dirty = true">+ Add option</button></template>
          <template v-if="selected.type === 'text' || selected.type === 'email'"><h3>VALIDATION</h3><label>Minimum length<input type="number" min="0" v-model.number="selected.minLength" @input="dirty = true" /></label><label>Maximum length<input type="number" min="1" v-model.number="selected.maxLength" @input="dirty = true" /></label></template>
          <template v-if="selected.type === 'number'"><h3>VALIDATION</h3><label>Minimum<input type="number" v-model.number="selected.min" @input="dirty = true" /></label><label>Maximum<input type="number" v-model.number="selected.max" @input="dirty = true" /></label></template>
          <h3>CONDITIONAL VISIBILITY</h3><label class="check"><input type="checkbox" v-model="selected.conditional" @change="dirty = true" /> Show conditionally</label><template v-if="selected.conditional"><label>When field<select v-model="selected.showIf.field" @change="dirty = true"><option value="">Choose a field</option><option v-for="f in fields.filter(x => x.localId !== selected.localId)" :key="f.localId" :value="f.key">{{ f.label }}</option></select></label><label>Equals<input v-model="selected.showIf.equals" @input="dirty = true" placeholder="Exact option or value" /></label></template>
        </div>
        <div v-else class="empty-settings">Select a field in the canvas to edit its settings.</div>
        <div class="save-area"><button :disabled="busy" class="outline" @click="saveDraft">{{ busy ? 'Working…' : 'Save draft' }}</button><small>Save draft does not change the published version.</small></div>
      </aside>
    </div>

    <main v-else class="preview-page">
      <div class="preview-heading"><span>INTERACTIVE PREVIEW · {{ previewIsDraft ? 'UNPUBLISHED DRAFT' : `PUBLISHED VERSION ${versionNumber}` }}</span><button class="quiet" @click="mode = 'build'">← Back to editor</button></div>
      <div v-if="versionId" class="public-link"><a :href="publicUrl" target="_blank" rel="noopener">Open published form ↗</a><span>Version {{ versionNumber }} · This link always serves the current published schema.</span></div>
      <div class="form-card preview-card"><div class="cover"></div><form class="form-inner" @submit.prevent="submitPreview" novalidate><small class="overline">YOUR FORM</small><h1>{{ formTitle }}</h1><p class="description">{{ formDescription }}</p>
        <div v-for="field in visibleFields" :key="field.localId" class="live-field"><label :for="`preview-${field.localId}`">{{ field.label }} <em v-if="field.required">*</em></label><small v-if="field.helpText">{{ field.helpText }}</small>
          <input v-if="['text','email','number','date'].includes(field.type)" :id="`preview-${field.localId}`" :type="field.type" :placeholder="field.placeholder" :value="previewValues[field.key] ?? ''" @input="setPreviewValue(field, $event.target.value)" />
          <select v-else-if="field.type === 'select'" :id="`preview-${field.localId}`" :value="previewValues[field.key] ?? ''" @change="setPreviewValue(field, $event.target.value)"><option value="">Choose an option</option><option v-for="option in field.options" :key="option" :value="option">{{ option }}</option></select>
          <div v-else-if="field.type === 'radio'" class="check-options"><label v-for="option in field.options" :key="option"><input type="radio" :name="`radio-${field.localId}`" :value="option" :checked="previewValues[field.key] === option" @change="setPreviewValue(field, option)" /> {{ option }}</label></div>
          <div v-else-if="field.type === 'checkbox'" class="check-options"><label v-for="option in field.options" :key="option"><input type="checkbox" :checked="Array.isArray(previewValues[field.key]) && previewValues[field.key].includes(option)" @change="togglePreviewOption(field, option, $event.target.checked)" /> {{ option }}</label></div>
          <select v-else-if="field.type === 'multiselect'" multiple class="multi-select" :id="`preview-${field.localId}`" :value="previewValues[field.key] || []" @change="setPreviewValue(field, Array.from($event.target.selectedOptions).map(option => option.value))"><option v-for="option in field.options" :key="option" :value="option">{{ option }}</option></select>
          <span v-if="errors[field.key]" class="error">{{ errors[field.key] }}</span>
        </div>
        <input v-model="honeypot" class="honeypot" tabindex="-1" autocomplete="off" aria-hidden="true" />
        <button class="submit-look" type="submit" :disabled="submitting || previewIsDraft || !versionId">{{ submitting ? 'Submitting…' : 'Submit form →' }}</button>
        <p v-if="previewIsDraft" class="draft-warning">Preview only: publish this draft before collecting submissions.</p>
        <p v-if="message" role="status" :class="messageKind">{{ message }}</p>
      </form></div>
    </main>
    <div v-if="message && mode === 'build'" role="status" class="toast" :class="messageKind">{{ message }}</div>
  </div>
</template>

<script>
function csrfHeader() {
  const entry = document.cookie.split('; ').find(value => value.startsWith('webform_csrf='));
  return entry ? decodeURIComponent(entry.substring('webform_csrf='.length)) : '';
}
const CATALOG = [
  { title: 'BASIC FIELDS', items: [{ type: 'text', label: 'Short text', icon: 'T' }, { type: 'email', label: 'Email address', icon: '@' }, { type: 'number', label: 'Number', icon: '#' }, { type: 'date', label: 'Date', icon: '◷' }] },
  { title: 'CHOICE FIELDS', items: [{ type: 'select', label: 'Dropdown', icon: '☷' }, { type: 'radio', label: 'Single choice', icon: '◉' }, { type: 'checkbox', label: 'Checkboxes', icon: '☑' }, { type: 'multiselect', label: 'Multi-select', icon: '⊞' }] }
];

export default {
  name: 'FormBuilder',
  props: { formId: { type: String, required: true } },
  data() {
    return {
      mode: 'build', tab: 'field', search: '', fields: [], selectedId: null,
      formTitle: 'Newsletter signup', formDescription: 'Tell us a little about yourself.',
      versionId: null, versionNumber: null, publishedSchema: null, dirty: false,
      busy: false, submitting: false, previewValues: {}, errors: {}, honeypot: '',
      message: '', messageKind: 'success', noticeTimer: null
    };
  },
  computed: {
    filteredGroups() {
      return CATALOG.map(group => ({ ...group, items: group.items.filter(item => item.label.toLowerCase().includes(this.search.toLowerCase())) })).filter(group => group.items.length);
    },
    selected() { return this.fields.find(field => field.localId === this.selectedId); },
    statusText() { return this.dirty ? 'Unsaved changes' : this.versionId ? `Published version ${this.versionNumber} · draft saved` : 'Draft'; },
    publicUrl() { return `/public.html?formId=${encodeURIComponent(this.formId)}`; },
    previewIsDraft() {
      if (!this.versionId || this.dirty || !this.publishedSchema) return true;
      return JSON.stringify(this.toSchema()) !== JSON.stringify(this.publishedSchema);
    },
    visibleFields() {
      return this.fields.filter(field => !field.conditional || String(this.previewValues[field.showIf.field] ?? '') === field.showIf.equals);
    }
  },
  async mounted() { await this.loadForm(); },
  beforeUnmount() { window.clearTimeout(this.noticeTimer); },
  methods: {
    iconFor(type) { return { email: '@', number: '#', date: '◷', select: '☷', radio: '◉', checkbox: '☑', multiselect: '⊞' }[type] || 'T'; },
    isChoice(field) { return ['select', 'radio', 'checkbox', 'multiselect'].includes(field.type); },
    inputHint(field) { return this.isChoice(field) ? (field.options[0] || 'Choose an option') : `Enter ${field.label.toLowerCase()}…`; },
    normalize(field, index) {
      return {
        localId: `${Date.now()}-${index}`, key: field.key, type: field.type,
        label: field.label || field.key, required: !!field.required,
        helpText: field.helpText || field.help_text || '', placeholder: field.placeholder || '',
        options: [...(field.validators?.options || ['Option one', 'Option two'])],
        minLength: field.validators?.minLength ?? '', maxLength: field.validators?.maxLength ?? '',
        min: field.validators?.min ?? '', max: field.validators?.max ?? '',
        conditional: !!field.showIf,
        showIf: field.showIf ? { ...field.showIf } : { field: '', equals: '' }
      };
    },
    async loadForm() {
      try {
        const [draftResponse, publicResponse] = await Promise.all([
          fetch(`/api/forms/${this.formId}/draft`, { cache: 'no-store' }),
          fetch(`/f/${this.formId}`, { cache: 'no-store' })
        ]);
        if (!draftResponse.ok) throw new Error(`Could not load saved draft (HTTP ${draftResponse.status}).`);
        const draft = await draftResponse.json();
        this.fields = (draft.schema?.fields || []).map((field, index) => this.normalize(field, index));
        this.selectedId = this.fields[0]?.localId || null;
        if (draft.name) this.formTitle = draft.name;
        if (publicResponse.ok) {
          const published = await publicResponse.json();
          this.versionId = published.versionId;
          this.versionNumber = published.versionNumber;
          this.publishedSchema = published.schema;
        }
        this.dirty = false;
      } catch (error) {
        this.notify(error.message, 'error-message');
      }
    },
    enterPreview() {
      if (this.mode === 'preview') { this.mode = 'build'; return; }
      this.previewValues = {}; this.errors = {}; this.message = '';
      this.mode = 'preview';
    },
    addField(type) {
      const label = { text: 'Your name', email: 'Email address', number: 'Your number', date: 'Choose a date', select: 'Choose one', radio: 'Single choice', checkbox: 'Select all that apply', multiselect: 'Choose options' }[type];
      const field = this.normalize({ key: `field_${Date.now()}`, type, label }, this.fields.length);
      this.fields.push(field); this.selectedId = field.localId; this.dirty = true;
    },
    removeField(id) {
      this.fields = this.fields.filter(field => field.localId !== id);
      if (this.selectedId === id) this.selectedId = this.fields[0]?.localId || null;
      this.dirty = true;
    },
    toSchema() {
      return { fields: this.fields.map(field => {
        const validators = {};
        if (this.isChoice(field)) validators.options = field.options.map(option => option.trim()).filter(Boolean);
        if (field.minLength !== '' && ['text', 'email'].includes(field.type)) validators.minLength = Number(field.minLength);
        if (field.maxLength !== '' && ['text', 'email'].includes(field.type)) validators.maxLength = Number(field.maxLength);
        if (field.min !== '' && field.type === 'number') validators.min = Number(field.min);
        if (field.max !== '' && field.type === 'number') validators.max = Number(field.max);
        return {
          key: field.key.trim(), type: field.type, label: field.label.trim(),
          helpText: field.helpText, required: field.required, validators,
          ...(field.conditional ? { showIf: { field: field.showIf.field, equals: field.showIf.equals } } : {})
        };
      }) };
    },
    validateSchema() {
      if (!this.fields.length) return 'Add at least one field.';
      const keys = new Set();
      for (const field of this.fields) {
        if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(field.key || '')) return `Invalid field key: ${field.key || '(empty)'}`;
        if (keys.has(field.key)) return `Duplicate field key: ${field.key}`;
        keys.add(field.key);
        if (!field.label.trim()) return 'Every field needs a label.';
        if (this.isChoice(field) && !field.options.some(value => value.trim())) return `${field.label} needs an option.`;
        if (field.conditional && (!field.showIf.field || !this.fields.some(other => other.key === field.showIf.field && other.localId !== field.localId))) return `Choose a valid controlling field for ${field.label}.`;
      }
      return null;
    },
    notify(text, kind = 'success') {
      this.message = text; this.messageKind = kind;
      window.clearTimeout(this.noticeTimer);
      this.noticeTimer = window.setTimeout(() => { this.message = ''; }, 7000);
    },
    async saveDraft() {
      const problem = this.validateSchema();
      if (problem) { this.notify(problem, 'error-message'); return false; }
      this.busy = true;
      try {
        const response = await fetch(`/api/forms/${this.formId}/draft`, {
          method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfHeader() }, body: JSON.stringify(this.toSchema())
        });
        if (!response.ok) throw new Error(`Save failed (HTTP ${response.status}).`);
        this.dirty = false;
        this.notify('Draft saved. Published form is unchanged.');
        return true;
      } catch (error) { this.notify(error.message, 'error-message'); return false; }
      finally { this.busy = false; }
    },
    async publish() {
      const saved = await this.saveDraft();
      if (!saved) return;
      this.busy = true;
      try {
        const response = await fetch(`/api/forms/${this.formId}/publish`, { method: 'POST', headers: { 'X-CSRF-Token': csrfHeader() } });
        if (!response.ok) throw new Error(`Publish failed (HTTP ${response.status}).`);
        const data = await response.json();
        this.versionId = data.versionId;
        this.versionNumber = data.versionNumber;
        this.publishedSchema = this.toSchema();
        this.previewValues = {}; this.errors = {};
        this.notify(`Published version ${data.versionNumber}. Open the public form from Preview.`);
      } catch (error) { this.notify(error.message, 'error-message'); }
      finally { this.busy = false; }
    },
    setPreviewValue(field, value) {
      this.previewValues = { ...this.previewValues, [field.key]: value };
      delete this.errors[field.key];
    },
    togglePreviewOption(field, option, checked) {
      const values = new Set(Array.isArray(this.previewValues[field.key]) ? this.previewValues[field.key] : []);
      if (checked) values.add(option); else values.delete(option);
      this.setPreviewValue(field, [...values]);
    },
    async submitPreview() {
      if (this.previewIsDraft || !this.versionId) { this.notify('Publish this draft before collecting submissions.', 'error-message'); return; }
      this.errors = {};
      const payload = {};
      for (const field of this.visibleFields) {
        const value = this.previewValues[field.key];
        const empty = value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0);
        if (field.required && empty) this.errors[field.key] = 'This field is required.';
        if (!empty) payload[field.key] = field.type === 'number' ? Number(value) : value;
      }
      if (Object.keys(this.errors).length) { this.notify('Please correct the highlighted fields.', 'error-message'); return; }
      this.submitting = true;
      try {
        const response = await fetch(`/f/${this.formId}/submit`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ versionId: this.versionId, payload, hp: this.honeypot })
        });
        if (response.status === 422) {
          const data = await response.json(); this.errors = data.errors || {};
          this.notify('Server validation rejected this submission.', 'error-message'); return;
        }
        if (!response.ok) throw new Error(`Submission failed (HTTP ${response.status}).`);
        const data = await response.json();
        this.previewValues = {}; this.errors = {};
        this.notify(`Submission accepted: ${data.submissionId}`);
      } catch (error) { this.notify(error.message, 'error-message'); }
      finally { this.submitting = false; }
    }
  }
};
</script>

<style scoped>.customer-link{color:#6656ef;text-decoration:none;font-size:12px;font-weight:700;margin:0 10px}
:global(*){box-sizing:border-box}:global(body){margin:0;font-family:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;background:#f7f8fc;color:#25253b}:global(button),:global(input),:global(select),:global(textarea){font:inherit}.studio{min-height:100vh}.topbar{height:74px;background:#fff;border-bottom:1px solid #e9e9f0;display:flex;align-items:center;justify-content:space-between;padding:0 30px}.brand{display:flex;align-items:center;gap:10px;font-size:18px;font-weight:800}.brand-icon{background:#6556ef;color:#fff;border-radius:8px;padding:4px 7px}.brand small{color:#a4a4b3;font-size:10px;letter-spacing:.11em;margin-left:3px}.status{font-size:12px;color:#8e8fa0}.status-dot{display:inline-block;background:#4cc693;width:7px;height:7px;border-radius:50%;margin-right:7px}.separator{margin:0 9px}.actions{display:flex;gap:10px}button{cursor:pointer}.quiet,.primary,.outline{border-radius:8px;border:1px solid #e3e3ed;background:#fff;color:#58596f;padding:10px 14px;font-size:12px;font-weight:700}.primary{background:#6656ef;color:#fff;border-color:#6656ef;box-shadow:0 4px 14px #6656ef33}.primary:disabled,.outline:disabled,.submit-look:disabled{opacity:.55;cursor:not-allowed}.workspace{display:grid;grid-template-columns:250px minmax(460px,1fr) 282px;min-height:calc(100vh - 74px)}.palette,.inspector{background:#fff;padding:24px 18px}.palette{border-right:1px solid #e9e9f0}.inspector{border-left:1px solid #e9e9f0;padding:0;display:flex;flex-direction:column}.palette h2{font-size:15px;margin:0}.muted{font-size:11px;color:#999aaa;margin:6px 0 18px}.search,.inspector input:not([type=checkbox]),.inspector textarea,.inspector select{width:100%;border:1px solid #e3e4ed;border-radius:7px;padding:10px;font-size:12px;outline:none;background:#fff;color:#44465b}.palette-group{margin-top:25px}.palette-group h3,.inspector-body h3{color:#a2a3b1;font-size:10px;letter-spacing:.13em;margin:18px 0 9px}.palette-item{width:100%;padding:8px 4px;display:flex;gap:12px;align-items:center;background:transparent;border:0;color:#5b5c70;text-align:left;font-size:12px;border-radius:7px}.palette-item:hover{background:#f5f3ff;color:#6656ef}.type-icon{background:#edebff;color:#6656ef;border-radius:7px;min-width:29px;height:29px;display:grid;place-items:center;font-weight:800}.plus{margin-left:auto;color:#a8a9b5}.tip{margin-top:35px;background:#f7f5ff;border:1px solid #eeebff;border-radius:9px;padding:14px;color:#6656ef;display:flex;gap:10px;font-size:11px;line-height:1.5}.tip span{color:#77778e}.canvas-area{padding:28px 35px;background:linear-gradient(130deg,#fafaff,#f5f6fb);overflow:auto}.canvas-heading{max-width:650px;display:flex;align-items:center;justify-content:space-between;margin:0 auto 20px}.canvas-heading small{font-size:10px;color:#999aa8;font-weight:800;letter-spacing:.13em}.canvas-heading h1{font-size:20px;margin:6px 0 0}.pill{font-size:11px;color:#7165d7}.form-card{max-width:630px;margin:auto;border:1px solid #ebebf2;border-radius:14px;overflow:hidden;background:#fff;box-shadow:0 15px 38px #20224513}.cover{height:122px;background:radial-gradient(circle at 77% 16%,#ad8dff,transparent 27%),linear-gradient(115deg,#3c347e,#6656ef 56%,#b072da)}.form-inner{padding:32px 42px 35px}.overline{color:#6656ef;font-weight:800;letter-spacing:.13em;font-size:10px}.form-inner h1,.form-inner h2{font-size:30px;letter-spacing:-.045em;margin:10px 0 6px}.description{font-size:13px;color:#9293a1;margin:0 0 25px;line-height:1.5}.empty{color:#999aaa;text-align:center;padding:25px;border:1px dashed #d5d0ff;border-radius:9px}.canvas-field{display:flex;gap:12px;padding:14px 10px;margin:0 -10px 10px;border:1px solid transparent;border-radius:9px;transition:.2s;cursor:pointer}.canvas-field:hover,.canvas-field.selected{background:#faf9ff;border-color:#b8b1ff}.ordinal{font-size:10px;color:#afb0bc}.field-body{flex:1;min-width:0}.field-body>label,.live-field>label{display:block;color:#515167;font-weight:700;font-size:12px;margin-bottom:8px}.field-body>small,.live-field>small{display:block;font-size:11px;color:#9899a8;margin-bottom:7px}em{font-style:normal;color:#e87581}.mock-input,.live-field input:not([type=checkbox]):not([type=radio]),.live-field select{height:40px;border:1px solid #e4e4ee;border-radius:7px;color:#a0a1ae;padding:10px 12px;background:#fff;font-size:12px;width:100%}.condition-tag{margin-top:7px;font-size:10px;color:#6556ef}.choice-preview{display:grid;gap:8px;color:#696a7e;font-size:12px}.choice-preview label{display:flex;gap:8px;align-items:center;font-weight:400}.choice-preview span{font-size:17px;color:#7668e9}.mock-multiple{border:1px solid #e4e4ee;border-radius:7px;padding:10px;display:flex;flex-wrap:wrap;gap:6px}.mock-multiple span{border-radius:5px;background:#efedff;padding:5px 8px;color:#6758df;font-size:11px}.mock-multiple small{display:block;flex-basis:100%;font-size:10px;color:#999}.delete{align-self:flex-start;background:transparent;border:0;color:#b0afbe;font-size:18px}.submit-look{width:100%;padding:13px 16px;background:#6656ef;color:#fff;border:0;border-radius:8px;font-size:13px;font-weight:700;margin-top:14px}.submit-look span{float:right}.footnote{text-align:center;color:#a5a6b2;font-size:10px;margin:14px 0 0}.tabs{display:flex;border-bottom:1px solid #ededf3;padding:0 15px}.tabs button{flex:1;padding:20px 0 14px;background:none;border:0;border-bottom:2px solid transparent;font-size:11px;color:#a1a1af}.tabs button.active{color:#6656ef;border-color:#6656ef;font-weight:700}.inspector-body{padding:20px;overflow-y:auto;flex:1}.inspector-body>label{display:block;color:#747486;font-size:11px;font-weight:650;margin:15px 0}.inspector-body>label input:not([type=checkbox]),.inspector-body>label textarea,.inspector-body>label select{margin-top:7px}.inspector-body .check{display:flex;gap:8px;align-items:center}.selected-heading{display:flex;align-items:center;gap:11px;margin-bottom:20px}.selected-heading strong{font-size:12px}.selected-heading small{display:block;color:#a1a2af;text-transform:capitalize;margin-top:3px;font-size:10px}.option-row{display:flex;gap:5px;margin:7px 0}.option-row button{background:#fff;border:0;color:#b66;font-size:18px}.text-button{border:0;background:transparent;color:#6656ef;font-size:11px;padding:8px 0}.hint,.empty-settings{font-size:11px;color:#9899a8;line-height:1.5}.empty-settings{padding:30px}.save-area{border-top:1px solid #eeeef3;padding:17px;display:grid;gap:9px}.save-area small{font-size:10px;color:#9899a8}.preview-page{padding:30px 16px 65px}.preview-heading{max-width:630px;margin:0 auto 15px;display:flex;align-items:center;justify-content:space-between;color:#999baa;font-size:10px;font-weight:800;letter-spacing:.12em}.preview-card{max-width:630px}.public-link{max-width:630px;margin:0 auto 15px;display:flex;gap:12px;align-items:center;font-size:11px;color:#999}.public-link a{color:#6656ef;font-weight:700}.live-field{margin:17px 0}.live-field input:not([type=checkbox]):not([type=radio]),.live-field select{color:#454559;height:43px}.live-field input:focus,.live-field select:focus{outline:2px solid #bbb5ff}.check-options{display:grid;gap:8px}.check-options label{display:flex;align-items:center;gap:7px;font-size:12px;color:#5c5d70}.check-options input{accent-color:#6656ef}.live-field .multi-select{height:110px;padding:8px}.error,.error-message,.draft-warning{display:block;color:#be5267;font-size:11px;margin-top:7px}.success{color:#268763;font-size:11px;margin-top:12px}.honeypot{position:absolute;left:-9999px}.preview-card .description{margin-bottom:30px}.toast{position:fixed;bottom:20px;left:50%;transform:translateX(-50%);padding:12px 18px;border-radius:8px;background:#22243b;color:#fff;font-size:12px;box-shadow:0 7px 28px #0002;max-width:85vw}.toast.error-message{background:#9e304b;color:#fff}@media(max-width:1000px){.workspace{grid-template-columns:220px minmax(0,1fr)}.inspector{grid-column:1/-1;border-top:1px solid #e9e9f0}.canvas-area{padding:22px}}@media(max-width:650px){.workspace{display:block}.palette{border-bottom:1px solid #e9e9f0}.topbar{padding:0 12px}.brand small,.status{display:none}.canvas-area{padding:18px 12px}.form-inner{padding:26px}.palette-group{display:inline-block;vertical-align:top;width:48%}}
</style>