package fgb

import "github.com/gogama/flatgeobuf/flatgeobuf/flat"

type RawFgb struct {
	header   *flat.Header
	Features []flat.Feature
}

type Fgb struct {
	Header   *flat.Header
	Features []Feature
}

type Feature struct {
	Raw        flat.Feature
	Properties map[string]any
}
