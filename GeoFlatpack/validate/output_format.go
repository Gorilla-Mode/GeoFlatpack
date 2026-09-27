package validate

import "fmt"

type StyleFormat string

const (
	FormatMapLibre StyleFormat = "maplibre"
	FormatSLD      StyleFormat = "sld"
)

func Format(format StyleFormat) error {
	switch format {
	case FormatMapLibre:
		return nil
	case FormatSLD:
		return fmt.Errorf("format not supported yet: %s", format)
	default:
		return fmt.Errorf("invalid style format: %s", format)
	}
}
