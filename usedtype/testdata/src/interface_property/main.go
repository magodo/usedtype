package main

import "sdk"

func main() {
	dogFamily := sdk.DogFamily{
		Animals: []sdk.Animal{
			sdk.Dog{
				Name:     "wangcai",
				RunSpeed: 100,
			},
		},
	}

	fish := sdk.Fish{
		Name:      "nemo",
		SwimSpeed: 10,
	}
	_ = fish

	_ = dogFamily
}
