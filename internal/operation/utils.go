package operation

func Pointer[T any](t T) *T {
	return &t
}

func Dereference[T any](ptr *T) T {
	if ptr == nil {
		return *new(T)
	}

	return *ptr
}
