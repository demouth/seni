package main

import (
	"fmt"

	"github.com/demouth/seni"
)

func main() {
	app := seni.New()

	// GET / --> "Hello, World 👋!"
	app.Get("/", func(c *seni.Ctx) {
		c.Status(200)
		c.Write("Hello, World 👋!")
	})

	v1 := app.Group("/v1", func(c *seni.Ctx) {
		c.Write("Hello ")
		c.Next()
	})

	// GET /v1/hello/john/and/doe --> "Hello john and doe"
	v1.Get("/hello/:param/and/:param2", func(c *seni.Ctx) {
		c.Status(200)
		c.Write(c.Params("param") + " and " + c.Params("param2"))
	})

	// cURL:
	//    curl -X POST "localhost:3000/v1/post?id=123" -d "name=john"
	//
	// HTTP:
	//    POST /v1/post?id=123 HTTP/1.1
	//    Host: localhost:3000
	//    Content-Type: application/x-www-form-urlencoded
	//
	//    name=john
	//
	// RESPONSE:
	//    Hello id: 123; name: john;
	v1.Post("/post", func(c *seni.Ctx) {
		id := c.Query("id", "default")
		name := c.FormValue("name", "default")
		s := fmt.Sprintf("id: %s; name: %s;", id, name)
		c.Status(200)
		c.Write(s)
	})

	app.Listen(":3000")
}
