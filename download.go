package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/dustin/go-humanize"
	"invoice_merge/app/library"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const PDFCPU_VERSION = "0.3.13"

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete\n", humanize.Bytes(wc.Total))
}

func main() {
	var goArch = runtime.GOARCH //amd64 arm64
	var goos = runtime.GOOS     //windows linux darwin(macos)
	goos = "linux"
	var currArch = fmt.Sprintf("%s-%s", goos, goArch)
	var zipFileName = transferArchStr(goos, goArch)
	if zipFileName == "" {
		panic("not supported arch:" + currArch)
	}

	var unzipDst = "bin/" + currArch

	//https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.13
	// https://github.com/pdfcpu/pdfcpu/releases/download/v0.3.13/pdfcpu_0.3.13_Windows_x86_64.zip
	var downloadUrl = "https://github.com/pdfcpu/pdfcpu/releases/download/v0.3.13/" + zipFileName
	fmt.Printf("download url:%s\n", downloadUrl)

	zipfileExists, excutePkgExists := checkIfExistPdfcpu(goos, unzipDst, zipFileName)

	if excutePkgExists {
		fmt.Print("pdfcpu is exists!")
		return
	}
	if !zipfileExists {
		// zip 文件不存在进行下载并校验checksum
		err, fileHash := DownloadFile(zipFileName, downloadUrl)
		if err != nil {
			panic(err)
		}
		// checkSum
		checkSumMap := checkSum()
		if checkSumMap[zipFileName] != fileHash {
			panic(fmt.Sprintf("file %s check sum err, right check_sum:%s, file check_sum is:%s!\n", zipFileName, checkSumMap[zipFileName], fileHash))
		}
	} else {
		fmt.Println("pdfcpu tar or zip is exists, not need download.")
	}

	if goos == "windows" {
		fmt.Printf("unzip file %s to:%s\n", zipFileName, unzipDst)
		// 使用zip方法解压缩文件
		files, err := library.Unzip(zipFileName, unzipDst)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))
	} else {
		fmt.Printf("unxz file %s to:%s\n", zipFileName, unzipDst)
		// 使用zip方法解压缩文件
		err := library.UnXz(zipFileName, unzipDst)
		if err != nil {
			log.Fatal(err)
		}
	}

	// 再次判断,有可能会出现匹配的pdfcpu无法执行的情况
	_, excutePkgExists = checkIfExistPdfcpu(goos, unzipDst, zipFileName)

	if !excutePkgExists {
		fmt.Print("您的系统无法使用pdfcpu!")
		return
	}
}

func checkIfExistPdfcpu(goos, unzipDst, zipFileName string) (bool, bool) {
	var dirBase, _ = os.Getwd()
	var excutePath = filepath.Join(dirBase, unzipDst, "pdfcpu")
	if goos == "windows" {
		excutePath = filepath.Join(dirBase, unzipDst, "pdfcpu.exe")
	}

	var excuteExists = false
	_, excuteRes := library.CheckCmdRes(excutePath, []string{"version"})
	if strings.Contains(excuteRes, PDFCPU_VERSION) {
		excuteExists = true
	}

	// 判断是否存在压缩文件
	var pkgExists = false
	var pkgPath = filepath.Join(dirBase, zipFileName)
	log.Println("pkgpath------->", pkgPath)
	_, pkgErr := os.Lstat(pkgPath)

	if pkgErr == nil {
		pkgExists = true
	}
	return pkgExists, excuteExists
}

func checkSum() map[string]string {
	//07fbc336aab5257f29244d8839b3c990338462e5a9a3c9c167c04808c8fe7961  pdfcpu_0.3.13_Linux_x86_64.tar.xz
	//7c4051c919dc3c209e65a171510219197d2efa4ff6de01d53cbdd7b7be55f7ca  pdfcpu_0.3.13_macOS_x86_64.tar.xz
	//99f9c1a5d32b0066436e67fe9e35ba1251ad865bc24ee51ca63ba617e5230546  pdfcpu_0.3.13_Windows_x86_64.zip
	//c19684a8c5976ab84a609f4bccaafea727ca1a2b6d5f61583232a5e0bc9e56b1  pdfcpu_0.3.13_Linux_i386.tar.xz
	//edecea5d9be7de96b6698957be22553a04a09e8b279c08f9b4c205b93738affa  pdfcpu_0.3.13_Windows_i386.zip
	var result = make(map[string]string)
	result["pdfcpu_0.3.13_Linux_x86_64.tar.xz"] = "07fbc336aab5257f29244d8839b3c990338462e5a9a3c9c167c04808c8fe7961"
	result["pdfcpu_0.3.13_macOS_x86_64.tar.xz"] = "7c4051c919dc3c209e65a171510219197d2efa4ff6de01d53cbdd7b7be55f7ca"
	result["pdfcpu_0.3.13_Windows_x86_64.zip"] = "99f9c1a5d32b0066436e67fe9e35ba1251ad865bc24ee51ca63ba617e5230546"
	result["pdfcpu_0.3.13_Linux_i386.tar.xz"] = "c19684a8c5976ab84a609f4bccaafea727ca1a2b6d5f61583232a5e0bc9e56b1"
	result["pdfcpu_0.3.13_Windows_i386.zip"] = "edecea5d9be7de96b6698957be22553a04a09e8b279c08f9b4c205b93738affa"
	return result
}

func transferArchStr(goos, arch string) string {
	var result string
	if goos == "windows" {
		result = "pdfcpu_0.3.13_Windows_i386.zip"
		if strings.Contains(arch, "64") {
			result = "pdfcpu_0.3.13_Windows_x86_64.zip"
		}
	}
	if goos == "darwin" {
		result = "pdfcpu_0.3.13_macOS_x86_64.tar.xz"
	}
	if goos == "linux" {
		result = "pdfcpu_0.3.13_Linux_i386.tar.xz"
		if strings.Contains(arch, "64") {
			result = "pdfcpu_0.3.13_Linux_x86_64.tar.xz"
		}
	}

	return result
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func DownloadFile(filepath string, url string) (error, string) {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err, ""
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err, ""
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err, ""
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err, ""
	}

	hasher := sha256.New()
	f, err := os.Open(filepath)
	if err != nil {
		return err, ""
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return err, ""
	}
	fmt.Printf("%s checksum is:%x\n", filepath, hasher.Sum(nil))

	return nil, fmt.Sprintf("%x", hasher.Sum(nil))
}
