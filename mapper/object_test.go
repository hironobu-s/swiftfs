package mapper

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"time"

	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	TEST_CONTAINER = "object-test-container"
	TEST_OBJECT    = "test-object"
	TEST_DATA      = "testdata"
)

// func testobject() *Object {
// 	path,err := os.Getwd()
// 	if err
// 	return &Object{
// 		Path: path,
// 	}
// }

// func (s *MockSwift) Get(name string) objects.DownloadResult {
// 	str := strings.NewReader("testdata")
// 	r := objects.DownloadResult{
// 		Body: ioutil.NopCloser(str),
// 	}
// 	return r
// }

// type TestTransport struct {
// 	Transport http.RoundTripper
// }

// func (t *TestTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
// 	// resp := t.Transport.RoundTrip(req)
// 	// resp := &http.Response{
// 	// 	Request: req,
// 	// }

// 	log.Infof("Send    ==>: %s %s", req.Method, req.URL)

// 	// resp, err = t.Transport.RoundTrip(req)
// 	// resp, err = http.ReadResponse(bufio.NewReader(strings.NewReader("testdata")), req)
// 	resp = &http.Response{
// 		Status:     "200 OK",
// 		StatusCode: 200,
// 		Proto:      "HTTP/1.0",
// 		ProtoMajor: 1,
// 		ProtoMinor: 0,
// 		Header:     http.Header{},
// 		Body:       ioutil.NopCloser(strings.NewReader("testdata")),
// 		Close:      true,
// 		Trailer:    http.Header{},
// 		Request:    req,
// 	}

// 	log.Infof("Receive <==: %d %s (size=%d)", resp.StatusCode, resp.Request.URL, resp.ContentLength)

// 	return resp, nil
// }

var swift *openstack.Swift

func TestMain(m *testing.M) {
	c := config.NewConfig()
	c.ContainerName = TEST_CONTAINER

	// initialize swift
	swift = openstack.NewSwift(c)
	if err := swift.Auth(); err != nil {
		panic(err)
	}
	swift.DeleteContainer()
	swift.CreateContainer()
	swift.Upload(TEST_OBJECT, strings.NewReader(TEST_DATA))

	m.Run()
}

func TestLocalPath(t *testing.T) {
	path := TEST_OBJECT
	o := &object{
		Path: path,
	}

	if o.Localpath() != "/tmp/swiftfs/"+TEST_OBJECT {
		t.Fatalf("localpath mismatched %s != %s", path, o.Localpath())
	}
}

func TestDownload(t *testing.T) {
	var err error
	path := TEST_OBJECT
	o := &object{
		Path:  path,
		swift: swift,
	}

	err = o.download()
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = os.Stat(o.Localpath())
	if err != nil {
		t.Fatalf("%v", err)
	}

	data, err := ioutil.ReadFile(o.Localpath())
	if err != nil {
		t.Fatalf("%v", err)
	} else if string(data) != TEST_DATA {
		t.Fatalf("File data does not match TEST_DATA")
	}
}

// func TestOpen(t *testing.T) {
// 	path := TEST_OBJECT
// 	o := &object{
// 		Path: path,
// 	}

// 	file, err := o.Open(os.O_RDWR, 0600)
// 	if err != nil {
// 		t.Errorf("%v", err)

// 	} else if file.Name() != o.Localpath() {
// 		t.Errorf("localpath mismatched %s != %s", path, o.Localpath())
// 	}
// }

func TestFlush(t *testing.T) {
	var err error
	path := TEST_OBJECT
	o := &object{
		Path: path,
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")
	tt := time.Date(2015, 1, 1, 13, 0, 0, 0, loc)
	if err = os.Chtimes(o.Localpath(), tt, tt); err != nil {
		t.Fatalf("%v", err)
	}

	// o.Mtime will syncronize with mtime of localpath
	o.Flush()

	if !tt.Equal(o.Mtime) {
		t.Errorf("mtime mismatched %s != %s", tt.String(), o.Mtime.String())
	}
}

func TestUpload(t *testing.T) {
	var err error
	path := TEST_OBJECT + "2"
	o := &object{
		Path:  path,
		swift: swift,
	}

	file, err := o.Open(os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer file.Close()

	if _, err = file.WriteString(TEST_DATA); err != nil {
		t.Fatalf("%v", err)
	}

	if err = o.Upload(); err != nil {
		t.Fatalf("%v", err)
	}

	result := swift.Get(path)
	data, _ := ioutil.ReadAll(result.Body)
	if string(data) != TEST_DATA {
		t.Fatalf("File data does not match TEST_DATA")
	}
}
