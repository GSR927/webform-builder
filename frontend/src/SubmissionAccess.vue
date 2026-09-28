<template>
  <main class="page">
    <header class="topbar"><div class="brand">✦ Formly</div><nav><a :href="`/index.html?formId=${formId}`">← Form editor</a><button @click="logout">Sign out</button></nav></header>
    <section class="content">
      <div class="eyebrow">CUSTOMER WORKSPACE / FORM {{ formId }}</div>
      <div class="heading"><div><h1>{{ formName }} · submissions</h1><p>Responses are displayed with labels from the published version each visitor used.</p></div><span class="total">{{ total }} responses</span></div>
      <div class="toolbar"><input v-model.trim="query" placeholder="Search response content (press Enter)" @keyup.enter="search" /><button @click="search">Search</button><button @click="download">Download CSV</button><button @click="load">Refresh</button></div>
      <p v-if="error" role="alert" class="error">{{ error }}</p>
      <p v-if="loading" role="status" class="loading">Loading submissions…</p>
      <div v-else-if="!items.length" class="empty">No submissions match this filter.</div>
      <div v-else class="table-wrap"><table>
        <thead><tr><th>Received</th><th v-for="column in columns" :key="column.key">{{ column.label }}</th><th>Form version</th><th>Details</th></tr></thead>
        <tbody>
			<template v-for="item in items" :key="item.id">
			<tr>
				<td class="date">
				{{ formatDate(item.submittedAt) }}
				</td>
			
				<td
				v-for="column in columns"
				:key="column.key"
				class="value"
				:title="display(item.payload?.[column.key])"
				>
				{{ display(item.payload?.[column.key]) }}
				</td>
			
				<td>
				<span class="badge">
					v{{ versionNumber(item) }}
				</span>
				</td>
			
				<td>
				<button
					class="details"
					@click="expanded = expanded === item.id ? null : item.id"
				>
					{{ expanded === item.id ? "Close" : "View" }}
				</button>
				</td>
			</tr>
			
			<tr v-if="expanded === item.id" class="detail-row">
				<td :colspan="columns.length + 3">
				<div class="detail-head">
					<strong>Submission {{ item.id }}</strong>
					<span>Version ID {{ item.formVersionId }}</span>
				</div>
			
				<div
					v-for="[key, value] in Object.entries(item.payload || {})"
					:key="key"
					class="detail-pair"
				>
					<span>{{ labelFor(key, item.formVersionId) }}</span>
					<strong>{{ display(value) }}</strong>
				</div>
				</td>
			</tr>
			</template>
		</tbody></table></div>
      <footer><span>Page {{ page }} · {{ total }} total</span><button :disabled="page === 1 || loading" @click="page--; load()">← Previous</button><button :disabled="page * pageSize >= total || loading" @click="page++; load()">Next →</button></footer>
    </section>
  </main>
</template>

<script>
export default {
  name: 'SubmissionAccess',
  props: { formId: { type: String, required: true } },
  data() { return { items: [], total: 0, page: 1, pageSize: 25, query: '', loading: false, error: '', formName: 'Form', columns: [], versionSchemas: {}, expanded: null }; },
  mounted() { this.load(); },
  methods: {
    async load() {
      this.loading = true; this.error = ''; this.expanded = null;
      try {
        const response = await fetch(`/api/forms/${this.formId}/submissions?page=${this.page}&pageSize=${this.pageSize}&q=${encodeURIComponent(this.query)}`, { cache: 'no-store' });
        if (response.status === 401) { location.href = `/login.html?next=${encodeURIComponent(location.pathname + location.search)}`; return; }
        if (!response.ok) throw new Error(`Could not load submissions (HTTP ${response.status}).`);
        const data = await response.json();
        this.items = data.items || []; this.total = data.total || 0; this.formName = data.formName || 'Form';
        this.versionSchemas = data.versionSchemas || {};
        this.columns = this.buildColumns(this.items, this.versionSchemas);
      } catch (e) { this.error = e.message; } finally { this.loading = false; }
    },
    buildColumns(items, schemas) {
      const keys = new Set(); const columns = [];
      for (const item of items) {
        const schema = schemas[String(item.formVersionId)];
        for (const field of schema?.fields || []) {
          if (!keys.has(field.key)) { keys.add(field.key); columns.push({ key: field.key, label: field.label || field.key }); }
        }
        for (const key of Object.keys(item.payload || {})) {
          if (!keys.has(key)) { keys.add(key); columns.push({ key, label: this.labelFor(key, item.formVersionId) }); }
        }
      }
      return columns;
    },
    labelFor(key, versionId) { return this.versionSchemas[String(versionId)]?.fields?.find(field => field.key === key)?.label || key; },
    versionNumber(item) { return this.versionSchemas[String(item.formVersionId)]?.versionNumber || item.formVersionId; },
    display(value) { if (value === undefined || value === null || value === '') return '—'; if (Array.isArray(value)) return value.length ? value.join(', ') : '—'; if (typeof value === 'object') return JSON.stringify(value); return String(value); },
    formatDate(value) { return value ? new Date(value).toLocaleString() : '—'; },
    search() { this.page = 1; this.load(); },
    download() { location.href = `/api/forms/${this.formId}/submissions/export?q=${encodeURIComponent(this.query)}`; },
    async logout() {
      const entry = document.cookie.split('; ').find(value => value.startsWith('webform_csrf='));
      const csrf = entry ? decodeURIComponent(entry.slice('webform_csrf='.length)) : '';
      await fetch('/api/logout', { method: 'POST', headers: { 'X-CSRF-Token': csrf } });
      location.href = '/login.html';
    }
  }
};
</script>

<style scoped>
:global(*){box-sizing:border-box}:global(body){margin:0;font-family:Inter,system-ui,-apple-system,'Segoe UI',sans-serif;background:#f7f8fc;color:#25253b}:global(button),:global(input){font:inherit}.topbar{height:72px;background:#fff;border-bottom:1px solid #e7e8ef;display:flex;justify-content:space-between;align-items:center;padding:0 32px}.brand{font-weight:800;font-size:18px;color:#6556ef}nav{display:flex;gap:18px;align-items:center;font-size:12px}a{color:#6656ef}button{cursor:pointer;background:#fff;border:1px solid #dedee8;border-radius:7px;padding:8px 12px;color:#626375}.content{max-width:1140px;margin:0 auto;padding:42px 24px}.eyebrow{color:#6656ef;font-size:10px;font-weight:800;letter-spacing:.13em}.heading{display:flex;justify-content:space-between;align-items:center}.heading h1{font-size:27px;margin:11px 0 6px}.heading p{color:#8f90a0;font-size:12px;margin:0}.total{background:#eeebff;color:#6656ef;padding:8px 12px;border-radius:12px;font-size:11px;font-weight:800}.toolbar{display:flex;gap:9px;margin:25px 0}.toolbar input{flex:1;min-width:0;border:1px solid #e0e1eb;border-radius:8px;padding:10px 12px}.toolbar button{white-space:nowrap}.error{color:#b33;font-size:12px}.loading,.empty{background:#fff;border-radius:10px;padding:35px;color:#999;font-size:12px}.table-wrap{overflow:auto;background:#fff;border:1px solid #e7e8ef;border-radius:11px}table{width:100%;border-collapse:collapse;font-size:12px}th{text-align:left;background:#fafaff;color:#8d8e9e;font-size:10px;letter-spacing:.06em;white-space:nowrap}th,td{border-bottom:1px solid #eff0f5;padding:13px 15px}td{max-width:260px}.date{color:#848596;white-space:nowrap}.value{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.badge{background:#eeebff;color:#6656ef;padding:4px 8px;border-radius:6px;font-size:10px;font-weight:700}.details{font-size:11px}.detail-row td{background:#fafaff;padding:18px 25px}.detail-head{display:flex;justify-content:space-between;gap:15px;margin-bottom:13px;color:#68697b;font-size:11px}.detail-pair{display:grid;grid-template-columns:160px 1fr;gap:15px;padding:6px 0;font-size:12px}.detail-pair span{color:#8c8d9e}.detail-pair strong{font-weight:600;overflow-wrap:anywhere}footer{display:flex;gap:9px;align-items:center;justify-content:flex-end;margin-top:18px;font-size:11px;color:#8c8d9e}footer span{margin-right:auto}button:disabled{opacity:.45;cursor:not-allowed}@media(max-width:720px){.content{padding:25px 12px}.toolbar{flex-wrap:wrap}.toolbar input{flex-basis:100%}.heading{display:block}.total{display:inline-block;margin-top:12px}}
</style>