package uploadFiles

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go"
	"github.com/minio/sio"
	"golang.org/x/crypto/argon2"
)

/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2018 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const (
	// SSE DARE package block size.
	sseDAREPackageBlockSize = 64 * 1024 // 64KiB bytes

	// SSE DARE package meta padding bytes.
	sseDAREPackageMetaSize = 32 // 32 bytes
)

const MAX = 3

// EncryptedSize returns the size of the object after encryption.
// An encrypted object is always larger than a plain object
// except for zero size objects.
func getEncryptedSize(size int64) int64 {
	ssize := (size / sseDAREPackageBlockSize) * (sseDAREPackageBlockSize + sseDAREPackageMetaSize)
	if mod := size % (sseDAREPackageBlockSize); mod > 0 {
		ssize += mod + sseDAREPackageMetaSize
	}
	return ssize
}

func Upload(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) (error, []string) {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(filelist))

	// This starts up MAX workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= MAX; w++ {
		go UploadFile(bucket, url, secure, accesskey, secretkey, enckey, w, jobs, results)
	}

	// Here we send MAX `jobs` and then `close` that
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
			// break unrecoverable errors
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- hash
				break
			}

			// minio-go example code modified:
			object, err := os.Open(filepath)
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- hash
				break
			}
			defer object.Close()
			objectStat, err := object.Stat()
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- hash
				break
			}
			password := []byte(enckey)
			salt := []byte(path.Join(bucket, hash))
			encrypted, err := sio.EncryptReader(object, sio.Config{
				// generate a 256 bit long key.
				Key: argon2.IDKey(password, salt, 1, 64*1024, 4, 32),
			})
			if err != nil {
				out := fmt.Sprintf("[F] %s => %s failed to upload: %s", hash, filepath, err)
				fmt.Println(out)
				results <- hash
				break
			}

			// Encrypt file content and upload to the server
			// try multiple times
			b := path.Base(filepath)
			for i := 0; i < 4; i++ {
				start := time.Now()
				size, err := s3Client.PutObject(bucket, hash, encrypted, getEncryptedSize(objectStat.Size()), minio.PutObjectOptions{})
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
					results <- ""
					break
				}
			}
		}
	}
}
