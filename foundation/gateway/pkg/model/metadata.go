package model

import "time"

// Metadata carries common resource identity and classification data used in domain models.
type Metadata struct {
	Name            string
	Namespace       string
	Labels          map[string]string
	ResourceVersion string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// CloneLabels returns a copy to avoid external mutation side-effects.
func (m Metadata) CloneLabels() map[string]string {
	if m.Labels == nil {
		return nil
	}
	cp := make(map[string]string, len(m.Labels))
	for k, v := range m.Labels {
		cp[k] = v
	}
	return cp
}
