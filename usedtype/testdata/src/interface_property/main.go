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

	_ = sdk.Fish{
		Name:      "nemo",
		SwimSpeed: 10,
	}

	animalFamily(dogFamily)
}

func animalFamily(family sdk.AnimalFamily) {}
