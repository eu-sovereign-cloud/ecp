package handler

// DomainToSDK defines the interface for mapping domain objects to SDK objects
type DomainToSDK[D any, Out any] func(domain D) Out

// DomainToSDKList defines the interface for mapping a list of domain objects to an SDK object.
type DomainToSDKList[D any, Out any] func(domain []D, nextSkipToken *string) Out
