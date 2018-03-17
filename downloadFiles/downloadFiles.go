/* Copyright <2018> <Wilyarti Howard>
*
* Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
* 
* 1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
* 
* 2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentatio
* n and/or other materials provided with the distribution.
*  
* 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software w
* ithout specific prior written permission.
* 
* THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
* IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
* LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE 
* GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRIC
* T LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SU
* CH DAMAGE.
*/
package downloadFiles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

// New returns an error that formats as the given text.
func New(text string) error {
	return &errorString{text}
}
func (e *errorString) Error() string {
	return e.s
}

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

	var grmsgs []string
	var failed []string
	// Finally we collect all the results of the work.
	for a := 1; a <= len(filelist); a++ {
		grmsgs = append(grmsgs, <-results)
	}
	close(results)
	var count float64 = 0
	var errCount float64 = 0
	for _, msg := range grmsgs {
		if msg != "" {
			errCount++
			failed = append(failed, msg)
		} else {
			count++
		}
	}
	//fmt.Println(count, " files downloaded successfully.")

	if errCount != 0 {
		out := fmt.Sprintf("Failed to download: %v files", errCount)
		return errors.New(out), failed
	} else {
		return nil, failed
	}

}

func DownloadFile(bucket string, url string, secure bool, accesskey string, secretkey string, enckey string, id int, jobs <-chan map[string]string, results chan<- string) {
	for j := range jobs {
		// hash is reversed: filepath => hash
		for fpath, hash := range j {
			if _, err := os.Stat(fpath); err == nil {
				data, err := ioutil.ReadFile(fpath)
				if err != nil {
					out := fmt.Sprintf("[!] %s => %s failed to verify: %s", hash, fpath, err)
					fmt.Println(out)
					results <- hash
					break
				}

				digest := sha256.Sum256(data)
				checksum := hex.EncodeToString(digest[:])
				if hash != checksum {
					out := fmt.Sprintf("[!] %s => %s local file differs from remote version!", hash, fpath)
					fmt.Println(out)
					results <- hash
					break

				} else {
					b := path.Base(fpath)
					out := fmt.Sprintf("[V]\t%s => %s", hash[:8], b)
					fmt.Println(out)
					results <- ""
					break
				}
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
					data, err := ioutil.ReadFile(fpath)
					if err != nil {
						out := fmt.Sprintf("[!] %s => %s failed to download: %s", hash, fpath, err)
						fmt.Println(out)
						results <- hash
						break
					}

					digest := sha256.Sum256(data)
					checksum := hex.EncodeToString(digest[:])
					if hash != checksum {
						out := fmt.Sprintf("[!] %s => %s checksum mismatch!", hash, fpath)
						fmt.Println(out)
						results <- hash
						break

					}
					out := fmt.Sprintf("[D][V]\t(%s)\t(%s)    \t%s => %s", elapsed, humanize.Bytes(s), hash[:8], b)
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
