# Seni

Seni is a simple fiber-like web framework written in Golang.

The Japanese word "Seni" is pronounced as "Sen-ee" in English.

## Quickstart

```golang
package main

import "github.com/demouth/seni"

func main() {
	app := seni.New()

	// GET / --> "Hello, World 👋!"
	app.Get("/", func(c *seni.Ctx) {
		c.Status(200)
		c.Write("Hello, World 👋!")
	})

	app.Listen(":3000")
}
```


## Examples

```golang
package main

import "github.com/demouth/seni"

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

	app.Listen(":3000")
}
```