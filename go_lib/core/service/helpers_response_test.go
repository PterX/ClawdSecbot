package service

import (
	"errors"
	"reflect"
	"testing"
)

func TestResponseHelpers(t *testing.T) {
	if got := successResult(); got["success"] != true || len(got) != 1 {
		t.Fatalf("Expected success result without payload, got %#v", got)
	}

	data := map[string]string{"id": "asset-1"}
	if got := successDataResult(data); got["success"] != true || !reflect.DeepEqual(got["data"], data) {
		t.Fatalf("Expected success result with payload, got %#v", got)
	}

	if got := errorResult(errors.New("failed")); got["success"] != false || got["error"] != "failed" {
		t.Fatalf("Expected error result from error, got %#v", got)
	}

	if got := errorMessageResult("bad input"); got["success"] != false || got["error"] != "bad input" {
		t.Fatalf("Expected error result from message, got %#v", got)
	}
}
