package main

import "strings"

func defaultWindowsPickerDirectory() string {
	return `C:\`
}

func windowsPowerShellPickerScript(initialDir string) string {
	return strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms",
		"$dialog = New-Object System.Windows.Forms.FolderBrowserDialog",
		"$dialog.Description = '请选择允许 CLI_SANDBOX 访问的目录'",
		"$dialog.ShowNewFolderButton = $false",
		"$dialog.SelectedPath = '" + strings.ReplaceAll(initialDir, `'`, `''`) + "'",
		"$result = $dialog.ShowDialog()",
		"if ($result -eq [System.Windows.Forms.DialogResult]::OK -and -not [string]::IsNullOrWhiteSpace($dialog.SelectedPath)) {",
		"    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8",
		"    Write-Output $dialog.SelectedPath",
		"    exit 0",
		"}",
		"exit 1",
	}, "; ")
}
