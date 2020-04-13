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

        c.JSON(200, gin.H{
            "boolean": 1,
            "data": []order{
                {
                    Id:      110,
                    OrderNo: "ewrewrewrwerwe",
                    Amount:  1110.05,
                },
                {
                    Id:      111,
                    OrderNo: "bcvbvcbcvbcvbf",
                    Amount:  2110.05,
                },
                {
                    Id:      112,
                    OrderNo: "gfdhgfhfghgfhs",
                    Amount:  3110.05,
                },
            },
        })
    })

    r.Run(":8889")
}
