// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of
// Attribution-ShareAlike 4.0 International (CC BY-SA 4.0) license
// https://creativecommons.org/licenses/by-sa/4.0/
// SPDX-License-Identifier: CC-BY-SA-4.0
package main

import (
	"log"
	"time"

	"github.com/ahmetb/go-linq"
	"github.com/go-resty/resty/v2"
	"mvdan.cc/xurls/v2"
)

const serviceURL = "http://localhost:8080/checkUrls"
const contentURL = "https://raw.githubusercontent.com/binhnguyennus/awesome-scalability/master/README.md"

func main() {

	var text string
	timeIt("extracting text", func() {
		text = extractText(contentURL)
	})

	var links []string
	timeIt("extracting links", func() {
		links = extractLinks(text)
	})
	count := len(links)
	log.Println("Collected", count, "links")

	var response CheckURLsResponse

	timeIt("checking links", func() {
		response = checkURLs(links)
	})

	groups := linq.From(response.Urls).
		Where(func(url interface{}) bool {
			urlResponse := url.(URLStatusResponse)
			return urlResponse.HTTPStatus >= 300 ||
				urlResponse.Status != "ok"
		}).
		GroupBy(func(url interface{}) interface{} {
			return url.(URLStatusResponse).HTTPStatus
		}, func(url interface{}) interface{} {
			return url
		}).
		OrderBy(func(group interface{}) interface{} {
			return group.(linq.Group).Key
		}).
		Results()

	totalBroken := 0
	for _, group := range groups {
		g := group.(linq.Group)
		for _, lr := range g.Group {
			totalBroken++
			linkResult := lr.(URLStatusResponse)
			log.Printf("Broken link: '%v' %v", linkResult.URL, linkResult.Error)
		}
	}

	log.Printf("Total broken: %v", totalBroken)
}

func checkURLs(links []string) CheckURLsResponse {
	request := prepareRequest(links)
	var response CheckURLsResponse
	_, err := resty.New().R().
		SetBody(&request).
		SetResult(&response).
		Post(serviceURL)
	if err != nil {
		panic(err)
	}
	return response
}

func prepareRequest(links []string) CheckURLsRequest {
	urls := make([]URLRequest, 0)
	for _, link := range links {
		if link != "" {
			urls = append(urls, URLRequest{
				URL:     link,
				Context: "",
			})
		} else {
			log.Println("link empty: ", link)
		}
	}

	return CheckURLsRequest{
		Urls: urls,
	}
}

func extractLinks(text string) []string {
	rxStrict := xurls.Strict()
	links := rxStrict.FindAllString(text, -1)
	return links
}

func extractText(url string) string {
	client := resty.New()
	res, err := client.R().
		Get(url)

	if err != nil {
		panic(err)
	}
	text := string(res.Body())
	return text
}

func timeIt(context string, what func()) {
	start := time.Now()

	what()

	elapsed := time.Since(start)
	log.Printf("%v took %s", context, elapsed)
}
