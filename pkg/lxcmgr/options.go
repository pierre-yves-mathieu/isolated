package lxcmgr

// ProjectOption configures project creation
type ProjectOption func(*projectOpts)

type projectOpts struct {
	name     string
	ports    []int
	user     string
	password string
}

// WithProjectName sets the project name (defaults to directory name)
func WithProjectName(name string) ProjectOption {
	return func(o *projectOpts) {
		o.name = name
	}
}

// WithDefaultPorts sets the default ports for containers in the project
func WithDefaultPorts(ports ...int) ProjectOption {
	return func(o *projectOpts) {
		o.ports = ports
	}
}

// WithDefaultUser sets the default user for containers in the project
func WithDefaultUser(name, password string) ProjectOption {
	return func(o *projectOpts) {
		o.user = name
		o.password = password
	}
}

// CreateOption configures container creation
type CreateOption func(*createOpts)

type createOpts struct {
	ports    []int
	user     string
	password string
}

// WithPorts sets the ports for the container
func WithPorts(ports ...int) CreateOption {
	return func(o *createOpts) {
		o.ports = ports
	}
}

// WithUser sets the user for the container
func WithUser(name, password string) CreateOption {
	return func(o *createOpts) {
		o.user = name
		o.password = password
	}
}

// CloneOption configures container cloning
type CloneOption func(*cloneOpts)

type cloneOpts struct {
	fromSnapshot string
}

// FromSnapshot clones from a specific snapshot instead of current state
func FromSnapshot(name string) CloneOption {
	return func(o *cloneOpts) {
		o.fromSnapshot = name
	}
}

// MountOption configures mount operations
type MountOption func(*mountOpts)

type mountOpts struct {
	name           string
	readWrite      bool
	shift          bool
	allowRiskyPath bool
}

// WithMountName sets the device name for the mount
func WithMountName(name string) MountOption {
	return func(o *mountOpts) {
		o.name = name
	}
}

// WithReadWrite makes the mount read-write (default is read-only)
func WithReadWrite() MountOption {
	return func(o *mountOpts) {
		o.readWrite = true
	}
}

// WithShift enables UID/GID shifting
func WithShift() MountOption {
	return func(o *mountOpts) {
		o.shift = true
	}
}

// AllowRiskyPaths allows mounting paths that are flagged as risky
func AllowRiskyPaths() MountOption {
	return func(o *mountOpts) {
		o.allowRiskyPath = true
	}
}

// ShellOption configures shell access
type ShellOption func(*shellOpts)

type shellOpts struct {
	user string
}

// AsUser specifies which user to run the shell as
func AsUser(name string) ShellOption {
	return func(o *shellOpts) {
		o.user = name
	}
}

// CopyOption configures file copy operations
type CopyOption func(*copyOpts)

type copyOpts struct {
	autoCreateDir bool
}

// AutoCreateDir automatically creates the destination directory if it doesn't exist
func AutoCreateDir() CopyOption {
	return func(o *copyOpts) {
		o.autoCreateDir = true
	}
}
