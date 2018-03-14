package downloadFiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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
// max number of concurrent downloads
const MAX = 3

const (
	// SSE DARE package block size.
	sseDAREPackageBlockSize = 64 * 1024 // 64KiB bytes

	// SSE DARE package meta padding bytes.
	sseDAREPackageMetaSize = 32 // 32 bytes
)

func decryptedSize(encryptedSize int64) (int64, error) {
	if encryptedSize == 0 {
		return encryptedSize, nil
	}
	size := (encryptedSize / (sseDAREPackageBlockSize + sseDAREPackageMetaSize)) * sseDAREPackageBlockSize
	if mod := encryptedSize % (sseDAREPackageBlockSize + sseDAREPackageMetaSize); mod > 0 {
		if mod < sseDAREPackageMetaSize+1 {
			return -1, errors.New("object is tampered")
		}
		size += mod - sseDAREPackageMetaSize
	}
	return size, nil
}

func Download(url string, port int, secure bool, accesskey string, secretkey string, enckey string, filelist map[string]string, bucket string) (error, []string) {
	// break up map into 5 parts
	jobs := make(chan map[string]string, MAX)
	results := make(chan string, len(filelist))

	// This starts up MAX workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= MAX; w++ {
		go DownloadFile(bucket, url, secure, accesskey, secretkey, enckey, w, jobs, results)
	}

	// Here we send MAX `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	for hash, fpath := range filelist {
		job := make(map[string]string)
		job[hash] = fpath
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
		// hash is reversed: filepath => hash
		for fpath, hash := range j {
			if _, err := os.Stat(fpath); err == nil {
				out := fmt.Sprintf("[!] %s => %s Error! File exists.", hash, fpath)
				fmt.Println(out)
				results <- hash
				break
			}
			s3Client, err := minio.New(url, accesskey, secretkey, secure)
			// break unrecoverable errors
			if err != nil {
				out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
				fmt.Println(out)
				results <- hash
				break
			}
			////
			// create directorys for files:
			// create file path:
			b := path.Base(fpath)
			basedir := filepath.Dir(fpath)
			os.MkdirAll(basedir, os.ModePerm)
			////
			// minio-go download object code:
			// Encrypt file content and upload to the server
			// try multiple times
			for i := 0; i < 4; i++ {
				start := time.Now()
				obj, err := s3Client.GetObject(bucket, hash, minio.GetObjectOptions{})
				if err != nil {
					if i == 3 {
						out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
						fmt.Println(out)
						results <- hash
						break
					}
				}

				objSt, err := obj.Stat()
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
					fmt.Println(out)
					results <- hash
					break
				}

				size, err := decryptedSize(objSt.Size)
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
					fmt.Println(out)
					results <- hash
					break
				}
				localFile, err := os.Create(fpath)
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s Error creating file.", hash, fpath)
					fmt.Println(out)
					results <- hash
					break
				}
				defer localFile.Close()

				password := []byte(enckey)              // Change as per your needs.
				salt := []byte(path.Join(bucket, hash)) // Change as per your needs.
				decrypted, err := sio.DecryptReader(obj, sio.Config{
					// generate a 256 bit long key.
					Key: argon2.IDKey(password, salt, 1, 64*1024, 4, 32),
				})
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
					fmt.Println(out)
					results <- hash
					break
				}
				dsize, err := io.CopyN(localFile, decrypted, size)
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
					fmt.Println(out)
					results <- hash
					break
				}
				elapsed := time.Since(start)
				var s uint64 = uint64(dsize)
				if len(hash) == 64 {
					out := fmt.Sprintf("[D][%d]\t(%s)\t(%s)    \t%s => %s", i, elapsed, humanize.Bytes(s), hash[:8], b)
					fmt.Println(out)
					results <- ""
					break

				} else {
					out := fmt.Sprintf("[D][%d]\t(%s)\t(%s)    \t%s => %s", i, elapsed, humanize.Bytes(s), hash, b)
					fmt.Println(out)
					results <- ""
					break
				}
			}

		}
	}
}
