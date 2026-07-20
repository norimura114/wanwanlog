package usecase

import "sort"

type AttributeNameCollector struct {
	names map[string]struct{}
}

func NewAttributeNameCollector() *AttributeNameCollector {
	return &AttributeNameCollector{names: make(map[string]struct{})}
}

func (c *AttributeNameCollector) Add(attributes map[string]any) {
	for key, value := range attributes {
		c.collect("@"+key, value)
	}
}

func (c *AttributeNameCollector) collect(name string, value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			c.collect(name+"."+key, child)
		}
	case []any:
	default:
		c.names[name] = struct{}{}
	}
}

func (c *AttributeNameCollector) Names() []string {
	result := make([]string, 0, len(c.names))
	for name := range c.names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
