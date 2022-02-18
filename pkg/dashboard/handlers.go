package dashboard

import (
	"github.com/gin-gonic/gin"
	"log"
)

func Error(err error)  {

	if err !=nil {
		log.Println(err)
		panic(err)
	}
}

// error middle ware
func errorHandler() gin.HandlerFunc  {
	// 记录error 日志
	gin.ErrorLogger()
	return func(context *gin.Context) {
		defer func() {
			if e := recover(); e != nil {
				context.AbortWithStatusJSON(500, gin.H{"error": e.(error).Error()})
			}
		}()
		context.Next()
	}
}
