package signalman

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func Test_Register(t *testing.T) {
	sm := New()

	f := func() error { return nil }

	sm.Register(os.Interrupt, f)
	actual := sm.handlers[os.Interrupt]
	if len(actual) != 1 {
		t.Fatalf("expected %v\ngot %v", 1, len(actual))
	}

	sm.Register(os.Interrupt, f)
	actual = sm.handlers[os.Interrupt]
	if len(actual) != 2 {
		t.Fatalf("expected %v\ngot %v", 2, len(actual))
	}
}

func Test_RegisterMap(t *testing.T) {
	t.Skip("pending")
}

func Test_handleSignal(t *testing.T) {
	sm := New()
	ec := make(chan error, 1)
	sm.SetErrChannel(ec)

	// handle unregistered signal
	sm.handleSignal(os.Interrupt)

	// read from error channel
	expected := "Signal interrupt has no registered handlers."
	err := <-ec
	if err == nil {
		t.Fatal("Error is nil.")
	}

	if err.Error() != expected {
		t.Fatalf("expected %v\ngot %v", expected, err.Error())
	}

	// register signal and handlers on signalman
	i := 0
	f := func() error { i++; return nil }
	g := func() error { i++; return fmt.Errorf("error function") }
	sm.Register(os.Interrupt, f, g)

	// Second function should return an error, which is sent on the
	// error channel.
	sm.handleSignal(os.Interrupt)
	// Not pleasant, but much less code than a select with timeout.
	// Needed since the test check will complete before the error is
	// sent on the channel.
	time.Sleep(time.Millisecond * 5)

	if len(ec) != 1 {
		t.Fatal("Wrong number of errors received on channel")
	}

	if i != 2 {
		t.Fatalf("expected %v\ngot %v", 2, i)
	}
}
