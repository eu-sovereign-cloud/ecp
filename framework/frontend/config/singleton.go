package config

// TODO: review this implementation

import "sync"

type singleton struct {
	regionOnce sync.Once
	region     string
}

func (c *singleton) Region() string {
	return c.region
}

func (c *singleton) SetRegion(region string) {
	c.regionOnce.Do(func() {
		c.region = region
	})
}

var configInstance singleton

func Singleton() *singleton {
	return &configInstance
}
