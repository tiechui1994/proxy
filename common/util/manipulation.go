package util

func EmptyOr(a string, v ...string) string {
	if a != "" {
		return a
	}
	return v[0]
}
