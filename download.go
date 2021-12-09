package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/xi2/xz"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
		files, err := Unzip(zipFileName, unzipDst)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))
	} else {
		fmt.Printf("unxz file %s to:%s\n", zipFileName, unzipDst)
		// 使用zip方法解压缩文件
		err := UnXz(zipFileName, unzipDst)
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
	_, excuteRes := checkCmdRes(excutePath, []string{"version"})
	if strings.Contains(excuteRes, PDFCPU_VERSION) {
		excuteExists = true
	}

	// 判断是否存在压缩文件
	var pkgExists = false
	var pkgPath = filepath.Join(dirBase, zipFileName)
	_, pkgErr := os.Stat(pkgPath)
	log.Println("pkgpath------->", pkgPath)
	if pkgErr == nil {
		pkgExists = true
	}
	//log.Fatal("===============")

	return pkgExists, excuteExists
}

func checkCmdRes(cmdPath string, args []string) (bool, string) {
	_, excutePathErr := os.Stat(cmdPath)
	if excutePathErr == nil {
		cmd := exec.Command(cmdPath, args...)

		stdout, err := cmd.StdoutPipe()

		if err != nil {
			log.Println("cmd.StdoutPipe...", err)
			return false, ""
		}

		if err := cmd.Start(); err != nil {
			log.Println("cmd.Start...", err)
			return false, ""
		}

		data, err := ioutil.ReadAll(stdout)

		if err != nil {
			log.Println("ioutil.ReadAll...", err)
			return false, ""
		}

		if err := cmd.Wait(); err != nil {
			log.Println("cmd wait...", err)
			return false, ""
		}

		// 通过pdfcpu的执行结果中是否包含版本信息来判断
		var excuteRes = string(data)
		fmt.Printf("cmd excute result%s\n", excuteRes)
		return true, excuteRes
	}
	return false, ""
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

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		// 直接将解压后的文件放到指定目录下
		fpath := filepath.Join(dest, filepath.Base(f.Name))

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func Untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnXz(tarball, target string) error {
	// Open a file
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	// Create an xz Reader
	r, err := xz.NewReader(f, 0)
	if err != nil {
		return err
	}
	// Create a tar Reader
	tr := tar.NewReader(r)
	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			// create a directory
			fmt.Println("creating:   " + hdr.Name)
			err = os.MkdirAll(hdr.Name, 0777)
			if err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:

			// Store filename/path for returning and using later on
			// hdr.Name 可能是名字的拼接，直接将文件放到指定目录下
			fpath := filepath.Join(target, filepath.Base(hdr.Name))

			if !strings.HasPrefix(fpath, filepath.Clean(target)+string(os.PathSeparator)) {
				return fmt.Errorf("%s: illegal file path", fpath)
			}

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return err
			}

			// Close the file without defer to close before next iteration of loop
			defer outFile.Close()
		}
	}
	defer f.Close()
	return nil
}
