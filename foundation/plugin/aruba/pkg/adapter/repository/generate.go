package repository

//go:generate mockgen -package repository_test -destination=./zz_mock_k8s_cache_test.go sigs.k8s.io/controller-runtime/pkg/cache Cache,Informer
