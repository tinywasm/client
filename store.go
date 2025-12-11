package client

// Store defines the interface for a key-value storage system
// used to persist the compiler state (e.g. current mode).
type Store interface {
	Get(key string) (string, error)
	Set(key, value string) error
}
