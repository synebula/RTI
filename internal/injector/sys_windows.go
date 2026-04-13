//go:build windows

package injector

import (
	"fmt"
	"os/exec"
	"strings"
	"remote-text-input/internal/logger"
	"remote-text-input/internal/util"
)

func ValidateRuntime() error {
	// Windows native dependencies are usually present (powershell/cmd), so no explicit check required.
	return nil
}

func PrintTerminalQR(url string) {
	if _, err := exec.LookPath("qrencode"); err != nil {
		fmt.Println("QR Code  : qrencode not found, skipping terminal QR")
		return
	}
	cmd := exec.Command("qrencode", "-t", "ANSIUTF8", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("QR Code  : failed to render in terminal")
		return
	}
	fmt.Println("Phone QR  :")
	fmt.Print(strings.TrimRight(string(output), "\n"))
	fmt.Println()
}

func OpenBrowserPage(url string) {
	cmd := exec.Command("cmd", "/c", "start", url)
	if err := cmd.Start(); err != nil {
		logger.Event("open-browser-failed", "url", fmt.Sprintf("%q", url), "error", fmt.Sprintf("%q", err.Error()))
		return
	}
	logger.Event("open-browser", "url", fmt.Sprintf("%q", url))
}

func (i *Injector) readClipboard() (string, error) {
	if i.dryRun {
		return "", nil
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	str := string(out)
	if strings.HasSuffix(str, "\r\n") {
		str = str[:len(str)-2]
	} else if strings.HasSuffix(str, "\n") {
		str = str[:len(str)-1]
	}
	return str, nil
}

func (i *Injector) writeClipboard(text string) error {
	if i.dryRun {
		fmt.Printf("[dry-run] set clipboard: %s\n", util.FormatTextPreview(text, 80))
		return nil
	}
	scriptSimpler := `
Add-Type -AssemblyName System.Windows.Forms
$text = [Console]::In.ReadToEnd()
if ($text) {
    [System.Windows.Forms.Clipboard]::SetText($text)
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "$OutputEncoding = [System.Text.Encoding]::UTF8; [Console]::InputEncoding = [System.Text.Encoding]::UTF8; " + scriptSimpler)

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return &InjectionError{msg: fmt.Sprintf("powershell clipboard write failed: %v", err)}
	}
	return nil
}

func (i *Injector) dispatchShortcut(modifiers, key string) error {
	var keys string
	if modifiers == "CTRL" && key == "V" {
		keys = "^v"
	} else if key == "RETURN" {
		keys = "{ENTER}"
	} else {
		return fmt.Errorf("unsupported key: %s %s", modifiers, key)
	}

	if i.dryRun {
		fmt.Printf("[dry-run] shortcut %s\n", keys)
		return nil
	}

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait('%s')
`, keys)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	err := cmd.Run()
	if err != nil {
		return &InjectionError{msg: fmt.Sprintf("powershell SendKeys failed: %v", err)}
	}
	return nil
}
