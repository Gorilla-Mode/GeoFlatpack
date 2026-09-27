package maplibre

var TestPaints = map[StyleGroup]Paint{
	{GeometryType: Polygon, Category: CategoryValue{kind: "string", value: "building"}}: {
		"fill-color":         "#2563eb",
		"fill-opacity":       0.55,
		"fill-outline-color": "#1e3a8a",
	},
	{GeometryType: Line, Category: CategoryValue{kind: "string", value: "power_line"}}: {
		"line-color": "#dc2626",
		"line-width": 5,
	},
	{GeometryType: Point, Category: CategoryValue{kind: "string", value: "street_light"}}: {
		"circle-color":        "#f59e0b",
		"circle-radius":       9,
		"circle-stroke-color": "#78350f",
		"circle-stroke-width": 2,
	},
}
