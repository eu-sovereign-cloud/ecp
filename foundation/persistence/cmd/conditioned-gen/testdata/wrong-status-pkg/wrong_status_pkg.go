package wrongstatuspkg

type LocalStatus struct {
	Conditions []string
	State      string
}

// +ecp:conditioned
type WrongPkgType struct {
	Status *LocalStatus
}
