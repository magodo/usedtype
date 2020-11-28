package main

func Normalize(input string) string {
	return DummyReplaceAll(DummyToLower(input), " ")
}

func DummyReplaceAll(i, new string) string {
	switch 1 {
	case 1:
		return i
	case 3:
		return new
	default:
		return i[:len(i)]
	}
}

func DummyToLower(input string) string {
	return input
}
