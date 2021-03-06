package app_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/noisy-neighbor-nozzle/pkg/collector"

	"code.cloudfoundry.org/cli/plugin/models"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/noisy-neighbor-nozzle/cmd/cli-plugin/app"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogNoise", func() {
	var (
		logger       *stubLogger
		tableWriter  *bytes.Buffer
		cli          *stubCliConnection
		httpClient   *stubHTTPClient
		appInfoStore *stubAppInfoStore
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		tableWriter = bytes.NewBuffer(nil)
		cli = newStubCliConnection()
		httpClient = newStubHTTPClient(`{
			"timestamp":1517855100,
			"counts":{
			   "app-guid-0/0":100,
			   "app-guid-1/0":200,
			   "app-guid-1/1":1234567890,
			   "app-guid-2/0":300,
			   "app-guid-3/0":400,
			   "app-guid-4/0":500,
			   "app-guid-5/0":600,
			   "app-guid-6/0":700,
			   "app-guid-7/0":800,
			   "app-guid-8/0":900,
			   "app-guid-9/0":1000
			}
		 }`)
		appInfoStore = newStubAppInfoStore(map[collector.AppGUID]collector.AppInfo{
			collector.AppGUID("app-guid-1"): collector.AppInfo{
				Name:  "name-1",
				Space: "space-1",
				Org:   "org-1",
			},
			collector.AppGUID("app-guid-2"): collector.AppInfo{
				Name:  "name-2",
				Space: "space-2",
				Org:   "org-2",
			},
			collector.AppGUID("app-guid-3"): collector.AppInfo{
				Name:  "name-3",
				Space: "space-3",
				Org:   "org-3",
			},
			collector.AppGUID("app-guid-4"): collector.AppInfo{
				Name:  "name-4",
				Space: "space-4",
				Org:   "org-4",
			},
			collector.AppGUID("app-guid-5"): collector.AppInfo{
				Name:  "name-5",
				Space: "space-5",
				Org:   "org-5",
			},
			collector.AppGUID("app-guid-6"): collector.AppInfo{
				Name:  "name-6",
				Space: "space-6",
				Org:   "org-6",
			},
			collector.AppGUID("app-guid-7"): collector.AppInfo{
				Name:  "name-7",
				Space: "space-7",
				Org:   "org-7",
			},
			collector.AppGUID("app-guid-8"): collector.AppInfo{
				Name:  "name-8",
				Space: "space-8",
				Org:   "org-8",
			},
			collector.AppGUID("app-guid-9"): collector.AppInfo{
				Name:  "name-9",
				Space: "space-9",
				Org:   "org-9",
			},
		})
	})

	It("calls the provided accumulator app", func() {
		app.LogNoise(
			cli,
			[]string{"accumulator"},
			httpClient,
			appInfoStore,
			tableWriter,
			logger,
		)

		Expect(cli.requestedAppName).To(Equal("accumulator"))
		url := `https:\/\/nn-accumulator\.localhost\/rates\/(\d+)\?truncate_timestamp=true`
		Expect(httpClient.requestURL).To(MatchRegexp(url))
		Expect(httpClient.requestHeaders.Get("Authorization")).To(
			Equal("my-token"),
		)

		urlRegexp := regexp.MustCompile(url)
		parts := urlRegexp.FindStringSubmatch(httpClient.requestURL)
		ts, err := strconv.ParseInt(parts[1], 10, 64)
		Expect(err).ToNot(HaveOccurred())
		Expect(ts).To(BeNumerically("~", time.Now().Add(-30*time.Second).Unix(), 1))

		Expect(tableWriter.String()).To(Equal("\x1b[91;0mVolume Last Minute\x1b[0m  App Instance\n" +
			"\x1b[91;1m1,234,567,890\x1b[0m       org-1.space-1.name-1/1\n" +
			"\x1b[91;0m1,000\x1b[0m               org-9.space-9.name-9/0\n" +
			"\x1b[91;0m900\x1b[0m                 org-8.space-8.name-8/0\n" +
			"\x1b[91;0m800\x1b[0m                 org-7.space-7.name-7/0\n" +
			"\x1b[91;0m700\x1b[0m                 org-6.space-6.name-6/0\n" +
			"\x1b[91;0m600\x1b[0m                 org-5.space-5.name-5/0\n" +
			"\x1b[91;0m500\x1b[0m                 org-4.space-4.name-4/0\n" +
			"\x1b[91;0m400\x1b[0m                 org-3.space-3.name-3/0\n" +
			"\x1b[91;0m300\x1b[0m                 org-2.space-2.name-2/0\n" +
			"\x1b[91;0m200\x1b[0m                 org-1.space-1.name-1/0\n",
		))
	})

	It("reports a single log source with the app guid when app info lookup fails", func() {
		httpClient = newStubHTTPClient(`{
			"timestamp":1517855100,
			"counts":{
			   "app-guid-0/0":100
			}
		}`)
		appInfoStore.lookupError = errors.New("look up error")

		app.LogNoise(
			cli,
			[]string{"accumulator"},
			httpClient,
			appInfoStore,
			tableWriter,
			logger,
		)

		Expect(logger.printfMessages).To(ContainElement(
			"look up error",
		))
		println(tableWriter.String())
		Expect(tableWriter.String()).To(Equal("\x1b[91;0mVolume Last Minute\x1b[0m  App Instance\n" +
			"\x1b[91;0m100\x1b[0m                 app-guid-0/0\n",
		))
	})

	It("defaults the app name to nn-accumulator", func() {
		app.LogNoise(
			cli,
			[]string{},
			httpClient,
			appInfoStore,
			tableWriter,
			logger,
		)

		Expect(cli.requestedAppName).To(Equal("nn-accumulator"))
	})

	It("fatally logs if too many args are given", func() {
		Expect(func() {
			app.LogNoise(
				cli,
				[]string{"one", "two"},
				httpClient,
				appInfoStore,
				tableWriter,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid number of arguments, expected 0 or 1, got 2"))
	})

	It("fatally logs if an error occurs while getting the accumulator app", func() {
		cli.getAppError = errors.New("an error")

		Expect(func() {
			app.LogNoise(
				cli,
				[]string{"unknown-app"},
				httpClient,
				appInfoStore,
				tableWriter,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})

	It("fatally logs if getting an auth token fails", func() {
		cli.accessTokenError = errors.New("failed")

		Expect(func() {
			app.LogNoise(
				cli,
				[]string{"nn-accumulator"},
				httpClient,
				appInfoStore,
				tableWriter,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("failed"))
	})

	It("fatally logs if the request to the accumulator fails", func() {
		httpClient.responseErr = errors.New("some error")

		Expect(func() {
			app.LogNoise(
				cli,
				[]string{"nn-accumulator"},
				httpClient,
				appInfoStore,
				tableWriter,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("some error"))
	})

	It("fatally logs if the accumulator does not return a 200", func() {
		httpClient.responseCode = http.StatusBadRequest

		Expect(func() {
			app.LogNoise(
				cli,
				[]string{"nn-accumulator"},
				httpClient,
				appInfoStore,
				tableWriter,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Failed to get rates from accumulator, expected 200, got 400."))
	})
})

type stubLogger struct {
	fatalfMessage  string
	printfMessages []string
}

func (l *stubLogger) Fatalf(format string, args ...interface{}) {
	l.fatalfMessage = fmt.Sprintf(format, args...)
	panic(l.fatalfMessage)
}

func (l *stubLogger) Printf(format string, args ...interface{}) {
	l.printfMessages = append(l.printfMessages, fmt.Sprintf(format, args...))
}

type stubCliConnection struct {
	plugin.CliConnection

	accessToken      string
	accessTokenError error

	requestedAppName string
	getAppError      error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		accessToken: "my-token",
	}
}

func (c *stubCliConnection) GetApp(name string) (plugin_models.GetAppModel, error) {
	c.requestedAppName = name

	return plugin_models.GetAppModel{
		Routes: []plugin_models.GetApp_RouteSummary{{
			Host: "nn-accumulator",
			Domain: plugin_models.GetApp_DomainFields{
				Name: "localhost",
			},
			Path: "/",
		}},
	}, c.getAppError
}

func (c *stubCliConnection) AccessToken() (string, error) {
	return c.accessToken, c.accessTokenError
}

type stubHTTPClient struct {
	responseCount int
	responseBody  string
	responseCode  int
	responseErr   error

	requestURL     string
	requestHeaders http.Header
}

func newStubHTTPClient(payload string) *stubHTTPClient {
	return &stubHTTPClient{
		responseCode: http.StatusOK,
		responseBody: payload,
	}
}

func (s *stubHTTPClient) Do(r *http.Request) (*http.Response, error) {
	s.requestURL = r.URL.String()
	s.requestHeaders = r.Header

	resp := &http.Response{
		StatusCode: s.responseCode,
		Body: ioutil.NopCloser(
			strings.NewReader(s.responseBody),
		),
	}

	return resp, s.responseErr
}

type stubAppInfoStore struct {
	lookupGUIDs []string
	lookupError error
	lookupInfo  map[collector.AppGUID]collector.AppInfo
}

func newStubAppInfoStore(lookupInfo map[collector.AppGUID]collector.AppInfo) *stubAppInfoStore {
	return &stubAppInfoStore{
		lookupInfo: lookupInfo,
	}
}

func (s *stubAppInfoStore) Lookup(
	guids []string,
) (map[collector.AppGUID]collector.AppInfo, error) {
	s.lookupGUIDs = guids

	return s.lookupInfo, s.lookupError
}
