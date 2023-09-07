package options

func Ptr[K any](m K) *K {
	return &m
}
