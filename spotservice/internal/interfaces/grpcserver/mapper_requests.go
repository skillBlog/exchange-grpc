package grpcserver

import (
	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/spotservice/internal/application"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

// Mapper преобразует protobuf-запросы в application input/output.
type Mapper struct{}

func (Mapper) ViewMarketsRequestToInput(req *spotv1.ViewMarketsRequest, userID string, roles []string) application.ViewMarketsInput {
	input := application.ViewMarketsInput{
		UserID:    userID,
		UserRoles: roles,
	}
	if pagination := req.GetPagination(); pagination != nil {
		input.PageToken = pagination.GetPageToken()
		input.PageSize = pagination.GetPageSize()
	}
	return input
}

func (Mapper) ViewMarketsOutputToResponse(out application.ViewMarketsOutput) *spotv1.ViewMarketsResponse {
	return &spotv1.ViewMarketsResponse{
		Markets: marketsToProto(out.Markets),
		PageInfo: &commonv1.CursorPageInfo{
			NextPageToken: out.NextPageToken,
			HasMore:       out.HasMore,
		},
	}
}

func (Mapper) GetMarketOutputToResponse(market domain.Market) *spotv1.GetMarketResponse {
	return &spotv1.GetMarketResponse{Market: marketToProto(market)}
}
