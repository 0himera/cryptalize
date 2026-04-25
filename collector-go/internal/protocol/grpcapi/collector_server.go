package grpcapi

import (
	"context"
	"fmt"

	"github.com/0himera/cryptalize/collector-go/internal/domains/market"
	collectorv1 "github.com/0himera/cryptalize/collector-go/internal/proto/collector/v1"
)

type CollectorServer struct {
	collectorv1.UnimplementedCollectorServiceServer
	collectors map[string]market.Collector
}

func NewCollectorServer(collectors map[string]market.Collector) *CollectorServer {
	return &CollectorServer{
		collectors: collectors,
	}
}

func (s *CollectorServer) Subscribe(ctx context.Context, req *collectorv1.SubscribeRequest) (*collectorv1.SubscribeResponse, error) {
	c, ok := s.collectors[req.Exchange]
	if !ok {
		return &collectorv1.SubscribeResponse{
			Success: false,
			Message: fmt.Sprintf("exchange %s not found", req.Exchange),
		}, nil
	}

	if err := c.Subscribe(ctx, req.Pair); err != nil {
		return &collectorv1.SubscribeResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &collectorv1.SubscribeResponse{
		Success: true,
		Message: "subscribed",
	}, nil
}

func (s *CollectorServer) Unsubscribe(ctx context.Context, req *collectorv1.UnsubscribeRequest) (*collectorv1.UnsubscribeResponse, error) {
	c, ok := s.collectors[req.Exchange]
	if !ok {
		return &collectorv1.UnsubscribeResponse{
			Success: false,
			Message: fmt.Sprintf("exchange %s not found", req.Exchange),
		}, nil
	}

	if err := c.Unsubscribe(ctx, req.Pair); err != nil {
		return &collectorv1.UnsubscribeResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &collectorv1.UnsubscribeResponse{
		Success: true,
		Message: "unsubscribed",
	}, nil
}
