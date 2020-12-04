package main

import "sdk"

func main() {
	var fish sdk.Animal
	fish = sdk.Fish{
		Name:      "nemo",
		SwimSpeed: 10,
	}

	var dog sdk.Animal
	dog = sdk.Dog{
		Name:     "wangcai",
		RunSpeed: 100,
	}

	_ = dog
	_ = fish
}
