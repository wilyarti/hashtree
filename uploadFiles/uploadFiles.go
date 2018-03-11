package uploadFiles

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
)

const MAX = 2

func Upload(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) (error, []string) {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(filelist))

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= MAX; w++ {
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
		fmt.Println(count, " files uploaded successfully.")
	} else {
		fmt.Println(count, " files uploaded successfully.")
		fmt.Println(errCount, " files failed to upload.")
	}

	return nil, failed

}

func UploadFile(bucket string, url string, secure bool, accesskey string, secretkey string, enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, filepath := range j {
			s3Client, err := minio.New(url, accesskey, secretkey, secure)
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- filepath
			}

			// Open a local file that we will upload
			file, err := os.Open(filepath)
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- filepath
			}
			defer file.Close()

			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- filepath
			}

			// Encrypt file content and upload to the server
			// try multiple times
			b := path.Base(filepath)
			for i := 0; i < 4; i++ {
				start := time.Now()
				size, err := s3Client.PutEncryptedObject(bucket, hash, file, cbcMaterials)
				elapsed := time.Since(start)
				if err != nil {
					if i == 3 {
						out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
						fmt.Println(out)
						results <- hash
						break
					}
				} else {
					var s uint64 = uint64(size)
					if len(hash) == 64 {
						out := fmt.Sprintf("[U][%d]\t(%s)\t(%s)    \t%s => %s", i, elapsed, humanize.Bytes(s), hash[:8], b)
						fmt.Println(out)

					} else {
						out := fmt.Sprintf("[U][%d]\t(%s)\t(%s)    \t%s => %s", i, elapsed, humanize.Bytes(s), hash, b)
						fmt.Println(out)
					}
					break
					results <- ""
				}
			}
		}
	}
}
