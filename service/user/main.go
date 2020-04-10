package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.POST("/user/login", func(c *gin.Context) {
		name := c.GetHeader("name")
		if name == "" {
			name = "luoman"
		}

		c.JSON(200, gin.H{
			"boolean": 1,
			"message": "登录成功",
			"data":    map[string]interface{}{"uid": 100, "username": name},
		})
	})

	r.POST("/user/loginout", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"boolean": 1,
			"message": "退出成功",
			"data":    nil,
		})
	})

	r.Run(":8888")
}
