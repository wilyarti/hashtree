// +build ignore

/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 Minio, Inc.
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

package main

import (
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-testfile, my-bucketname and
	// my-objectname are dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", "YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", true)
	if err != nil {
		log.Fatalln(err)
	}

	// Open a local file that we will upload
	file, err := os.Open("my-testfile")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	//// Build an asymmetric key from private and public files
	//
	privateKey, err := ioutil.ReadFile("private.key")
	if err != nil {
		log.Fatal(err)
	}

	publicKey, err := ioutil.ReadFile("public.key")
	if err != nil {
		log.Fatal(err)
	}

	asymmetricKey := NewAsymmetricKey(privateKey, publicKey)
	if err != nil {
		log.Fatal(err)
	}

	// Build encryption materials which will encrypt uploaded data
	cbcMaterials, err := encrypt.NewCBCSecureMaterials(asymmetricKey)
	if err != nil {
		log.Fatalln(err)
	}

	// Encrypt file content and upload to the server
	n, err := s3Client.PutEncryptedObject("my-bucketname", "my-objectname", file, cbcMaterials)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Uploaded", "my-objectname", " of size: ", n, "Successfully.")
}
