// Package signalman provides a convenient way to register functions to be
// executed when operating signals are received.
//
// A typical application of this package might be to run cleanup logic when
// your OS instructs your application it is going to kill it.
//
// A global Signalman is initialised within the package, and typical
// usage can involve using the package level functions. However, it's
// possible to create many Signalmen, and utilise the methods defined on
// the Signalman type for each one.
//
// Any function matching the SignalFunc signature may be registered on a
// Signalman, using Register or RegisterMap. Each function can be
// registered against any signal that implements the os.Signal interface.
// Each function will be executed in its own goroutine when its Signalman
// receives an appropriate signal.
//
// Start will instruct a Signalman to start
// listening for incoming signals.
//
// 		func f() error {
//			// clean up logic here
//		}
//
//		signalman.Register(os.Interrupt, f)
//		signalman.Start()
//
// At any point, an error channel can be registered on a Signalman, and
// any errors returned from registered functions will be, sent along this
// channel.
//
//		errCh := make(chan error)
//		signalman.SetErrChannel(errCh)
//
// Since a Signalman runs each registered function in its own goroutine,
// it's not necessary to provide a buffered channel for receiving errors.
package signalman

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
)

type SignalFunc func() error

// A Signalman provides methods for registering functions to be executed
// when the Signalman receives signals.
type Signalman struct {
	sc       chan os.Signal
	ec       chan error
	handlers map[os.Signal][]SignalFunc
	started  bool
	mu       *sync.Mutex
}

// By default, the signalman package provides a global Signalman.
//
// Unless multiple Signalmen are required, this is the easiest way to
// utilise the signalman package.
var std = New()

// New creates a new Signalman, which can then be used for registering
// functions against signals.
//
// Its methods are safe for concurrent access via multiple goroutines.
func New() *Signalman {
	return &Signalman{
		sc:       make(chan os.Signal, 1),
		handlers: make(map[os.Signal][]SignalFunc),
		mu:       &sync.Mutex{},
	}
}

// SetErrChannel instructs the Signalman to send non-nil errors returned
// from registered functions on the provided channel.
func (s *Signalman) SetErrChannel(ec chan error) {
	s.mu.Lock()
	s.ec = ec
	s.mu.Unlock()
}

// SetErrChannel instructs the global Signalman to send non-nil errors
// returned from registered functions on the provided channel.
func SetErrChannel(ec chan error) {
	std.SetErrChannel(ec)
}

func notify(sig os.Signal, sc chan os.Signal) {
	if sig == nil {
		// all signals are to be sent on channel
		signal.Notify(sc)
	} else {
		signal.Notify(sc, sig)
	}
}

// Register one or more SignalFuncs against an os.Signal.
//
// If sig is nil, all signals received by the Signalman will result in
// the provided SignalFuncs being executed.
func (s *Signalman) Register(sig os.Signal, fun ...SignalFunc) {
	notify(sig, s.sc)
	s.mu.Lock()
	h, ok := s.handlers[sig]
	if !ok {
		s.handlers[sig] = fun
	} else {
		for _, f := range fun {
			s.handlers[sig] = append(h, f)
		}
	}
	s.mu.Unlock()
}

// Register registers one or more SignalFuncs against an os.Signal.
//
// If sig is nil, all signals received by the global Signalman
// will result in the provided SignalFuncs being executed.
func Register(sig os.Signal, fun ...SignalFunc) {
	std.Register(sig, fun...)
}

// RegisterMap registers multiple SignalFuncs against signals, on the
// Signalman.
//
// A nil key will result in the Signalman executing the functions
// associated with the nil key, for all received signals.
func (s *Signalman) RegisterMap(signals map[os.Signal][]SignalFunc) {
	s.mu.Lock()
	for sig, handlers := range signals {
		notify(sig, s.sc)
		h, ok := s.handlers[sig]
		if !ok {
			s.handlers[sig] = handlers
		} else {
			for _, handler := range handlers {
				s.handlers[sig] = append(h, handler)
			}
		}
	}
	s.mu.Unlock()
}

// RegisterMap registers multiple SignalFuncs against signals.
//
// A nil key will result in the global Signalman executing the functions
// associated with the nil key, for all received signals.
func RegisterMap(signals map[os.Signal][]SignalFunc) {
	std.RegisterMap(signals)
}

func (s *Signalman) handleSignal(sig os.Signal) {
	funcs, ok := s.handlers[sig]
	if !ok && s.ec != nil {
		s.ec <- fmt.Errorf("Signal %v has no registered handlers.", sig)
		return
	}

	for _, f := range funcs {
		go func() {
			if err := f(); err != nil && s.ec != nil {
				s.ec <- err
			}
		}()
	}
}

// Start instructs the Signalman to begin listening for incoming signals.
func (s *Signalman) Start() {
	s.mu.Lock()
	if !s.started {
		s.started = true
		go func() {
			for sig := range s.sc {
				s.handleSignal(sig)
			}
		}()
	}
	s.mu.Unlock()
}

// Start instructs the global Signalman to begin listening for incoming signals.
func Start() {
	std.Start()
}

// Stop instructs the Signalman to stop listening for incoming signals.
//
// When Stop is called, all mapped SignalFuncs are removed. If Start is
// called in the future, SignalFuncs will need to be registered again
// before they're executed.
func (s *Signalman) Stop() {
	// Stop all signals being sent on channel.
	signal.Stop(s.sc)
	// Destroy handler mapping since signals are no longer registered.
	s.mu.Lock()
	s.handlers = make(map[os.Signal][]SignalFunc)
	s.mu.Unlock()
	// Close the channel
	close(s.sc)
}

// Stop instructs the global Signalman to stop listening for incoming signals.
//
// When Stop is called, all mapped SignalFuncs are removed. If Start is
// called in the future, SignalFuncs will need to be registered again
// before they're executed.
func Stop() {
	std.Stop()
}
