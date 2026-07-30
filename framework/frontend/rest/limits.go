package rest

// MaxRequestBodyBytes caps the request body size for body-bearing methods.
// Concurrent unbounded reads can OOM gateway pods under multi-tenant load.
const MaxRequestBodyBytes = 1 << 20 // 1 MiB
