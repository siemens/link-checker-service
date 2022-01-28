// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of
// Attribution-ShareAlike 4.0 International (CC BY-SA 4.0) license
// https://creativecommons.org/licenses/by-sa/4.0/
// SPDX-License-Identifier: CC-BY-SA-4.0
//go:generate go get github.com/rakyll/statik
//go:generate statik -include=*.html,*.txt,*.ico,*.js,*.css
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rakyll/statik/fs"
	_ "github.com/siemens/link-checker-service/test/jquery_example/statik"
)

func main() {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	port := "8092"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(statikFS)))
	log.Println("run the link checker service with appropriate CORS headers, e.g. `link-checker-service serve  -o http://localhost:" + port + "`")
	log.Println("open http://localhost:" + port)

	if err = http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("%v", err)
	}
}
