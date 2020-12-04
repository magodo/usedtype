package sdk

type clientA struct{}

func (c clientA) Create(a ModelA) {}
func (c clientA) Delete()         {}

type clientAnimal struct{}

func (c clientAnimal) Create(a Animal) {}
func (c clientAnimal) Delete()         {}

type clientAnimalFamily struct{}

func (c clientAnimalFamily) Create(f AnimalFamily) {}
func (c clientAnimalFamily) Delete()               {}

type clientZoo struct {}
func (c clientZoo) Create(z Zoo) {}
func (c clientZoo) Delete() {}
