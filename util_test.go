package tender

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
	"net/http"
	"net/url"
)

var _ = Describe("util", func() {
	Describe("parsePort", func() {
		It("returns port of the request", func() {
			testCases := []struct {
				url  string
				host string
				port string
			}{
				{"http://google.com", "google.com", "80"},
				{"https://google.com", "google.com", "443"},
				{"http://google.com:8000", "google.com:8000", "8000"},
			}

			for _, tc := range testCases {
				url, err := url.Parse(tc.url)
				Expect(err).To(BeNil(), "test case should be a valid url")
				p := parsePort(&http.Request{Host: tc.host, URL: url})
				Expect(p).To(Equal(tc.port), "Expect correct port to be returned")
			}
		})
	})

	Describe("hostToIP", func() {
		It("changes the request's host to the specified IP", func() {
			url, _ := url.Parse("http://google.com:8000")
			r := &http.Request{URL: url, Host: "google.com:8000"}
			hostToIP(r, net.ParseIP("10.0.0.2"))
			Expect(r.URL.Host).To(Equal("10.0.0.2:8000"), "Expect the host to be changed to ip:port")
		})
	})

	Describe("getHost", func() {
		It("returns the host's name when domain contains only host", func() {
			var h string
			h, _ = getHost("yammer.nitrousapp.com:65333", ".nitrousapp.com")
			Expect(h).To(Equal("yammer"), "Expect to get correct host's name")

			h, _ = getHost("dropbox.nitrousapp.com", ".nitrousapp.com")
			Expect(h).To(Equal("dropbox"), "Expect to get correct host's name")
		})

		It("returns the host's name when domain contains container name", func() {
			var h string
			h, _ = getHost("hello.yammer.nitrousapp.com:65333", ".nitrousapp.com")
			Expect(h).To(Equal("yammer"), "Expect to get correct host's name")

			h, _ = getHost("hello.dropbox.nitrousapp.com", ".nitrousapp.com")
			Expect(h).To(Equal("dropbox"), "Expect to get correct host's name")
		})
	})
})
