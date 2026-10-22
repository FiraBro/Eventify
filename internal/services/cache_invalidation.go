package services

import (
	"context"

	"github.com/FiraBro/local-go/internal/cacheaside"
)

const (
	availabilityCacheNamespace = "local-go:availability:service:v1"
	servicesListCacheNamespace = "local-go:services:list:v1"
	settingsCacheNamespace     = "local-go:settings:singleton:v1"
)

// mutateAndInvalidate keeps cache invalidation after the mutation's successful
// return. Transactional repository methods return only after Commit succeeds.
func mutateAndInvalidate(ctx context.Context, store cacheaside.Store, namespaces []string, mutate func(context.Context) error) error {
	_, err := cacheaside.Mutate(ctx, store, namespaces, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, mutate(ctx)
	})
	return err
}
