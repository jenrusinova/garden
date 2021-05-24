package driver

/// WireActor is an interface for a single actuator control
type WireActor interface {
	/// Get pin identifier
	GetID() string

	/// Check pin running
	IsRunning() bool

	/// Activate the device
	Start()

	/// Stop the device
	Stop()
}


/// WireDriver is an interface for hardware part
type WireDriver interface {
	/// Enumerate the pins
	AvailableActors() []WireActor
}
