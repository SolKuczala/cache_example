// Package server implements methods for grpc server
package server

import (
	"cacheservice/pkg/model"
	"cacheservice/proto"
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	Cache *model.Cache
}

// this line will not compile if this struct is not implementing the interface.
var _ proto.CacheServiceServer = (*Server)(nil)

func (s *Server) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetReply, error) {
	var (
		modelValue *model.Value
		found      bool
	)
	doneCh := make(chan struct{})
	// prepare job
	job := func(m map[string]*model.Value) {
		if err := ctx.Err(); err != nil {
			return
		}
		defer close(doneCh)
		modelValue, found = m[req.GetKey()]
		if found {
			log.Printf("value found:%s\n", modelValue.Value)
		} else {
			log.Printf("Get: value not found: %v\n", modelValue)
		}
	}
	// call method to send job to the channel
	if err := s.Cache.Execute(ctx, job); err != nil {
		return nil, fmt.Errorf("Get: %w", err)
	}

	select {
	case <-ctx.Done():
		log.Printf("Get command aborted")
		return nil, fmt.Errorf("select get: %w", ctx.Err())
	case <-doneCh:
	}
	if !found {
		return nil, status.Error(codes.NotFound, "get: value for key not set")
	}

	return &proto.GetReply{Value: modelValue.Value}, nil
}

func (s *Server) Set(cmdCtx context.Context, req *proto.SetRequest) (*proto.SetReply, error) {
	doneCh := make(chan error, 1)
	key := req.GetKey()
	value := req.GetValue()
	// prepare job
	job := func(cache map[string]*model.Value) {
		if err := cmdCtx.Err(); err != nil {
			return
		}
		defer close(doneCh)
		// check if value exists
		_, exists := cache[key]
		if !exists {
			// if the key doesn't exists, set the entire struct.
			cache[req.GetKey()] = &model.Value{
				Value: value,
			}
		} else {
			// if exists, return error.
			doneCh <- status.Error(codes.PermissionDenied, "value already set, use CmpAndSet")
			return
		}
	}
	if err := s.Cache.Execute(cmdCtx, job); err != nil {
		return nil, fmt.Errorf("Set: %w", err)
	}
	if err := cmdCtx.Err(); err != nil {
		log.Printf("Set command aborted")
		return nil, cmdCtx.Err()
	}

	select {
	case <-cmdCtx.Done():
		log.Printf("Get command aborted")
		return nil, cmdCtx.Err()
	case statusErr := <-doneCh:
		if statusErr != nil {
			return nil, statusErr
		}
		log.Printf("value set, key %s : value %s", key, value)
		return &proto.SetReply{}, nil
	}
}

func (s *Server) CmpAndSet(ctx context.Context, req *proto.CmpAndSetRequest) (*proto.CmpAndSetReply, error) {
	key := req.GetKey()
	oldValue := req.OldValue
	newValue := req.NewValue

	doneCh := make(chan error, 1)
	job := func(cache map[string]*model.Value) {
		if err := ctx.Err(); err != nil {
			return
		}
		defer close(doneCh)

		value, exists := cache[key]
		if !exists {
			doneCh <- status.Error(codes.NotFound, "value for key not set")
			return
		}
		if exists && oldValue == value.Value {
			if req.GetNewValue() != value.Value {
				cache[key].Value = newValue
				log.Printf("old value(%s) matched, set: key %s, new value: %s\n", oldValue, key, newValue)
			} else {
				log.Printf("value was the same\n")
			}
			doneCh <- nil
		} else {
			log.Printf("map not set, value does not exist or value did not match: %s\n", value.Value)
			doneCh <- status.Error(codes.FailedPrecondition, "old value does not match, try again")
			return
		}
	}

	if err := s.Cache.Execute(ctx, job); err != nil {
		return nil, fmt.Errorf("CmpAndSet: %w", err)
	}

	select {
	case <-ctx.Done():
		log.Printf("Get command aborted")
		return nil, ctx.Err()
	case statusErr := <-doneCh:
		if statusErr != nil {
			return nil, statusErr
		}
		log.Printf("value set for key %s, old value %s : value %s", key, oldValue, newValue)
		return &proto.CmpAndSetReply{
			Changed: true,
		}, nil
	}
}
