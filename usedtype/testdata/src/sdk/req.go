package sdk

type client struct{}

func BuildClient() client { return client{} }

func (c *client) CreateOrUpdate(req Req) {}
func (c *client) Delete()                {}
