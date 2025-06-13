package typeutils

import "maps"

// MergeMaps merges maps into the target.
func MergeMaps[M ~map[K]V, K comparable, V any](target M, merged ...M) M {
	for _, m := range merged {
		maps.Copy(target, m)
	}

	return target
}
