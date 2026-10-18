import { execFileSync } from "child_process";
import * as fs from "fs";
import * as paths from "../../paths";

// The task is registered only when a saved Steam shortcut opts into recovery.
// It runs with the current user's interactive token, without elevation.
const registerScript = `
$ErrorActionPreference = 'Stop'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$name = 'SRM Store Agent ' + $identity.User.Value
$action = New-ScheduledTaskAction -Execute $env:SRM_STORE_AGENT_PATH
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $identity.Name
$principal = New-ScheduledTaskPrincipal -UserId $identity.Name -LogonType Interactive -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName $name -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
if (-not (Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'srm-store-agent.exe' })) {
  Start-ScheduledTask -TaskName $name
}
Write-Output $name
`;

function run(script: string): void {
  execFileSync("powershell.exe", [
    "-NoProfile", "-NonInteractive", "-EncodedCommand",
    Buffer.from(script, "utf16le").toString("base64"),
  ], {
    env: { ...process.env, SRM_STORE_AGENT_PATH: paths.storeAgent },
    windowsHide: true,
    timeout: 30000,
  });
}

export function ensureStoreAgentTask(): void {
  if (process.platform !== "win32") return;
  if (!fs.existsSync(paths.storeAgent)) {
    throw new Error(`SRM store agent is missing: ${paths.storeAgent}`);
  }
  run(registerScript);
}

export function refreshStoreAgentTaskIfInstalled(): void {
  if (process.platform !== "win32" || !fs.existsSync(paths.storeAgent)) return;
  run(`
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$name = 'SRM Store Agent ' + $identity.User.Value
if (Get-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue) {
${registerScript}
}
`);
}
