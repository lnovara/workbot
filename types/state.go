package types

// State holds the information for the current state the user is in.
type State uint

// Enumeration of possibile states.
const (
	Main = State(iota)
	Enter
	Exit
	SetAccessTime
	SetTimezone
	Settings
	UserSetupAccessTime
	UserSetupClientSecret
	UserSetupTimezone
)
