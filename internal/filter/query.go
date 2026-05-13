package filter

import "wanwanlog/internal/logrecord"

type SearchQuery struct {
	Filter string
	Limit  int
	Offset int
}

type SearchResult struct {
	Records []logrecord.LogRecord
	Total   int
}
