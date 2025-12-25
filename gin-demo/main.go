package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Users struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {

	// 1. 使用godotenv加载 .env 文件 (如果不配置gin的默认端口号是8080)
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	r := gin.Default()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, Gin!",
		})
	})

	api := r.Group("/api")
	{
		api.GET("/users", getUsers)
		api.POST("/users", createUser)
	}

	// Gin 并发处理入门
	r.GET("/async", func(c *gin.Context) {
		go func(ctx *gin.Context) {
			time.Sleep(2 * time.Second)
			fmt.Println("Async processing done")
		}(c.Copy()) // 必须使用 Copy() 避免 Context 并发问题

		c.JSON(200, gin.H{"status": "processing"})
	})

	r.Run(":" + port)
}

// 该function没有返回值，gin框架返回的json使用context进行包装返回
func createUser(context *gin.Context) {
	fmt.Println("this is createUser")
	newUser := Users{Name: "alice", Age: 12}
	//todo 进行创建用户的业务处理
	context.JSON(200, gin.H{"success": true, "data": newUser})
}

func getUsers(context *gin.Context) {
	newUser := Users{Name: "alice", Age: 12}
	context.JSON(200, gin.H{"success": true, "data": newUser})
	fmt.Println("this is getUsers")
}
