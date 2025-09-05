package ptyx

import (
	"testing"
	"time"
)

func TestConsole_Close(t *testing.T) {
	c, err := NewConsole()
	if err != nil {
		t.Fatalf("NewConsole() failed: %v", err)
	}

	select {
	case <-c.OnResize():
	case <-time.After(500 * time.Millisecond):
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	select {
	case _, ok := <-c.OnResize():
		if ok {
			t.Error("OnResize() channel was not closed after Close()")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("OnResize() channel did not close within 1s after Close()")
	}

	if err := c.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}
