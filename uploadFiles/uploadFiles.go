package uploadFiles

import (
	"fmt"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
	//"log"
	"os"
)

const MAX = 5

func Upload(Url string, Port int, Accesskey string, Secretkey string, Enckey string, Filelist map[string]string, Bucket string) error {
	// create channels
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(Filelist))

	// Start five workers
	for w := 1; w <= 5; w++ {
		go UploadFile(Bucket, Url, Accesskey, Secretkey, Enckey, w, jobs, results)
	}

	// Itterate through map and push jobs to workers
	for hash, filepath := range Filelist {
		job := make(map[string]string)
		job[hash] = filepath
		jobs <- job
	}
	close(jobs)

	var failed []string
	//Collect errors
	for a := 1; a <= len(Filelist); a++ {
		failed = append(failed, <-results)
	}
	close(results)
	// Igore empty strings
	for _, msg := range failed {
		if msg != "" {
			fmt.Println(msg)
		}
	}

	return nil

}

func UploadFile(Bucket string, Url string, Accesskey string, Secretkey string, Enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		for hash, filepath := range j {
			s3Client, err := minio.New(Url, Accesskey, Secretkey, true)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}

			// Open a local file that we will upload
			file, err := os.Open(filepath)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				//log.Fatalln(err)
			}
			defer file.Close()

			// Build a symmetric key
			symmetricKey := encrypt.NewSymmetricKey([]byte(Enckey))

			// Build encryption materials which will encrypt uploaded data
			cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			}

			// Encrypt file content and upload to the server
			_, err = s3Client.PutEncryptedObject(Bucket, hash, file, cbcMaterials)
			if err != nil {
				results <- fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
			} else {
				out := fmt.Sprintf("[U][%d] %s => %s", id, hash, filepath)
				fmt.Println(out)
				results <- ""
			}
		}
	}
}
