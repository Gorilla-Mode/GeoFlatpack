package validate

import "fmt"

type StyleFormat string

const (
	FormatMapLibre StyleFormat = "maplibre"
	FormatSLD      StyleFormat = "sld"
)

func Format(format StyleFormat) error {
	switch format {
	case FormatMapLibre, FormatSLD:
		return nil
	default:
		return fmt.Errorf("invalid style format: %s", format)
	}
}
