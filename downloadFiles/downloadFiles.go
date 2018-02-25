package downloadFiles

import (
	"fmt"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
	"io"
	"os"
)

const MAX = 5

func Download(Url string, Port int, Accesskey string, Secretkey string, Enckey string, Filelist map[string]string, Bucket string) error {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(Filelist))

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= 5; w++ {
		go DownloadFile(Bucket, Url, Accesskey, Secretkey, Enckey, w, jobs, results)
	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	for hash, filepath := range Filelist {
		job := make(map[string]string)
		job[hash] = filepath
		jobs <- job
	}
	close(jobs)

	var failed []string
	// Finally we collect all the results of the work.
	for a := 1; a <= len(Filelist); a++ {
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

func DownloadFile(Bucket string, Url string, Accesskey string, Secretkey string, Enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, filepath := range j {
			s3Client, err := minio.New(Url, Accesskey, Secretkey, true)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}

			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(Enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}

			// Encrypt file content and upload to the server
			reader, err := s3Client.GetEncryptedObject(Bucket, hash, cbcMaterials)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}
			defer reader.Close()
			localFile, err := os.Create(filepath)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}
			defer localFile.Close()

			if _, err := io.Copy(localFile, reader); err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			} else {
				out := fmt.Sprintf("[D][%d] %s <= %s", id, hash, filepath)
				fmt.Println(out)
				results <- ""
			}
		}
	}
}
