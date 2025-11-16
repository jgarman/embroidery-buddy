//go:build linux

package mdns

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// DBusPublisher publishes services using Avahi's DBus interface
// This is more reliable than using avahi-publish-service command
type DBusPublisher struct {
	conn         *dbus.Conn
	entryGroupPath dbus.ObjectPath
}

// NewDBusPublisher creates a new DBus-based Avahi publisher
func NewDBusPublisher() (*DBusPublisher, error) {
	// Connect to system bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	return &DBusPublisher{
		conn: conn,
	}, nil
}

// PublishService publishes a service using Avahi DBus API
func (p *DBusPublisher) PublishService(service *Service) error {
	// Get Avahi server object
	server := p.conn.Object("org.freedesktop.Avahi", "/")

	// Create entry group
	var entryGroupPath dbus.ObjectPath
	err := server.Call("org.freedesktop.Avahi.Server.EntryGroupNew", 0).Store(&entryGroupPath)
	if err != nil {
		return fmt.Errorf("failed to create entry group: %w", err)
	}

	p.entryGroupPath = entryGroupPath

	// Get entry group object
	entryGroup := p.conn.Object("org.freedesktop.Avahi", entryGroupPath)

	// Prepare TXT record data
	txtRecords := make([][]byte, len(service.TXTRecords))
	for i, txt := range service.TXTRecords {
		txtRecords[i] = []byte(txt)
	}

	// Add service to entry group
	// Parameters: interface, protocol, flags, name, type, domain, host, port, txt
	err = entryGroup.Call(
		"org.freedesktop.Avahi.EntryGroup.AddService",
		0,
		int32(-1),      // interface (-1 = all)
		int32(-1),      // protocol (-1 = unspecified, both IPv4/IPv6)
		uint32(0),      // flags
		service.Name,   // service name
		service.Type,   // service type
		service.Domain, // domain (empty = .local)
		service.Host,   // host (empty = use system hostname)
		uint16(service.Port),
		txtRecords,
	).Store()

	if err != nil {
		return fmt.Errorf("failed to add service: %w", err)
	}

	// Commit the entry group
	err = entryGroup.Call("org.freedesktop.Avahi.EntryGroup.Commit", 0).Store()
	if err != nil {
		return fmt.Errorf("failed to commit entry group: %w", err)
	}

	return nil
}

// Stop unpublishes the service
func (p *DBusPublisher) Stop() error {
	if p.entryGroupPath != "" {
		entryGroup := p.conn.Object("org.freedesktop.Avahi", p.entryGroupPath)
		err := entryGroup.Call("org.freedesktop.Avahi.EntryGroup.Reset", 0).Store()
		if err != nil {
			return fmt.Errorf("failed to reset entry group: %w", err)
		}

		err = entryGroup.Call("org.freedesktop.Avahi.EntryGroup.Free", 0).Store()
		if err != nil {
			return fmt.Errorf("failed to free entry group: %w", err)
		}
	}

	if p.conn != nil {
		p.conn.Close()
	}

	return nil
}

// PublishHTTPDBus publishes an HTTP service using DBus
func PublishHTTPDBus(name string, port int, txtRecords ...string) (*DBusPublisher, error) {
	publisher, err := NewDBusPublisher()
	if err != nil {
		return nil, err
	}

	service := &Service{
		Name:       name,
		Type:       "_http._tcp",
		Port:       port,
		Domain:     "",
		Host:       "",
		TXTRecords: txtRecords,
	}

	if err := publisher.PublishService(service); err != nil {
		publisher.Stop()
		return nil, err
	}

	return publisher, nil
}

// IsAvahiDBusAvailable checks if Avahi is available via DBus
func IsAvahiDBusAvailable() bool {
	conn, err := dbus.SystemBus()
	if err != nil {
		return false
	}
	defer conn.Close()

	// Try to call GetVersionString on Avahi server
	obj := conn.Object("org.freedesktop.Avahi", "/")
	var version string
	err = obj.Call("org.freedesktop.Avahi.Server.GetVersionString", 0).Store(&version)
	return err == nil
}
