$ErrorActionPreference = "Stop"
$Repo = "latif-essam/app-dev-clean"
$Bin  = "app-dev-clean"
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$tag  = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
$url  = "https://github.com/$Repo/releases/download/$tag/${Bin}_windows_${arch}.zip"
$dest = "$env:LOCALAPPDATA\Programs\$Bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = "$env:TEMP\$Bin.zip"
Write-Host "Downloading $url"
Invoke-WebRequest $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dest -Force
Write-Host "Installed to $dest. Add it to PATH: setx PATH \"$env:PATH;$dest\""
