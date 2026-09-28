param([string]$Project = (Get-Location).Path)
$ErrorActionPreference='Stop'
$target=Join-Path $Project 'frontend\src\FormBuilder.vue'
if(!(Test-Path $target)){ throw "Not found: $target. Run from webform-builder root." }
$text=[System.IO.File]::ReadAllText($target)
if($text.Contains('function csrfHeader(')){Write-Host 'Builder patch already applied';exit 0}
$marker='<script>'
$inject=@'
<script>
function csrfHeader() {
  const entry = document.cookie.split('; ').find(value => value.startsWith('webform_csrf='));
  return entry ? decodeURIComponent(entry.substring('webform_csrf='.length)) : '';
}
'@
if(!$text.Contains($marker)){throw 'Missing <script> in FormBuilder.vue'}
$text=$text.Replace($marker,$inject)
$text=$text.Replace("method: 'PUT', headers: { 'Content-Type': 'application/json' }","method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfHeader() }")
$before=@'
fetch(`/api/forms/${this.formId}/publish`, { method: 'POST' })
'@.Trim()
$after=@'
fetch(`/api/forms/${this.formId}/publish`, { method: 'POST', headers: { 'X-CSRF-Token': csrfHeader() } })
'@.Trim()
$text=$text.Replace($before,$after)
$old='</header>'
$new='<a class="customer-link" :href="`/submissions.html?formId=${formId}`">View submissions ↗</a></header>'
if($text.Contains($old)){$text=$text.Replace($old,$new)}
$text=$text.Replace('<style scoped>','<style scoped>.customer-link{color:#6656ef;text-decoration:none;font-size:12px;font-weight:700;margin:0 10px}')
[System.IO.File]::WriteAllText($target,$text,[System.Text.UTF8Encoding]::new($false))
Write-Host "Patched $target. Confirm the generated frontend still builds."
