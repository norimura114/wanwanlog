package filter

import "wanwanlog/internal/logrecord"

type SearchQuery struct {
	Filter string
	Limit  int
	Offset int
	Order  SearchOrder
}

type SearchOrder int

const (
	OldestFirst SearchOrder = iota
	LatestWindow
)

type SearchResult struct {
	Records []logrecord.LogRecord
	Total   int
}
