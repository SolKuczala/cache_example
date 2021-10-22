package server

import (
	"cacheservice/pkg/model"
	"cacheservice/proto"
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	keyTest       = "keyTest"
	expectedValue = "example value"
	valueTest     = "random value"
	otherValue    = "other value"

	newValue = "value1"
)

type keyValueExample struct {
	key   string
	value string
}

// dummyServer returns the base server (cache) with its goroutine initialized for the command executions.
func dummyServer(t *testing.T, ctx context.Context) *Server {
	t.Helper()
	cache := model.NewCache(ctx, 0)
	return &Server{Cache: cache}
}

// setKeyValue returns a server with a default key-value set.
func setDummyCache(t *testing.T, srv *Server, kv keyValueExample) *Server {
	t.Helper()
	_, err := srv.Set(context.Background(), &proto.SetRequest{
		Key:   kv.key,
		Value: kv.value,
	})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}
	return srv
}

func TestSetAndGet(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	s := dummyServer(t, cancelCtx)

	payload := &proto.SetRequest{Key: keyTest, Value: expectedValue}
	t.Run("Set test key", func(t *testing.T) {
		resp, err := s.Set(context.Background(), payload)
		if err != nil {
			t.Fatalf("set failed: %v", err)
		}

		if resp.String() != "" {
			t.Fatalf("response mismatch, got: %v, want %s", resp.String(), "")
		}
	})

	t.Run("Get value for TestKey", func(t *testing.T) {
		resp, err := s.Get(context.Background(), &proto.GetRequest{Key: keyTest})
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if resp.GetValue() != expectedValue {
			t.Fatalf("response mismatch, got: %v, want %v", resp.GetValue(), expectedValue)
		}
	})
}

// TestGetWithNotExistantKey fails if the response value is not empty when quering for a
// non-existent key.
func TestGetWithNotExistantKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := dummyServer(t, ctx)

	resp, err := s.Get(context.Background(), &proto.GetRequest{Key: keyTest})
	if resp != nil {
		t.Fatalf("get fetched something: %v, want %v", err, nil)
	}
}

//
func TestCmpAndSetCmd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := dummyServer(t, ctx)

	// success path
	defaultValues := keyValueExample{
		key:   keyTest,
		value: valueTest,
	}
	s = setDummyCache(t, s, defaultValues)
	resp, err := s.CmpAndSet(context.Background(), &proto.CmpAndSetRequest{Key: defaultValues.key, OldValue: defaultValues.value, NewValue: newValue})
	if err != nil {
		t.Fatalf("CmpAndSet success failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Value was not changed.")
	}

	// check is the correct error
	resp, err = s.CmpAndSet(context.Background(), &proto.CmpAndSetRequest{Key: keyTest, OldValue: valueTest, NewValue: "2"})
	if err == nil {
		t.Fatalf("CmpAndSet failed to compare: %v", err)
	}
	if resp != nil {
		t.Fatal("Value was changed.")
	}

	grpcErr := status.Code(err)
	if grpcErr != codes.FailedPrecondition {
		t.Fatalf("got %s, want %s", grpcErr, codes.FailedPrecondition)
	}

	// check value remained unchanged
	t.Run("Get value for keyTest", func(t *testing.T) {
		resp, err := s.Get(context.Background(), &proto.GetRequest{Key: keyTest})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if resp.Value != newValue {
			t.Fatalf("Unexpected value: %v", resp)
		}
	})
}

func TestServer_Set_RequestWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := dummyServer(t, ctx)

	payload := &proto.SetRequest{Key: keyTest, Value: expectedValue}
	t.Run("Set test key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		// req canceled
		cancel()
		resp, err := s.Set(ctx, payload)
		if err == nil {
			t.Fatalf("set failed: %v", err)
		}

		if resp != nil {
			t.Fatalf("response mismatch, got: %v, want %s", resp.String(), "")
		}
	})
}

func TestServer_Set_CacheWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := dummyServer(t, ctx)
	select {
	case <-ctx.Done():
	default:
		t.Fatal("context was not canceled")
	} //do the same on the below tests

	payload := &proto.SetRequest{Key: keyTest, Value: expectedValue}
	t.Run("Set test key", func(t *testing.T) {
		// should fail
		resp, err := s.Set(context.Background(), payload)
		if err == nil {
			t.Fatalf("a value has been set: %s", resp.String())
		}
	})
}
