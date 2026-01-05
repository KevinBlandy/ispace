package util

func Require[T any](fn func() (T, error)) T {
	ret, err := fn()
	if err != nil {
		panic(err)
	}
	return ret
}

func If[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
