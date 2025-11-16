//go:build !linux

package mdns

import "fmt"

// DBusPublisher stub for non-Linux platforms
type DBusPublisher struct{}

// NewDBusPublisher returns an error on non-Linux platforms
func NewDBusPublisher() (*DBusPublisher, error) {
	return nil, fmt.Errorf("DBus-based Avahi publishing is only available on Linux")
}

// PublishService returns an error on non-Linux platforms
func (p *DBusPublisher) PublishService(service *Service) error {
	return fmt.Errorf("DBus-based Avahi publishing is only available on Linux")
}

// Stop does nothing on non-Linux platforms
func (p *DBusPublisher) Stop() error {
	return nil
}

// PublishHTTPDBus returns an error on non-Linux platforms
func PublishHTTPDBus(name string, port int, txtRecords ...string) (*DBusPublisher, error) {
	return nil, fmt.Errorf("DBus-based Avahi publishing is only available on Linux")
}

// IsAvahiDBusAvailable always returns false on non-Linux platforms
func IsAvahiDBusAvailable() bool {
	return false
}
