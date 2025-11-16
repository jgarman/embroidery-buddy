package mdns

import (
	"fmt"
	"os/exec"
	"strings"
)

// Service represents an Avahi service registration
type Service struct {
	Name     string // Service name (e.g., "Embroidery Buddy")
	Type     string // Service type (e.g., "_http._tcp")
	Port     int    // Port number
	Domain   string // Domain (usually "local")
	Host     string // Hostname (optional, uses system hostname if empty)
	TXTRecords []string // TXT records (key=value pairs)
}

// Publisher handles Avahi service publication
type Publisher struct {
	cmd *exec.Cmd
}

// NewPublisher creates a new Avahi service publisher
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Publish registers a service with Avahi using avahi-publish-service
func (p *Publisher) Publish(service *Service) error {
	// Check if avahi-publish-service is available
	if _, err := exec.LookPath("avahi-publish-service"); err != nil {
		return fmt.Errorf("avahi-publish-service not found: %w (install avahi-utils)", err)
	}

	// Build command arguments
	args := []string{
		service.Name,
		service.Type,
		fmt.Sprintf("%d", service.Port),
	}

	// Add TXT records if any
	if len(service.TXTRecords) > 0 {
		args = append(args, service.TXTRecords...)
	}

	// Create the command
	p.cmd = exec.Command("avahi-publish-service", args...)

	// Start the service publication
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start avahi-publish-service: %w", err)
	}

	return nil
}

// Stop stops the service publication
func (p *Publisher) Stop() error {
	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		// Wait for the process to exit
		_ = p.cmd.Wait()
	}
	return nil
}

// PublishHTTP is a convenience function to publish an HTTP service
func PublishHTTP(name string, port int, txtRecords ...string) (*Publisher, error) {
	service := &Service{
		Name:       name,
		Type:       "_http._tcp",
		Port:       port,
		TXTRecords: txtRecords,
	}

	publisher := NewPublisher()
	if err := publisher.Publish(service); err != nil {
		return nil, err
	}

	return publisher, nil
}

// PublishHTTPS is a convenience function to publish an HTTPS service
func PublishHTTPS(name string, port int, txtRecords ...string) (*Publisher, error) {
	service := &Service{
		Name:       name,
		Type:       "_https._tcp",
		Port:       port,
		TXTRecords: txtRecords,
	}

	publisher := NewPublisher()
	if err := publisher.Publish(service); err != nil {
		return nil, err
	}

	return publisher, nil
}

// IsAvahiAvailable checks if Avahi is available on the system
func IsAvahiAvailable() bool {
	_, err := exec.LookPath("avahi-publish-service")
	return err == nil
}

// GetServiceURL constructs a typical service URL
func GetServiceURL(service *Service) string {
	protocol := "http"
	if strings.Contains(service.Type, "https") {
		protocol = "https"
	}

	hostname := service.Host
	if hostname == "" {
		hostname = "localhost"
	}

	// For .local domain, construct mDNS hostname
	if service.Domain == "local" || service.Domain == "" {
		// Try to get system hostname
		if out, err := exec.Command("hostname").Output(); err == nil {
			hostname = strings.TrimSpace(string(out))
		}
		return fmt.Sprintf("%s://%s.local:%d", protocol, hostname, service.Port)
	}

	return fmt.Sprintf("%s://%s:%d", protocol, hostname, service.Port)
}
