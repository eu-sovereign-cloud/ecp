package rest

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// getterFromRepo wraps a ReaderRepo as a Getter.
type getterFromRepo[D persistence.IdentifiableResource] struct {
	repo            persistence.ReaderRepo[D]
	newWithIdentity func(persistence.IdentifiableResource) D
}

// GetterFromRepo returns a Getter[D] backed by the given ReaderRepo.
// newWithIdentity builds an identity-populated zero value of D from the incoming IdentifiableResource,
// which is then passed to ReaderRepo.Load for the actual fetch.
func GetterFromRepo[D persistence.IdentifiableResource](
	repo persistence.ReaderRepo[D],
	newWithIdentity func(persistence.IdentifiableResource) D,
) Getter[D] {
	return &getterFromRepo[D]{repo: repo, newWithIdentity: newWithIdentity}
}

func (a *getterFromRepo[D]) Do(ctx context.Context, ir persistence.IdentifiableResource) (D, error) {
	d := a.newWithIdentity(ir)
	if err := a.repo.Load(ctx, &d); err != nil {
		var zero D
		return zero, err
	}
	return d, nil
}

// listerFromRepo wraps a ReaderRepo as a Lister.
type listerFromRepo[D persistence.IdentifiableResource] struct {
	repo persistence.ReaderRepo[D]
}

// ListerFromRepo returns a Lister[D] backed by the given ReaderRepo.
func ListerFromRepo[D persistence.IdentifiableResource](repo persistence.ReaderRepo[D]) Lister[D] {
	return &listerFromRepo[D]{repo: repo}
}

func (a *listerFromRepo[D]) Do(ctx context.Context, params resource.ListParams) ([]D, *string, error) {
	var items []D
	nextToken, err := a.repo.List(ctx, params, &items)
	if err != nil {
		return nil, nil, err
	}
	return items, nextToken, nil
}

// creatorFromRepo wraps a WriterRepo as a Creator.
type creatorFromRepo[D persistence.IdentifiableResource] struct {
	repo persistence.WriterRepo[D]
}

// CreatorFromRepo returns a Creator[D] backed by the given WriterRepo.
func CreatorFromRepo[D persistence.IdentifiableResource](repo persistence.WriterRepo[D]) Creator[D] {
	return &creatorFromRepo[D]{repo: repo}
}

func (a *creatorFromRepo[D]) Do(ctx context.Context, m D) (D, error) {
	p, err := a.repo.Create(ctx, m)
	if err != nil {
		var zero D
		return zero, err
	}
	return *p, nil
}

// updaterFromRepo wraps a WriterRepo as an Updater.
type updaterFromRepo[D persistence.IdentifiableResource] struct {
	repo persistence.WriterRepo[D]
}

// UpdaterFromRepo returns an Updater[D] backed by the given WriterRepo.
func UpdaterFromRepo[D persistence.IdentifiableResource](repo persistence.WriterRepo[D]) Updater[D] {
	return &updaterFromRepo[D]{repo: repo}
}

func (a *updaterFromRepo[D]) Do(ctx context.Context, m D) (D, error) {
	p, err := a.repo.Update(ctx, m)
	if err != nil {
		var zero D
		return zero, err
	}
	return *p, nil
}

// deleterFromRepo wraps a WriterRepo as a Deleter.
type deleterFromRepo[D persistence.IdentifiableResource] struct {
	repo            persistence.WriterRepo[D]
	newWithIdentity func(persistence.IdentifiableResource) D
}

// DeleterFromRepo returns a Deleter backed by the given WriterRepo.
// newWithIdentity builds an identity-populated zero value of D from the incoming IdentifiableResource.
func DeleterFromRepo[D persistence.IdentifiableResource](
	repo persistence.WriterRepo[D],
	newWithIdentity func(persistence.IdentifiableResource) D,
) Deleter {
	return &deleterFromRepo[D]{repo: repo, newWithIdentity: newWithIdentity}
}

func (a *deleterFromRepo[D]) Do(ctx context.Context, ir persistence.IdentifiableResource) error {
	d := a.newWithIdentity(ir)
	return a.repo.Delete(ctx, d)
}
