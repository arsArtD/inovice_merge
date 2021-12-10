package bootstrap

import (
	"github.com/arsArtD/inovice_merge/app/controller"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

func Excute() {
	baseDir, _ := os.Getwd()
	log.Println("bootstrap---->", baseDir)

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 20MB
	r.Static("/", "./public")

	var api controller.ApiController
	r.POST("/api/upload", api.Upload)
	r.POST("/api/merge_pdf", api.FormatPdf)

	r.Run(":8080")
}
