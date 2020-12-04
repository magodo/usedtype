package main

import (
	"os"
	"sdk"
)

func main() {
	var zoo sdk.Zoo
	kind := os.Args[1]
	switch kind {
	case "dog":
		zoo.AnimalFamilies = []sdk.AnimalFamily{
			sdk.DogFamily{Animals: []sdk.Animal{sdk.Dog{
				Name: "wangcai",
			}}},
		}
	case "fish":
		zoo.AnimalFamilies = []sdk.AnimalFamily{
			sdk.FishFamily{Animals: []sdk.Animal{sdk.Fish{
				Name: "nemo",
			}}},
		}
	case "bird":
		zoo.AnimalFamilies = []sdk.AnimalFamily{
			sdk.BirdFamily{Animals: []sdk.Animal{sdk.Bird{
				Name: "jjzz",
			}}},
		}
	}
	_ = zoo
}
