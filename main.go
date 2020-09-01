package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const uploadForm = `
<html>
<head>
  <title>Upload File</title>
</head>
<body>
  <form
	enctype="multipart/form-data"
	action="/upload"
	method="post"
  >
	<input type="file" name="myFile" />
	<input type="submit" value="Go!" />
  </form>
</body>
</html>`

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func rString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var auth = false
var username = "admin"
var password = "dummy"

func logReq(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Printf("[%v] %v", r.Method, r.URL.String())

		if auth {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			if s := r.Header.Get("Authorization"); s != "" {
				if b, err := base64.StdEncoding.DecodeString(s[6:]); err == nil {
					if pair := strings.Split(string(b), ":"); len(pair) == 2 {
						if pair[0] == username && pair[1] == password {
							h.ServeHTTP(w, r)
							return
						}
					}
				}
			}

			http.Error(w, "Not authorized", 401)
			return
		}

		h.ServeHTTP(w, r)

	})
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1 << 30)

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		log.Fatal("Error Retrieving the File")
		return
	}
	defer file.Close()
	log.Printf("[>>] Begin Fileupload: %+v\n", handler.Filename)

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = ioutil.WriteFile(handler.Filename, fileBytes, 0644)

	if err != nil {
		fmt.Println(err)
		return
	}
	// return that we have successfully uploaded our file!
	size := float32(handler.Size)
	suffix := "B"

	if size > 1000 {
		size /= 1000
		suffix = "K"
	}

	if size > 1000 {
		size /= 1000
		suffix = "M"
	}

	fmt.Fprintf(w, "Successfully Uploaded File\n")
	log.Printf("[>>] Done! File Size: %+v%v\n", size, suffix)
}

func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprintf(w, uploadForm)
	} else if r.Method == "POST" {
		uploadFile(w, r)
	}
}

func main() {

	port := flag.Int("p", 8000, "port of fileserver")
	dir := flag.String("dir", ".", "directory to serve")

	authP := flag.Bool("auth", false, "enable auth")
	userP := flag.String("user", "admin", "user for auth")
	passP := flag.String("pass", "dummy", "password for auth")

	flag.Parse()

	auth = *authP
	username = *userP
	password = *passP

	http.Handle("/files/", logReq(http.StripPrefix("/files/", http.FileServer(http.Dir(*dir)))))
	http.Handle("/", logReq(http.RedirectHandler("/files", http.StatusMovedPermanently)))
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.Handle("/upload", logReq(http.HandlerFunc(upload)))

	log.Printf("Serving files at http://0.0.0.0:%v/ ...", *port)

	if auth {
		if password == "dummy" {
			password = rString(10)
		}
		log.Printf("Username is '%v' and Password is '%v' !", username, password)
	}

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))

}
