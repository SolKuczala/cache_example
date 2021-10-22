// Package model contains the struct for the cache its method and storage constructor function.
package model

import (
	"context"
	"errors"
	"fmt"
)

// Cache contains the key value store and the channel that would handle the commmands function.
type Cache struct {
	cache map[string]*Value
	// read only channel
	commandCh chan Job
	closedCh  chan struct{}
}

type Value struct {
	Value string
}

// Job to be executed by the cache.
type Job func(map[string]*Value)

// Execute will send the job to the command channel in the cache to be executed.
func (c *Cache) Execute(cmdCtx context.Context, j Job) error {
	select {
	case <-cmdCtx.Done():
		return fmt.Errorf("Cache.Execute: %w", cmdCtx.Err())
	case <-c.closedCh:
		return errors.New("cache stopped working")
	default:
	}

	select {
	case <-cmdCtx.Done():
		return fmt.Errorf("Cache.Execute: %w", cmdCtx.Err())
	case c.commandCh <- j:
		return nil
	case <-c.closedCh:
		return errors.New("cache stopped working")
	}
}

// executeCommands will receive the function passed to the cache and execute it,
// until ctx done is received.
func (c *Cache) executeCommands(mainCtx context.Context) {
	if err := mainCtx.Err(); err != nil {
		close(c.closedCh)
		return
	}
	for {
		select {
		case f := <-c.commandCh:
			f(c.cache)
		case <-mainCtx.Done():
			close(c.closedCh)
			return
		}
	}
}

// NewCache cache constructor that returns an initialized cache struct.
func NewCache(ctx context.Context, buf int) *Cache {
	cache := Cache{
		cache:     make(map[string]*Value),
		commandCh: make(chan Job, buf),
		closedCh:  make(chan struct{}),
	}
	go cache.executeCommands(ctx)
	return &cache
}
