package shell

// SessionState represents the state of a session
type SessionState string

const (
	SessionRunning SessionState = "running"
	SessionClosed  SessionState = "closed"
)
