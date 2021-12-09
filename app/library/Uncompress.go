package library

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"github.com/xi2/xz"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
