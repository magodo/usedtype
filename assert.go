package main

func assert(b bool) {
	if !b {
		panic("assert fail")
	}
}
