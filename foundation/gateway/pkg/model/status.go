package model

/*
 *
 * NOTE: This model is not complete nor ready for production usage.
 *       It's only for exploratory development.
 *
 */

type ResourceState string

const (
	StatePending   ResourceState = "pending"
	StateCreating  ResourceState = "creating"
	StateActive    ResourceState = "active"
	StateUpdating  ResourceState = "updating"
	StateDeleting  ResourceState = "deleting"
	StateSuspended ResourceState = "suspended"
	StateError     ResourceState = "error"
)

type ResourceStatus struct {
	State ResourceState
	Error error
}

func (s *ResourceStatus) SetPending() {
	s.Error = nil
	s.State = StatePending
}

func (s *ResourceStatus) SetCreating() {
	s.Error = nil
	s.State = StateCreating
}

func (s *ResourceStatus) SetAvtive() {
	s.Error = nil
	s.State = StateActive
}

func (s *ResourceStatus) SetUpdating() {
	s.Error = nil
	s.State = StateUpdating
}

func (s *ResourceStatus) SetDeleting() {
	s.Error = nil
	s.State = StateDeleting
}

func (s *ResourceStatus) SetSuspended() {
	s.Error = nil
	s.State = StateSuspended
}

func (s *ResourceStatus) SetError(err error) {
	s.Error = err
	s.State = StateError
}
