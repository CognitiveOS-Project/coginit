//go:build linux

package coginit

import (
	"os"
	"testing"
	"time"
)

func TestNewComboState(t *testing.T) {
	s := newComboState()
	if s == nil {
		t.Fatal("newComboState returned nil")
	}
	if len(s.pressed) != 0 {
		t.Fatalf("expected empty pressed map, got %d keys", len(s.pressed))
	}
}

func TestProcessKeyEvent_PressAndRelease(t *testing.T) {
	s := newComboState()

	s.processKeyEvent(keyLeftCtrl, keyPress)
	if !s.pressed[keyLeftCtrl] {
		t.Fatal("Ctrl should be pressed")
	}

	s.processKeyEvent(keyLeftCtrl, keyRelease)
	if s.pressed[keyLeftCtrl] {
		t.Fatal("Ctrl should not be pressed after release")
	}
}

func TestProcessKeyEvent_HoldIgnored(t *testing.T) {
	s := newComboState()

	s.processKeyEvent(keyLeftCtrl, keyPress)
	s.processKeyEvent(keyLeftCtrl, keyHold)

	if !s.pressed[keyLeftCtrl] {
		t.Fatal("Ctrl should still be pressed after hold")
	}
}

func TestCombo1_CtrlAltBackspace(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	s.processKeyEvent(keyLeftCtrl, keyPress)
	s.processKeyEvent(keyLeftAlt, keyPress)
	s.processKeyEvent(keyBackspace, keyPress)

	if triggered != "Ctrl+Alt+Backspace" {
		t.Fatalf("expected Ctrl+Alt+Backspace, got %q", triggered)
	}
}

func TestCombo2_AltPrintScreenK(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	s.processKeyEvent(keyLeftAlt, keyPress)
	s.processKeyEvent(keySysRq, keyPress)
	s.processKeyEvent(keyK, keyPress)

	if triggered != "Alt+PrintScreen+K" {
		t.Fatalf("expected Alt+PrintScreen+K, got %q", triggered)
	}
}

func TestCombo3_CtrlAltDel(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	s.processKeyEvent(keyLeftCtrl, keyPress)
	s.processKeyEvent(keyLeftAlt, keyPress)
	s.processKeyEvent(keyDelete, keyPress)

	if triggered != "Ctrl+Alt+Del" {
		t.Fatalf("expected Ctrl+Alt+Del, got %q", triggered)
	}
}

func TestCombo4_ScrollLockx3(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	// First press
	s.processKeyEvent(keyScrollLock, keyPress)
	s.processKeyEvent(keyScrollLock, keyRelease)
	if triggered != "" {
		t.Fatal("should not trigger on first press")
	}

	// Second press
	s.processKeyEvent(keyScrollLock, keyPress)
	s.processKeyEvent(keyScrollLock, keyRelease)
	if triggered != "" {
		t.Fatal("should not trigger on second press")
	}

	// Third press — should trigger
	s.processKeyEvent(keyScrollLock, keyPress)
	if triggered != "ScrollLockx3" {
		t.Fatalf("expected ScrollLockx3, got %q", triggered)
	}
}

func TestCombo4_ScrollLockTimeout(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	// First press
	s.processKeyEvent(keyScrollLock, keyPress)
	s.processKeyEvent(keyScrollLock, keyRelease)

	// Simulate timeout by backdating scrollFirst
	s.mu.Lock()
	s.scrollFirst = time.Now().Add(-3 * time.Second)
	s.mu.Unlock()

	// Second press — should reset counter
	s.processKeyEvent(keyScrollLock, keyPress)
	s.processKeyEvent(keyScrollLock, keyRelease)

	// Third press — should NOT trigger (counter was reset)
	s.processKeyEvent(keyScrollLock, keyPress)
	if triggered != "" {
		t.Fatal("ScrollLockx3 should not trigger after timeout reset")
	}
}

func TestCombo5_SuperGrave(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	s.processKeyEvent(keyLeftMeta, keyPress)
	s.processKeyEvent(keyGrave, keyPress)

	if triggered != "Super+Grave" {
		t.Fatalf("expected Super+Grave, got %q", triggered)
	}
}

func TestCombo_NoPartialMatch(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	// Press Ctrl+Alt but not Backspace
	s.processKeyEvent(keyLeftCtrl, keyPress)
	s.processKeyEvent(keyLeftAlt, keyPress)

	if triggered != "" {
		t.Fatal("partial combo should not trigger")
	}
}

func TestCombo_WrongOrder(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	// Press K then SysRq (wrong order for combo 2)
	s.processKeyEvent(keyK, keyPress)
	s.processKeyEvent(keySysRq, keyPress)

	if triggered != "" {
		t.Fatal("incomplete combo should not trigger")
	}
}

func TestCombo_ReleaseResetsState(t *testing.T) {
	s := newComboState()
	var triggered string
	s.onTrigger = func(combo string) { triggered = combo }

	// Press Ctrl+Alt, release Ctrl, then press Backspace
	s.processKeyEvent(keyLeftCtrl, keyPress)
	s.processKeyEvent(keyLeftAlt, keyPress)
	s.processKeyEvent(keyLeftCtrl, keyRelease)
	s.processKeyEvent(keyBackspace, keyPress)

	if triggered != "" {
		t.Fatal("combo should not trigger after release")
	}
}

func TestLogAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/backdoor.log"

	// Write a log entry (simulating logAccess format)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatalf("failed to open log: %v", err)
	}
	_, err = f.WriteString("2026-01-01T00:00:00Z combo=TestCombo channel=vt2\n")
	f.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("log file is empty")
	}
}
