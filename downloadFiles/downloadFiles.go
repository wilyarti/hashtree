package downloadFiles

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
)

const MAX = 5

func Download(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) (error, []string) {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(filelist))

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= 5; w++ {
		go DownloadFile(bucket, url, secure, accesskey, secretkey, enckey, w, jobs, results)
	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	for filepath, hash := range filelist {
		job := make(map[string]string)
		job[hash] = filepath
		jobs <- job
	}
	close(jobs)

	var failed []string
	// Finally we collect all the results of the work.
	for a := 1; a <= len(filelist); a++ {
		failed = append(failed, <-results)
	}
	close(results)
	var count float64 = 0
	var errCount float64 = 0
	for _, msg := range failed {
		if msg != "" {
			errCount++
			fmt.Println(msg)
		} else {
			count++
		}
	}
	if count != 0 {
		fmt.Println(count, " files downloaded successfully.")
	} else {
		fmt.Println(count, " files downloaded successfully.")
		fmt.Println(errCount, " files failed to download.")
	}

	return nil, failed

}

func DownloadFile(bucket string, url string, secure bool, accesskey string, secretkey string, enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, fpath := range j {
			if _, err := os.Stat(fpath); err == nil {
				out := fmt.Sprintf("[D] %s => %s Error! File exists.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			}
			s3Client, err := minio.New(url, accesskey, secretkey, secure)
			if err != nil {
				out := fmt.Sprintf("[D] %s => %s Error downloading file.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			}
			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				out := fmt.Sprintf("[D] %s => %s Error constructing encryption.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			}

			// Encrypt file content and upload to the server
			reader, err := s3Client.GetEncryptedObject(bucket, hash, cbcMaterials)
			if err != nil {
				out := fmt.Sprintf("[D] %s => %s Error opening file.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			}
			defer reader.Close()
			// create file path:
			b := path.Base(fpath)
			basedir := filepath.Dir(fpath)
			os.MkdirAll(basedir, os.ModePerm)
			start := time.Now()
			localFile, err := os.Create(fpath)
			if err != nil {
				out := fmt.Sprintf("[D] %s => %s Error creating file.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			}
			defer localFile.Close()
			if size, err := io.Copy(localFile, reader); err != nil {
				out := fmt.Sprintf("[D] %s => %s Error creating file.", hash, fpath)
				fmt.Println(out)
				results <- hash
				continue
			} else {
				elapsed := time.Since(start)
				var s uint64 = uint64(size)
				if len(hash) == 64 {
					out := fmt.Sprintf("[D][%d]\t(%s)\t(%s)    \t%s => %s", id, elapsed, humanize.Bytes(s), hash[:8], b)
					fmt.Println(out)

				} else {
					out := fmt.Sprintf("[D][%d]\t(%s)\t(%s)    \t%s => %s", id, elapsed, humanize.Bytes(s), hash, b)
					fmt.Println(out)
				}
				results <- ""
			}
		}
	}
}
