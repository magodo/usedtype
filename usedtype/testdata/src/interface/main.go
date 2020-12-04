package main

import "sdk"

func main() {
	animal := sdk.OneAnimal{}
	animal.Name = "wangcai"
	animal.Animal = sdk.Dog{RunSpeed: 123}
	_ = animal
}
