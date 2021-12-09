package main

import (
	"fmt"
	"github.com/arsArtD/inovice_merge/app/library"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
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
	var pdfFiles = []string{
		"c8fbeffd-a278-41e6-9346-2bd2a344f662.pdf",
		"e9fa3fbd-599d-4e70-9cb9-93eb44bd8be8.pdf",
		"ed5797cf-5b4b-445f-a2bd-8709cbf9888c.pdf",
	}
	var pdfuploadPath = filepath.Join(baseDir, "public", "upload", "20211207")
	var mergeDstPath = filepath.Join(baseDir, "public", "merge_tmp")
	var sourcePdfPath = make([]string, 0)
	for _, pdfFile := range pdfFiles {
		sourcePdfPath = append(sourcePdfPath, filepath.Join(pdfuploadPath, pdfFile))
	}

	var mergePdfName = filepath.Join(mergeDstPath, "out.pdf")
	var cmdArgs = []string{"merge", mergePdfName}
	cmdArgs = append(cmdArgs, sourcePdfPath...)
	_, res := library.CheckCmdRes(excutePath, cmdArgs)
	fmt.Println(res)

	var mergePdfName2 = filepath.Join(mergeDstPath, "format.pdf")
	cmdArgs = []string{"grid", "--", "bo:off", mergePdfName2, "4", mergePdfName}
	_, res = library.CheckCmdRes(excutePath, cmdArgs)
	fmt.Println(res)
}
