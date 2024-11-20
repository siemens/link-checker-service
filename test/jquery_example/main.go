// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of
// Attribution-ShareAlike 4.0 International (CC BY-SA 4.0) license
// https://creativecommons.org/licenses/by-sa/4.0/
// SPDX-License-Identifier: CC-BY-SA-4.0
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed public/*.html public/*.txt public/*.ico
var content embed.FS

func main() {
	publicFS, err := fs.Sub(content, "public")
	if err != nil {
		log.Fatal(err)
	}

	port := "8092"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(publicFS))))
	log.Println("run the link checker service with appropriate CORS headers, e.g. `link-checker-service serve  -o http://localhost:" + port + "`")
	log.Println("open http://localhost:" + port)

	if err = http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("%v", err)
	}
}
