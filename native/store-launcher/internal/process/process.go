package process

// Info describes one process at a point in time. CreationTime is Unix time in
// nanoseconds and combines with PID to form a stable process identity.
type Info struct {
	PID          uint32
	ParentPID    uint32
	Name         string
	Executable   string
	CreationTime uint64
}

type Identity struct {
	PID          uint32
	CreationTime uint64
}

func (p Info) Identity() Identity {
	return Identity{PID: p.PID, CreationTime: p.CreationTime}
}

type Source interface {
	Snapshot() ([]Info, error)
}
