package uploadFiles

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
)

const MAX = 5

func Upload(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) error {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(filelist))

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= 5; w++ {
		go UploadFile(bucket, url, secure, accesskey, secretkey, enckey, w, jobs, results)
	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	for hash, filepath := range filelist {
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

func UploadFile(bucket string, url string, secure bool, accesskey string, secretkey string, enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, filepath := range j {
			s3Client, err := minio.New(url, accesskey, secretkey, secure)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				log.Fatal(err)
			}

			// Open a local file that we will upload
			file, err := os.Open(filepath)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				log.Fatal(err)
			}
			defer file.Close()

			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}

			// Encrypt file content and upload to the server
			b := path.Base(filepath)
			start := time.Now()
			size, err := s3Client.PutEncryptedObject(bucket, hash, file, cbcMaterials)
			elapsed := time.Since(start)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			} else {
				out := fmt.Sprintf("[U][%d]\t(%s)\t(%d)\t%s => %s", id, elapsed, size, hash[:8], b)
				fmt.Println(out)
				results <- ""
			}
		}
	}
}
