package jsonschema

import "github.com/fredbi/core/swag/pools"

var (
	poolOfOverlays        = pools.New[Overlay]()
	poolOfSchemas         = pools.New[Schema]()
	poolOfOverlayContexts = pools.New[overlayContext]()
	poolOfOverlayOptions  = pools.New[overlayOptions]
	poolOfOptions         = pools.New[options]
)
