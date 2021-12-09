package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var root_path string

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 20MB
	r.Static("/", "./public")

	fmt.Println(getFileName(".pdf"))
	r.POST("/api/upload", func(c *gin.Context) {

		// Multipart form
		form, err := c.MultipartForm()
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
			return
		}
		files := form.File["file"]

		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			if ext != ".pdf" {
				c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", "file ext is not allowed!"))
				return
			}

			if err := c.SaveUploadedFile(file, getFileName(ext)); err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
				return
			}
		}

		c.String(http.StatusOK, fmt.Sprintf("Uploaded successfully %d files.", len(files)))
	})

	r.Run(":8080")
}

func getFileName(ext string) string {

	const UPLOAD_BASE_DIR = "public/upload/"
	// 时间格式必须按照这个来： RFC3339     = "2006-01-02T15:04:05Z07:00"
	datetime := time.Now().Format("20060102")

	dstPath := UPLOAD_BASE_DIR + datetime + "/"
	dstFile := dstPath + uuid.New().String() + ext

	_, err := os.Stat(dstPath)
	res := os.IsNotExist(err)
	if res == true {
		os.MkdirAll(dstPath, os.ModePerm)
	}

	return dstFile
}
