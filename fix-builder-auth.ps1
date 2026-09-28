$ErrorActionPreference = 'Stop'
$path = Join-Path (Get-Location).Path 'frontend\src\FormBuilder.vue'
if (-not (Test-Path $path)) { throw "Run from webform-builder root; missing $path" }
$text = [System.IO.File]::ReadAllText($path)
$badLink = @'
:href="/submissions.html?formId=${formId}"
'@.Trim()
$goodLink = @'
:href="`/submissions.html?formId=${formId}`"
'@.Trim()
$badPublish = @'
fetch(/api/forms/${this.formId}/publish, {
'@.Trim()
$goodPublish = @'
fetch(`/api/forms/${this.formId}/publish`, {
'@.Trim()
if (-not $text.Contains($badLink)) { throw 'Expected damaged submissions link not found; stop and inspect file.' }
if (-not $text.Contains($badPublish)) { throw 'Expected damaged publish fetch not found; stop and inspect file.' }
$text = $text.Replace($badLink, $goodLink).Replace($badPublish, $goodPublish)
[System.IO.File]::WriteAllText($path, $text, [System.Text.UTF8Encoding]::new($false))
Write-Host 'Corrected submissions link and publish URL. Rebuild frontend next.'
