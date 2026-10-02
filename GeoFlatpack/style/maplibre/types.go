package maplibre

type Paint map[string]any
type GeometryType string

const (
	Point   GeometryType = "Point"
	Line    GeometryType = "LineString"
	Polygon GeometryType = "Polygon"
)

type RenderType string

const (
	RenderBackground RenderType = "background"
	RenderFill       RenderType = "fill"
	RenderLine       RenderType = "line"
	RenderCircle     RenderType = "circle"
	RenderSymbol     RenderType = "symbol"
)

type categoryKind string

const (
	categoryAll     categoryKind = ""
	categoryMissing categoryKind = "missing"
	categoryString  categoryKind = "string"
	categoryBoolean categoryKind = "boolean"
	categoryNumber  categoryKind = "number"
)

type PropertyType string

const (
	ColorType      PropertyType = "color"
	StringType     PropertyType = "string"
	EnumType       PropertyType = "enum"
	BooleanType    PropertyType = "boolean"
	NumberType     PropertyType = "number"
	ArrayType      PropertyType = "array"
	ReferenceType  PropertyType = "reference"
	TransitionType PropertyType = "transition"
)

type StyleSection string

const (
	PaintSection  StyleSection = "paint"
	LayoutSection StyleSection = "layout"
)

// The first render type is the default for an unconfigured geometry group.
var RenderTypes = map[GeometryType][]RenderType{
	Polygon: {RenderFill, RenderLine, RenderCircle},
	Line:    {RenderLine},
	Point:   {RenderCircle},
}

type RenderLayerStyle struct {
	Type     RenderType
	Paint    Paint
	Layout   map[string]any
	IconName string
}

type StyleGroup struct {
	GeometryType GeometryType
	Category     CategoryValue
}

type StyleLayer struct {
	ID     string         `json:"id"`
	Type   RenderType     `json:"type"`
	Source string         `json:"source,omitempty"`
	Filter []any          `json:"filter,omitempty"`
	Paint  Paint          `json:"paint,omitempty"`
	Layout map[string]any `json:"layout,omitempty"`
}

type StyleHeader struct {
	Metadata map[string]any            `json:"metadata,omitempty"`
	Version  int                       `json:"version"`
	Name     string                    `json:"name"`
	Center   []float64                 `json:"center,omitempty"`
	Zoom     float64                   `json:"zoom,omitempty"`
	Sources  map[string]map[string]any `json:"sources"`
	Layers   []StyleLayer              `json:"layers"`
}
