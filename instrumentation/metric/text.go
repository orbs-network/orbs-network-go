package metric

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
)

type Text struct {
	namedMetric
	value string
}

type textExport struct {
	Name  string
	Value string
}

func newText(name string, defaultValue ...string) *Text {
	value := ""

	if len(defaultValue) == 1 {
		value = defaultValue[0]
	}

	return &Text{
		namedMetric: namedMetric{name: name},
		value:       value,
	}
}
func (t *Text) Export() exportedMetric {
	return textExport{
		t.name,
		t.value,
	}
}

func (t *Text) Update(value string) {
	t.value = value
}

func (t *Text) String() string {
	return fmt.Sprintf("metric %s: %s\n", t.name, t.value)
}

func (t *Text) Value() string {
	return t.value
}

func (t textExport) LogRow() []*log.Field {
	return []*log.Field{
		log.String("metric", t.Name),
		log.String("metric-type", "text"),
		log.String("text", t.Value),
	}
}
