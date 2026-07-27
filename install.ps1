$ErrorActionPreference = "Stop"
$Repo = "latif-essam/app-dev-clean"
$Bin  = "app-dev-clean"
# Only amd64 and arm64 are built for Windows (see .goreleaser.yaml).
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$tag  = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
$url  = "https://github.com/$Repo/releases/download/$tag/${Bin}_windows_${arch}.zip"
$dest = "$env:LOCALAPPDATA\Programs\$Bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = "$env:TEMP\$Bin.zip"
Write-Host "Downloading $url"
Invoke-WebRequest $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dest -Force
# Short alias: adc.cmd forwards to the exe next to it (%~dp0 = this file's dir).
Set-Content -Path "$dest\adc.cmd" -Value "@`"%~dp0$Bin.exe`" %*" -Encoding ASCII
Write-Host "Installed to $dest (alias: adc). Add it to PATH: setx PATH `"$env:PATH;$dest`""
