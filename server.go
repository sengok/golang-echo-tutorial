package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

type User struct {
	Name  string `json:"name" xml:"name" form:"name" query:"name"`
	Email string `json:"email" xml:"email" form:"email" query:"email"`
}

type Product struct {
	gorm.Model
	Code  string
	Price uint64
}

func main() {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/users/:id", getUser)
	e.GET("/show", show)

	e.POST("/users", createUser)
	e.POST("/save", save)
	e.POST("/multisave", multiSave)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	track := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			println("request to /users")
			return next(c)
		}
	}
	e.GET("/middle", func(c echo.Context) error {
		return c.String(http.StatusOK, "/middle")
	}, track)

	g := e.Group("/xxx")
	g.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "joe" && password == "secret" {
			return true, nil
		}
		return false, nil
	}))

	g.GET("/users", func(c echo.Context) error {
		return c.String(http.StatusOK, "/admin/users")
	}, track)

	e.GET("/products/migrate", migrate)
	e.GET("/products/:id", getProduct)
	e.POST("/products/register", registerProduct)
	e.POST("/products/update", updateProduct)
	e.POST("/products/delete", deleteProduct)

	e.GET("/redis/get/:key", redisGet)
	e.POST("/redis/set", redisSet)

	e.Static("/static", "static")
	e.Logger.Fatal(e.Start(":1323"))
}

func getUser(c echo.Context) error {
	id := c.Param("id")
	return c.String(http.StatusOK, id)
}

func createUser(c echo.Context) error {
	u := new(User)
	if err := c.Bind(u); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, u)
}

func show(c echo.Context) error {
	team := c.QueryParam("team")
	member := c.QueryParam("member")
	return c.String(http.StatusOK, "team:"+team+", member:"+member)
}

func save(c echo.Context) error {
	name := c.FormValue("name")
	email := c.FormValue("email")
	return c.String(http.StatusOK, "name:"+name+", email:"+email)
}

func multiSave(c echo.Context) error {
	name := c.FormValue("name")
	avatar, err := c.FormFile("avatar")
	if err != nil {
		return err
	}

	src, err := avatar.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {

		}
	}(src)

	dst, err := os.Create(avatar.Filename)
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {

		}
	}(dst)

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return c.HTML(http.StatusOK, "<b>Thank you! "+name+"</b>")
}

func getDb() *gorm.DB {
	dsn := "echo:echo@tcp(127.0.0.1:3306)/echo?parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("mysql connect error")
	}

	return db
}

func migrate(c echo.Context) error {
	db := getDb()
	err := db.AutoMigrate(&Product{})
	if err != nil {
		panic("mysql migrate error")
	}

	return c.String(http.StatusOK, "migrated")
}

func getProduct(c echo.Context) error {
	db := getDb()
	product := Product{}

	db.First(&product, c.Param("id"))
	fmt.Println(product)

	return c.String(http.StatusOK, "code: "+product.Code+", price: "+strconv.FormatUint(product.Price, 10))
}

func registerProduct(c echo.Context) error {
	db := getDb()
	code := c.FormValue("code")
	price, _ := strconv.ParseUint(c.FormValue("price"), 10, 64)

	db.Create(&Product{Code: code, Price: price})

	return c.String(http.StatusOK, "register product.")
}

func updateProduct(c echo.Context) error {
	db := getDb()
	price, _ := strconv.ParseUint(c.FormValue("price"), 10, 64)
	var product Product

	db.First(&product, c.FormValue("id"))
	db.Model(&product).Update("Price", price)

	return c.String(http.StatusOK, "updated.")
}

func deleteProduct(c echo.Context) error {
	db := getDb()
	var product Product

	db.Delete(&product, c.FormValue("id"))

	return c.String(http.StatusOK, "deleted.")
}

func getRedisClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	return rdb
}

func redisGet(c echo.Context) error {
	ctx := context.Background()
	key := c.Param("key")

	rdb := getRedisClient()

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return c.String(http.StatusOK, "key not found.")
		}
		panic(err)
	}

	return c.String(http.StatusOK, key+" : "+val)
}

func redisSet(c echo.Context) error {
	ctx := context.Background()
	key := c.FormValue("key")
	value := c.FormValue("value")

	rdb := getRedisClient()

	err := rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		panic(err)
	}

	return c.String(http.StatusOK, "set value")
}
