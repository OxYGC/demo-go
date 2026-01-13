package main

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

var enforcer *casbin.Enforcer

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求中获取用户（这里简化为 header，实际可用 JWT）
		user := c.GetHeader("X-User")
		if user == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
			return
		}

		// 2. 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 3. 使用 Casbin 判断权限
		allowed, err := enforcer.Enforce(user, path, method)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth error"})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}

func main() {
	// 初始化 Casbin
	var err error
	enforcer, err = casbin.NewEnforcer("model.conf", "policy.csv")
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	// 应用权限中间件到需要保护的路由
	api := r.Group("/api")
	api.Use(authMiddleware())
	{
		api.GET("/user", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "user list"})
		})
		api.POST("/user", func(c *gin.Context) {
			c.JSON(200, gin.H{"msg": "user created"})
		})
		api.GET("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "my profile"})
		})
	}

	r.Run(":8080")
}
