package downloadFiles

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
)

const MAX = 5

func Download(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) error {
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
	for _, msg := range failed {
		if msg != "" {
			fmt.Println(msg)
		}
	}

	return nil

}

func DownloadFile(bucket string, url string, secure bool, accesskey string, secretkey string, enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, fpath := range j {
			s3Client, err := minio.New(url, accesskey, secretkey, secure)
			if err != nil {
				results <- fmt.Sprintf("[D] %s => %s failed to download: %s", hash, fpath, err)
			}
			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				results <- fmt.Sprintf("[D] %s => %s failed to download: %s", hash, fpath, err)
			}

			// Encrypt file content and upload to the server
			reader, err := s3Client.GetEncryptedObject(bucket, hash, cbcMaterials)
			if err != nil {
				results <- fmt.Sprintf("[D] %s => %s failed to download: %s", hash, fpath, err)
			}
			defer reader.Close()
			// create file path:
			basedir := filepath.Dir(fpath)
			os.MkdirAll(basedir, os.ModePerm)
			localFile, err := os.Create(fpath)
			if err != nil {
				results <- fmt.Sprintf("[D] %s => %s failed to download: %s", hash, fpath, err)
			}
			defer localFile.Close()

			if _, err := io.Copy(localFile, reader); err != nil {
				results <- fmt.Sprintf("[D] %s => %s failed to download: %s", hash, fpath, err)
			} else {
				out := fmt.Sprintf("[D][%d] %s <= %s", id, hash, fpath)
				fmt.Println(out)
				results <- ""
			}
		}
	}
}
