/*
Copyright 2018 Google, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/golang/glog"
)

var (
	zipfile = flag.String("zipfile", "", "GCS archive object to fetch")
	bucket  = flag.String("bucket", "", "GCS bucket to fetch from")
)

func main() {
	flag.Parse()

	if *bucket == "" {
		glog.Fatal("Must specify --bucket")
	}
	if *zipfile == "" {
		glog.Fatal("Must specify --zipfile")
	}
	// This request has no credentials, so the object must be publicly readable.
	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", *bucket, *zipfile)
	resp, err := http.Get(url)
	if err != nil {
		glog.Fatal("Fetching %q: %v", url, err)
	}
	defer resp.Body.Close()

	// zip files must be buffered to read.
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		glog.Fatalf("Failed to read HTTP response body: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		glog.Fatalf("zip.NewReader: %v", err)
	}

	for _, zf := range zr.File {
		func() { // wrapped in a func for defer
			glog.Infof("Unzipping", zf.Name)
			rc, err := zf.Open()
			if err != nil {
				glog.Fatal(err)
			}
			defer rc.Close()
			of, err := os.Create(zf.Name)
			if err != nil {
				glog.Fatalf("Could not open %q: %v", zf.Name, err)
			}
			defer of.Close()
			if _, err := io.Copy(of, rc); err != nil {
				glog.Fatalf("Writing %s: %v", zf.Name, err)
			}
		}()
	}

}
