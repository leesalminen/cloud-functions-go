// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
	"strings"
	"os"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"

	"./nodego"
)

// define Success JSON
type Success struct {
	Success bool `json:"success"`
	Error   bool `json:"error"`
	Message string `json:"message"`
	Url     string `json:"url"`
}

// define Error JSON
type Error struct {
	Success bool `json:"success"`
	Error   bool `json:"error"`
	Message string `json:"message"`
}

// returns a microtime()
func makeTimestamp() int64 {
    return time.Now().UnixNano() / int64(time.Millisecond)
}

// function that uploads the image, used by the echo route
func upload(c echo.Context) error {
	// HTTP referer check, tis basic
	referer := fmt.Sprint(c.Request().Header.Get("Referer"))

	// we only want referer of gingrapp.com to use this
	if !strings.Contains(referer, "gingrapp.com") {
		e := &Error{
			Success: false,
			Error: true,
			Message: "Invalid Request",
		}

		return c.JSON(400, e)
	}

	// Source from POST
	file, err := c.FormFile("file")
	if err != nil {
		e := &Error{
			Success: false,
			Error: true,
			Message: err.Error(),
		}
		return c.JSON(404, e)
	}

	// get a reader for it
	src, err := file.Open()
	if err != nil {
		e := &Error{
			Success: false,
			Error: true,
			Message: err.Error(),
		}
		return c.JSON(500, e)
	}
	defer src.Close()

	// connect to google cloud storage
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
	    e := &Error{
			Success: false,
			Error: true,
			Message: err.Error(),
		}
		return c.JSON(500, e)
	}

	// name of bucket we're using to save files to
	bucketName := os.Getenv("GOOGLE_UPLOAD_BUCKET")

	// get the bucket
	bucket := client.Bucket(bucketName)

	// invent a unique file name for the cloud
	t := time.Now().Format("2006/01/02/")
	ts := fmt.Sprint(makeTimestamp())
	filename := t + ts + "-" + file.Filename

	// create an object in GCS
	obj := bucket.Object(filename)

	// Write something to obj.
	// w implements io.Writer.
	w := obj.NewWriter(ctx)

	// Write some text to obj. This will overwrite whatever is there.
	if _, err = io.Copy(w, src); err != nil {
		e := &Error{
			Success: false,
			Error: true,
			Message: err.Error(),
		}
		return c.JSON(500, e)
	}

	// Close, just like writing a file.
	if err := w.Close(); err != nil {
	   e := &Error{
			Success: false,
			Error: true,
			Message: err.Error(),
		}
		return c.JSON(500, e)
	}

	// URL of the uploaded file
	u := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, filename)

	s := &Success{
		Success: true,
		Error: false,
		Message: "File uploaded successfully.",
		Url: u,
	}

	return c.JSON(http.StatusOK, s)
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1024M"))
	e.Use(middleware.CORS())

	e.POST("/upload", upload)

	e.Logger.Fatal(e.Start(":1323"))
}