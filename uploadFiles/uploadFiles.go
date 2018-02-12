package uploadFiles

import (
    "fmt"
	"github.com/minio/minio-go"
    //"sync"
    "log"
)

const MAX = 5

func Upload (Url string , Port int, Accesskey string, Secretkey string, Enckey string, Filelist map[string]string, Bucket string)  error {
    //var errorlist []string
    sem := make(chan int, MAX)
    for hash, filepath := range Filelist {
        sem <- 1 // will block if there is MAX ints in sem
        go func() {
            err := UploadFile (Bucket, hash, filepath, Url, Accesskey, Secretkey)
            <-sem // removes an int from sem, allowing another to proceed
            if err != nil {
               //append(errorlist, filepath) 
               fmt.Println("error!")
            }
        }()
    }
    return nil
}

func UploadFile (Bucket string, hash string, filepath string, Url string, Accesskey string, Secretkey string) error {
        s3Client, err := minio.New(Url, Accesskey, Secretkey, true)
        fmt.Println("[U] ", hash, "=> ", filepath)
        if err != nil {
            return err
        }
        if _, err := s3Client.FPutObject(Bucket, hash, filepath, minio.PutObjectOptions{
            ContentType: "application/csv",
        }); err != nil {
            log.Fatalln(err)
            return err
        }
        return nil
}
