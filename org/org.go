package org

import "github.com/invopop/gobl/schema"

func init() {
	schema.Register(schema.GOBL.Add("org"),
		Address{},
		Coordinates{},
		Email{},
		Identity{},
		Image{},
		Inbox{},
		Item{},
		Name{},
		Party{},
		Person{},
		Registration{},
		Telephone{},
		Unit(""),
	)
}
