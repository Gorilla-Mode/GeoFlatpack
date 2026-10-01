package maplibre

type Paint map[string]any
type GeometryType string

const (
	Point   GeometryType = "Point"
	Line    GeometryType = "LineString"
	Polygon GeometryType = "Polygon"
)

// The first render type is the default for an unconfigured geometry group.
var RenderTypes = map[GeometryType][]string{
	Polygon: {"fill", "line", "circle"},
	Line:    {"line"},
	Point:   {"circle"},
}

type RenderLayerStyle struct {
	Type  string
	Paint Paint
}

type StyleGroup struct {
	GeometryType GeometryType
	Category     CategoryValue
}

type StyleLayer struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Source string         `json:"source,omitempty"`
	Filter []any          `json:"filter,omitempty"`
	Paint  Paint          `json:"paint,omitempty"`
	Layout map[string]any `json:"layout,omitempty"`
}

type StyleHeader struct {
	Version int                       `json:"version"`
	Name    string                    `json:"name"`
	Center  []float64                 `json:"center,omitempty"`
	Zoom    float64                   `json:"zoom,omitempty"`
	Sources map[string]map[string]any `json:"sources"`
	Layers  []StyleLayer              `json:"layers"`
}
