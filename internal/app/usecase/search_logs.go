package usecase

import (
	"context"
	"fmt"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/search"
)

type SearchLogsInput struct {
	Query  string
	Limit  int
	Offset int
	Order  filter.SearchOrder
}

type SearchLogsOutput struct {
	Result filter.SearchResult
}

type SearchLogsUseCase struct {
	Store search.Store
}

func NewSearchLogsUseCase(store search.Store) *SearchLogsUseCase {
	return &SearchLogsUseCase{Store: store}
}

func (u *SearchLogsUseCase) Search(ctx context.Context, query filter.SearchQuery) (filter.SearchResult, error) {
	result, err := u.Execute(ctx, SearchLogsInput{
		Query: query.Filter, Limit: query.Limit, Offset: query.Offset, Order: query.Order,
	})
	return result.Result, err
}

func (u *SearchLogsUseCase) Execute(ctx context.Context, input SearchLogsInput) (SearchLogsOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 100
	}
	result, err := u.Store.Search(ctx, filter.SearchQuery{
		Filter: input.Query, Limit: input.Limit, Offset: input.Offset, Order: input.Order,
	})
	if err != nil {
		return SearchLogsOutput{}, fmt.Errorf("search logs: %w", err)
	}
	return SearchLogsOutput{Result: result}, nil
}
