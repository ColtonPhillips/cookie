# Start ngrok in the background
$ngrokProcess = Start-Process -NoNewWindow -PassThru -FilePath "ngrok.exe" -ArgumentList "http 8080"

# Wait a moment for ngrok to initialize
Start-Sleep -Seconds 2

# Retry until the tunnel appears (max 10 tries)
$maxRetries = 10
$publicUrl = $null
for ($i = 0; $i -lt $maxRetries; $i++) {
    try {
        $response = Invoke-RestMethod http://127.0.0.1:4040/api/tunnels
        if ($response.tunnels.Count -gt 0) {
            $publicUrl = $response.tunnels[0].public_url
            break
        }
    }
    catch {
        Start-Sleep -Seconds 1
    }
}

if ($null -eq $publicUrl) {
    Write-Error "Failed to retrieve ngrok URL after $maxRetries attempts."
    Stop-Process -Id $ngrokProcess.Id
    exit 1
}

# Construct the final IGM URL
$igmUrl = "https://orteil.dashnet.org/igm/?g=$publicUrl/cookie/build/main.igm"

# Copy it to clipboard
Set-Clipboard -Value $igmUrl

# Optional: Save PID in case you want to stop ngrok later
$ngrokProcess.Id | Out-File "ngrok.pid"

# Wait for ngrok to finish (script stays alive)
Wait-Process -Id $ngrokProcess.Id
