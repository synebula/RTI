//go:build linux || freebsd || netbsd || openbsd || darwin

package injector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"remote-text-input/internal/logger"
)

func ValidateRuntime() error {
	required := []string{"wl-copy", "hyprctl"}
	missing := make([]string, 0)
	for _, cmd := range required {
		if _, err := exec.LookPath(cmd); err != nil {
			missing = append(missing, cmd)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required command(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

type activeWindow struct {
	Class string `json:"class"`
	Title string `json:"title"`
}

func generateQRSVG(url string) (string, error) {
	if _, err := exec.LookPath("qrencode"); err != nil {
		return "", nil
	}
	cmd := exec.Command("qrencode", "-t", "SVG", "-o", "-", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qrencode failed: %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
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
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		logger.Event("open-browser-failed", "url", fmt.Sprintf("%q", url), "error", fmt.Sprintf("%q", err.Error()))
		return
	}
	logger.Event("open-browser", "url", fmt.Sprintf("%q", url))
}

func (i *Injector) readClipboard() (string, error) {
	result, err := i.runCommand([]string{"wl-paste", "--no-newline"}, "")
	if i.dryRun {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

func (i *Injector) writeClipboard(text string) error {
	if i.dryRun {
		_, err := i.runCommand([]string{"wl-copy"}, text)
		return err
	}
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return &InjectionError{msg: "wl-copy failed"}
	}
	return nil
}

func (i *Injector) dispatchShortcut(modifiers, key string) error {
	shortcut := fmt.Sprintf("%s,%s,", modifiers, key)
	if modifiers == "" {
		shortcut = fmt.Sprintf(",%s,", key)
	}
	_, err := i.runCommand([]string{"hyprctl", "dispatch", "sendshortcut", shortcut}, "")
	return err
}

func (i *Injector) dispatchTerminalPaste() error {
	if _, err := exec.LookPath("wtype"); err == nil {
		_, err := i.runCommand([]string{"wtype", "-M", "ctrl", "-M", "shift", "-k", "v", "-m", "shift", "-m", "ctrl"}, "")
		return err
	}
	return i.dispatchShortcut("CTRL_SHIFT", "V")
}

func (i *Injector) isTerminalFocused() bool {
	window, err := i.activeWindow()
	if err != nil {
		return false
	}
	identity := strings.ToLower(window.Class + " " + window.Title)
	for _, marker := range []string{
		"kitty",
		"wezterm",
		"ghostty",
		"alacritty",
		"foot",
		"gnome-terminal",
		"terminal",
		"xterm",
		"konsole",
		"tilix",
		"rio",
		"tmux",
	} {
		if strings.Contains(identity, marker) {
			return true
		}
	}
	return false
}

func (i *Injector) activeWindow() (activeWindow, error) {
	output, err := i.runCommand([]string{"hyprctl", "-j", "activewindow"}, "")
	if err != nil {
		return activeWindow{}, err
	}
	if i.dryRun || output == "" {
		return activeWindow{}, nil
	}
	var window activeWindow
	if err := json.Unmarshal([]byte(output), &window); err != nil {
		return activeWindow{}, &InjectionError{msg: "failed to parse active window"}
	}
	return window, nil
}

func (i *Injector) runCommand(args []string, stdin string) (string, error) {
	if len(args) == 0 {
		return "", &InjectionError{msg: "empty command"}
	}
	if i.dryRun {
		fmt.Println("[dry-run]", shellJoin(args))
		if i.logEnabled && stdin != "" {
			preview := stdin
			if len(preview) > 80 {
				preview = preview[:77] + "..."
			}
			fmt.Printf("[dry-run] stdin=%q\n", preview)
		}
		return "", nil
	}
	cmd := exec.Command(args[0], args[1:]...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return "", &InjectionError{msg: trimmed}
	}
	return strings.TrimSpace(string(output)), nil
}

func shellJoin(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, strconvQuote(part))
	}
	return strings.Join(quoted, " ")
}

func strconvQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"$`\\()[]{}*?<>|&;!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
