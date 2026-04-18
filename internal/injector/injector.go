package injector

import (
	"sync"
	"time"

	"remote-text-input/internal/util"
)

type InjectionError struct{ msg string }

func (e *InjectionError) Error() string { return e.msg }

type Injector struct {
	logEnabled bool
	pasteDelay time.Duration
	mu         sync.Mutex
}

func New(logEnabled bool, pasteDelay time.Duration) *Injector {
	return &Injector{
		logEnabled: logEnabled,
		pasteDelay: pasteDelay,
	}
}

func (i *Injector) CommitText(text string, pressEnter bool, preferTerminalPaste bool) (string, error) {
	if text == "" {
		return "noop", nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	previous, _ := i.readClipboard()
	if err := i.writeClipboard(text); err != nil {
		return "", err
	}

	mode := "pasted"
	if preferTerminalPaste {
		if err := i.dispatchTerminalPaste(); err != nil {
			return "", err
		}
		mode = "terminal-pasted"
	} else {
		if err := i.dispatchShortcut("CTRL", "V"); err != nil {
			return "", err
		}
	}

	time.Sleep(i.pasteDelay)
	if pressEnter {
		if err := i.dispatchShortcut("", "RETURN"); err != nil {
			return "", err
		}
	}

	go i.restoreClipboardLater(previous)
	return mode, nil
}

func (i *Injector) SendEnter() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.dispatchShortcut("", "RETURN")
}

func (i *Injector) restoreClipboardLater(text string) {
	time.Sleep(util.MaxDuration(i.pasteDelay, 200*time.Millisecond))
	_ = i.writeClipboard(text)
}
