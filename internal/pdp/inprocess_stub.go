package pdp

import (
	"context"
	"errors"
	"time"
)

type InprocessConfig struct{
	PolicyPath string
	DataPath string
	EntryPoint string
	Timeout time.Duration
}

type inprocessStub struct{}

func (s *inprocessStub) Evaluate(ctx context.Context, req Request) (Response, error) {
	return Response{}, errors.New("inprocess PDP not built; rebuild with -tags=opa_inprocess")
}

// NewInprocessClient returns a stub unless built with opa_inprocess tag.
func NewInprocessClient(cfg InprocessConfig) (Client, error) {
	return &inprocessStub{}, errors.New("inprocess PDP not built; rebuild with -tags=opa_inprocess")
}
