package src

func BytesToAsciiStr(bytes [3]byte) string {
	for i, value := range bytes {
		bytes[i] = value + 96
	}
	return string(bytes[:])
}
