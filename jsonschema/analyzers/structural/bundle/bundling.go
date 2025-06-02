package bundle

type SchemaBundlingStragegy uint8

const (
	Flat SchemaBundlingStragegy = iota
	Hierarchical
)

type SchemaBundlingAggressiveness uint8

const (
	Lazy SchemaBundlingAggressiveness = iota
	Eager
)
