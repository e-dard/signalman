Signalman
=========

Signalman provides a convenient way to register functions to be
executed when operating signals are received.

A typical use for signalman might be to run cleanup logic when
your process supervisor instructs your application it is going to kill it.

The signalman package provides a package-level `Signalman` value. Typically this is all you'll
need. However, it's possible to create many `Signalmen`, and utilise the methods defined on
the `Signalman` type for each one.

Any function matching the `SignalFunc` signature may be registered on a
`Signalman`, using `Register` or `RegisterMap`. Each function can be
registered against any signal that implements the `os.Signal` interface.

### Example Usage

Assume some function that carries out cleanup work should the application
need to be killed.

```go
func f() error {
   // clean up logic here
}
```

All `Signalman` functions need to satisfy the `SignalFunc` type. This function can be 
registered on the global `Signalman` using the package-level functions:

```go
// Register f to be executed when a SIGUSR1 signal is received.
signalman.Register(os.Interrupt, f)

// Instruct the Signalman to begin listening for this signal.
signalman.Start()
```

All functions registered on a `Signalman` are run in their own goroutines, so `Signalman`
allows you to provide an error channel to receive any errors on.

```go
errCh := make(chan error)
signalman.SetErrChannel(errCh)
```

Since a Signalman runs each registered function in its own goroutine,
it's not necessary to provide a buffered channel for receiving errors.
