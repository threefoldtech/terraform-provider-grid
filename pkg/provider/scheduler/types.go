package scheduler

type Capacity struct {
	Memory uint64
	Sru    uint64
	Hru    uint64
}

type Request struct {
	Cap       Capacity
	Name      string
	Farm      string
	HasIPv4   bool
	HasDomain bool
	Certified bool

	farmID int
}
