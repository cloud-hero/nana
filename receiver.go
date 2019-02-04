package nana

// Receiver is the interface for receiving messages.
type Receiver interface {
	Start()
	Stop()
	Close() error
}
