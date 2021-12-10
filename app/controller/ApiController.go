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

func (controller *ApiController) Upload(context *gin.Context) {

	// Multipart form
	form, err := context.MultipartForm()
	if err != nil {
		context.JSON(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		context.JSON(http.StatusBadRequest, fmt.Sprintf("get upload file err"))
		return
	}

	var pdfFiles = make([]string, 0)
	for _, file := range files {
		ext := filepath.Ext(file.Filename)
		if ext != ".pdf" {
			context.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", "file ext is not allowed!"))
			return
		}

		var sourceUploadFileName = controller.getUploadFileName(ext, true)
		if err := context.SaveUploadedFile(file, sourceUploadFileName); err != nil {
			context.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}
		pdfFiles = append(pdfFiles, filepath.Base(sourceUploadFileName))
	}

	var result = make(map[string][]string, 0)
	result["file_list"] = pdfFiles
	context.JSON(http.StatusOK, result)
}

// @TODO:  前端传入上传文件名称
func (controller *ApiController) FormatPdf(context *gin.Context) {
	// Multipart form
	type reqData struct {
		FileList []string `form:"file_list[]" binding:"required"`
	}
	var reqD reqData
	err := context.ShouldBind(&reqD)
	if err != nil {
		log.Println("ApiController:FormatPdf-->", err)
		context.JSON(http.StatusBadRequest, "wrong request data")
		return
	}
	log.Printf("%#v", reqD)
	var uploadFiles = reqD.FileList
	if len(uploadFiles) == 0 {
		context.JSON(http.StatusBadRequest, "wrong request data, upload file is 0!")
		return
	}

	var pdfFiles = make([]string, 0)
	var uploadBase = controller.getUploadPath()
	for _, pdfName := range uploadFiles {
		var pdfAbsPath = filepath.Join(uploadBase, pdfName)
		_, pdfErr := os.Lstat(pdfAbsPath)

		if pdfErr != nil {
			context.JSON(http.StatusBadRequest, "upload file is not exists!")
			return
		}
		pdfFiles = append(pdfFiles, pdfAbsPath)
	}

	var dstMergePdfFileName = controller.getUploadFileName(".pdf", false)
	pdfcpuPath, megeDstPath := controller.getUploadConfig()

	// 合并pdf
	var mergePdfPath = filepath.Join(megeDstPath, dstMergePdfFileName)
	var cmdArgs = []string{"merge", mergePdfPath}
	cmdArgs = append(cmdArgs, pdfFiles...)
	mergeRes, excuteRes := library.CheckCmdRes(pdfcpuPath, cmdArgs)
	if !mergeRes {
		context.JSON(http.StatusBadRequest, fmt.Sprintf("merg pdf error"))
		return
	}
	log.Println("ApiController:upload:合并pdf----->\n", excuteRes)

	// pdf内容重新排版
	var resPdfFilePath = controller.getUploadFileName(".pdf", false)
	resPdfFilePath = filepath.Join(megeDstPath, resPdfFilePath)
	var cmdArgForReformat = []string{"grid", "--", "bo:off", resPdfFilePath, "4", mergePdfPath}
	formatRes, excuteRes := library.CheckCmdRes(pdfcpuPath, cmdArgForReformat)
	if !formatRes {
		context.JSON(http.StatusBadRequest, fmt.Sprintf("format pdf error"))
		return
	}
	log.Println("ApiController:upload:pdf内容重构----->\n", excuteRes)

	// 直接输入文件流
	//downloadFile, err := os.Open(resPdfFilePath) //Create a file
	//if err != nil {
	//	context.JSON(http.StatusBadRequest, "merge pdf not found!")
	//	return
	//}
	//defer downloadFile.Close()
	//context.Writer.Header().Add("Content-type", "application/octet-stream")
	//_, err = io.Copy(context.Writer, downloadFile)
	//if err != nil {
	//	context.JSON(http.StatusBadRequest, "文件加载失败!")
	//	return
	//}

	var pdfResponse = make(map[string]string, 0)
	pdfResponse["pdf_url"] = "/merge_tmp/" + filepath.Base(resPdfFilePath)

	context.JSON(http.StatusOK, pdfResponse)
	return
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
