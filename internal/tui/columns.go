package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	TimestampColumn = "timestamp"
	StatusColumn    = "status"
	ServiceColumn   = "service"
	MessageColumn   = "message"
)

var defaultColumnNames = []string{
	TimestampColumn,
	StatusColumn,
	ServiceColumn,
	MessageColumn,
}

type Column struct {
	Name    string
	Visible bool
}

type AttributeNameCollector struct {
	scalarNames map[string]struct{}
}

func NewAttributeNameCollector() *AttributeNameCollector {
	return &AttributeNameCollector{
		scalarNames: make(map[string]struct{}),
	}
}

func (c *AttributeNameCollector) Add(attributes map[string]any) {
	for key, value := range attributes {
		c.collect("@"+key, value)
	}
}

func (c *AttributeNameCollector) Names() []string {
	names := make([]string, 0, len(c.scalarNames))
	for name := range c.scalarNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *AttributeNameCollector) collect(name string, value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			c.collect(name+"."+key, child)
		}
	case []any:
		return
	default:
		c.scalarNames[name] = struct{}{}
	}
}

func NewColumns(attributeNames, selectedNames []string) []Column {
	available := append([]string(nil), defaultColumnNames...)

	attributes := uniqueSortedAttributeNames(attributeNames)
	available = append(available, attributes...)

	if len(selectedNames) == 0 {
		selectedNames = defaultColumnNames
	}

	seen := make(map[string]bool, len(available)+len(selectedNames))
	columns := make([]Column, 0, len(available)+len(selectedNames))
	for _, name := range selectedNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		columns = append(columns, Column{Name: name, Visible: true})
	}
	for _, name := range available {
		if seen[name] {
			continue
		}
		seen[name] = true
		columns = append(columns, Column{Name: name})
	}

	return columns
}

func uniqueSortedAttributeNames(names []string) []string {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, "@") {
			set[name] = struct{}{}
		}
	}

	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func cloneColumns(columns []Column) []Column {
	return append([]Column(nil), columns...)
}

func visibleColumns(columns []Column) []Column {
	visible := make([]Column, 0, len(columns))
	for _, column := range columns {
		if column.Visible {
			visible = append(visible, column)
		}
	}
	return visible
}

func formatColumnValue(column string, recordAttributes map[string]any, timestamp time.Time, status, service, message string) string {
	switch column {
	case TimestampColumn:
		if timestamp.IsZero() {
			return ""
		}
		return timestamp.Format("15:04:05")
	case StatusColumn:
		return singleLine(status)
	case ServiceColumn:
		return singleLine(service)
	case MessageColumn:
		return singleLine(message)
	}

	value, ok := attributeValue(recordAttributes, strings.TrimPrefix(column, "@"))
	if !ok || value == nil {
		return ""
	}

	switch value := value.(type) {
	case []any:
		return "[array]"
	case map[string]any:
		return "{object}"
	case string:
		return singleLine(value)
	case bool:
		return strconv.FormatBool(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	default:
		return singleLine(fmt.Sprint(value))
	}
}

func attributeValue(attributes map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}

	var current any = attributes
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
