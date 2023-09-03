package seni

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoot(t *testing.T) {
	app := New()

	app.Get("/", func(c *Ctx) {
		c.SendString("Hello, World 👋!")
	})

	res, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "Hello, World 👋!", bodyToString(res))
}

func TestBody(t *testing.T) {
	app := New()
	app.Get("/a/b/c", func(c *Ctx) {
		c.SendString("Hello, World 👋!")
	})

	res, err := app.Test(httptest.NewRequest("GET", "/a/b/c", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "Hello, World 👋!", bodyToString(res))
}

func TestNotFound(t *testing.T) {
	app := New()
	app.Get("/", func(c *Ctx) {
		c.SendString("Hello, World 👋!")
	})

	res, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 404, res.StatusCode)

	res, err = app.Test(httptest.NewRequest("GET", "/unknown", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 404, res.StatusCode)
}

func TestParams(t *testing.T) {
	app := New()
	app.Get("/test", func(c *Ctx) {
		c.Status(400).SendString("Should move on")
	})
	app.Get("/test/:param", func(c *Ctx) {
		c.Status(400).SendString("Should move on")
	})
	app.Get("/test/:param/test", func(c *Ctx) {
		c.Status(400).SendString("Should move on")
	})
	app.Get("/test/:param/test/:param2", func(c *Ctx) {
		c.Status(200).SendString("Good job " + c.Params("param") + " and " + c.Params("param2"))
	})

	res, err := app.Test(httptest.NewRequest("GET", "/test/john/test/doe", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "Good job john and doe", bodyToString(res))
}

func TestQuery(t *testing.T) {
	app := New()
	app.Get("/test", func(c *Ctx) {
		c.Status(200).SendString("Hello " + c.Query("name", "default"))
	})
	res, err := app.Test(httptest.NewRequest("GET", "/test?name=john", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "Hello john", bodyToString(res))
}

func TestHandlers(t *testing.T) {
	app := New()
	app.Use(func(c *Ctx) {
		c.Write("1")
		c.Next()
	})
	app.Get(
		"/test",
		func(c *Ctx) {
			c.Write("2")
			c.Next()
		},
		func(c *Ctx) {
			c.Write("3")
			c.Next()
		},
		func(c *Ctx) {
			c.Status(200)
			c.Write("4")
		},
	)

	res, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "1234", bodyToString(res))
}

func TestGroup(t *testing.T) {
	app := New()
	group := app.Group("/test", func(c *Ctx) {
		c.Write("1")
		c.Next()
	})
	group = group.Group("/v1", func(c *Ctx) {
		c.Write("2")
		c.Next()
	})
	group.Get("/", func(c *Ctx) {
		c.Write("3")
		c.Status(200)
	})

	res, err := app.Test(httptest.NewRequest("GET", "/test/v1/", nil))
	if err != nil {
		assert.FailNow(t, "serving error", err)
	}
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "123", bodyToString(res))
}

func bodyToString(res *http.Response) string {
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	return string(body)
}
