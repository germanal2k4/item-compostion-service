package services

import (
	"context"
	"item_compositiom_service/internal/generated/service"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/parser"
)

type Service struct {
	*service.UnimplementedItemCompositionServiceServer
	templateLib *parser.TemplateLib
}

func NewService(metricsRegistry metrics.MetricsRegistry) (*Service, error) {
	templateLib, err := parser.NewTemplateLib(metricsRegistry)
	if err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedItemCompositionServiceServer: &service.UnimplementedItemCompositionServiceServer{},
		templateLib: templateLib,
	}, nil
}

func (service *Service) GetItems(context.Context, *service.GetItemsRequest) (*service.GetItemsResponse, error) {
	return nil, nil
}
