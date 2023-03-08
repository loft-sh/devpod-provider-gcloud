package ptr

func Ptr[K any](m K) *K {
	return &m
}
