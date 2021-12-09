package controller

import (
	"fmt"
	"github.com/arsArtD/inovice_merge/app/library"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type ApiController struct {
}

func (controller *ApiController) Upload(contenxt *gin.Context) {

	// Multipart form
	form, err := contenxt.MultipartForm()
	if err != nil {
		contenxt.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	files := form.File["file"]

	var pdfFiles = make([]string, 0)
	for _, file := range files {
		ext := filepath.Ext(file.Filename)
		if ext != ".pdf" {
			contenxt.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", "file ext is not allowed!"))
			return
		}

		var sourceUploadFileName = controller.getUploadFileName(ext, true)
		if err := contenxt.SaveUploadedFile(file, sourceUploadFileName); err != nil {
			contenxt.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}
		pdfFiles = append(pdfFiles, sourceUploadFileName)
	}

	var dstMergePdfFileName = controller.getUploadFileName(".pdf", false)
	pdfcpuPath, megeDstPath := controller.getUploadConfig()

	// 合并pdf
	var mergePdfPath = filepath.Join(megeDstPath, dstMergePdfFileName)
	var cmdArgs = []string{"merge", mergePdfPath}
	cmdArgs = append(cmdArgs, pdfFiles...)
	mergeRes, excuteRes := library.CheckCmdRes(pdfcpuPath, cmdArgs)
	if !mergeRes {
		contenxt.String(http.StatusBadRequest, fmt.Sprintf("merg pdf error"))
		return
	}
	log.Println("ApiController:upload:合并pdf----->\n", excuteRes)

	// pdf内容重新排版
	var resPdfFilePath = controller.getUploadFileName(".pdf", false)
	resPdfFilePath = filepath.Join(megeDstPath, resPdfFilePath)
	var cmdArgForReformat = []string{"grid", "--", "bo:off", resPdfFilePath, "4", mergePdfPath}
	formatRes, excuteRes := library.CheckCmdRes(pdfcpuPath, cmdArgForReformat)
	if !formatRes {
		contenxt.String(http.StatusBadRequest, fmt.Sprintf("format pdf error"))
		return
	}
	log.Println("ApiController:upload:pdf内容重构----->\n", excuteRes)

	contenxt.String(http.StatusOK, fmt.Sprintf("Uploaded successfully %d files.", len(files)))
}

func (controller *ApiController) getUploadConfig() (string, string) {
	var goArch = runtime.GOARCH //amd64 arm64
	var goos = runtime.GOOS     //windows linux darwin(macos)
	var currArch = fmt.Sprintf("%s-%s", goos, goArch)
	log.Println(currArch)

	var pdfExeName = "pdfcpu"
	if goos == "windows" {
		pdfExeName = "pdfcpu.exe"
	}
	baseDir, _ := os.Getwd()
	var excutePath = filepath.Join(baseDir, "bin", currArch, pdfExeName)
	var mergeDstPath = filepath.Join(baseDir, "public", "merge_tmp")
	return excutePath, mergeDstPath
}

func (controller *ApiController) getUploadPath() string {
	// 时间格式必须按照这个来： RFC3339     = "2006-01-02T15:04:05Z07:00"
	datetime := time.Now().Format("20060102")

	baseDir, _ := os.Getwd()
	//log.Println("ApiController--------->",baseDir)
	dstPath := filepath.Join(baseDir, "public", "upload", datetime)
	return dstPath
}

// 返回上传文件的绝对路径或名称
func (controller *ApiController) getUploadFileName(ext string, isAllPath bool) string {
	dstPath := controller.getUploadPath()
	dstFileName := uuid.New().String() + ext
	dstFileAbsPath := filepath.Join(dstPath, dstFileName)

	_, err := os.Stat(dstPath)
	res := os.IsNotExist(err)
	if res == true {
		os.MkdirAll(dstPath, os.ModePerm)
	}

	if isAllPath {
		return dstFileAbsPath
	}
	return dstFileName
}
