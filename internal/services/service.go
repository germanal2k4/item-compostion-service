package services

import (
	"context"

	servicepb "item_compositiom_service/internal/generated/service"
)

type Service struct {
	*servicepb.UnimplementedItemCompositionServiceServer
}

func NewService() *Service {
	return &Service{&servicepb.UnimplementedItemCompositionServiceServer{}}
}

func (service *Service) GetItems(context.Context, *servicepb.GetItemsRequest) (*servicepb.GetItemsResponse, error) {
	return nil, nil
}
