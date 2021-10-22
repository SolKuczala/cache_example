package model_test

import (
	"cacheservice/pkg/model"
	"context"
	"testing"
)

func TestExecuteCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := model.NewCache(ctx, 0)

	set := func(m map[string]*model.Value) {
		m["test"] = &model.Value{Value: "test"}
	}

	if err := cache.Execute(ctx, set); err != nil {
		t.Errorf("%v", err)
	}

	get := func(m map[string]*model.Value) {
		value := &model.Value{Value: "test"}
		if m["test"] != value {
			t.Error("value was not set")
		}
	}
	if err := cache.Execute(ctx, get); err != nil {
		t.Errorf("%v", err)
	}
}
