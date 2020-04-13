package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    r.GET("/order/list", func(c *gin.Context) {
        type order struct {
            Id      int64   `json:"id"`
            OrderNo string  `json:"order_no"`
            Amount  float64 `json:"amount"`
        }

        uid := c.GetHeader("uid")
        dataList := []order{
            {
                Id:      100,
                OrderNo: uid + ":order100",
                Amount:  1110.05,
            },
            {
                Id:      110,
                OrderNo: uid + ":order110",
                Amount:  2110.05,
            },
        }

        c.JSON(200, gin.H{
            "boolean": 1,
            "message": "OK",
            "data":    dataList,
        })
    })

    r.Run(":8889")
}
