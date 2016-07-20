package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type config struct {
	AccessKey      string `toml:"access_key_id"`
	SecretAcessKey string `toml:"secret_access_key"`
	Buckets        map[string]bucket
}

type bucket struct {
	Region string
	Name   string
	CNAME  bool
}

type objectInfo struct {
	body        io.ReadSeeker
	key         string
	contentType string
}

type result struct {
	url string
	err error
}

var (
	private    bool
	timestamp  bool
	bucketName string
)

func main() {
	flag.BoolVar(&private, "p", false, "private upload")
	flag.BoolVar(&timestamp, "t", false, "add timestamp")
	flag.StringVar(&bucketName, "bucket", "default", "bucket to upload")

	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-p] [-t] [-bucket <bucketName>] <filename>...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	user, err := user.Current()
	if err != nil {
		exit(err)
	}

	var conf config
	credentialsPath := filepath.Join(user.HomeDir, ".aws-credentials.toml")
	if _, err := toml.DecodeFile(credentialsPath, &conf); err != nil {
		exit(fmt.Errorf("invalid config file: %v", err))
	}

	if _, ok := conf.Buckets[bucketName]; !ok {
		exit(fmt.Errorf("missing config for bucket: %v", bucketName))
	}

	awsConfig := aws.NewConfig().WithCredentials(credentials.NewStaticCredentials(conf.AccessKey, conf.SecretAcessKey, "")).WithRegion(conf.Buckets[bucketName].Region)
	svc := s3.New(session.New(awsConfig))

	results := make(chan result, len(files))
	for _, file := range files {
		go uploadFile(file, conf, svc, results)
	}

	for range files {
		res := <-results

		if res.err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed to upload object: %v", res.err))
		} else {
			fmt.Printf("uploaded %s\n", res.url)
		}
	}
	close(results)
}

func uploadFile(file string, conf config, svc *s3.S3, results chan<- result) {
	obj, err := newObjectInfo(file)
	if err != nil {
		results <- result{err: fmt.Errorf("unable to read file: %v", err)}
		return
	}

	bname := conf.Buckets[bucketName].Name
	params := &s3.PutObjectInput{
		Bucket:      &bname,
		Key:         &obj.key,
		Body:        obj.body,
		ContentType: &obj.contentType,
	}

	if !private {
		params.ACL = aws.String("public-read")
	}

	_, err = svc.PutObject(params)

	if err != nil {
		results <- result{err: err}
	} else {
		results <- result{url: buildOutputURL(obj, conf)}
	}
}

func newObjectInfo(s string) (objectInfo, error) {
	_, err := os.Stat(s)

	if err == nil {
		content, err := ioutil.ReadFile(s)

		if err != nil {
			return objectInfo{}, err
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	} else if os.IsNotExist(err) && isURL(s) {
		fmt.Printf("%s is not a local file, will attempt to fetch it as an URL\n", s)

		content, err := fetchRemoteContent(s)

		if err != nil {
			return objectInfo{}, err
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	}

	return objectInfo{}, err
}

func buildKey(s string) string {
	split := strings.Split(filepath.Base(s), ".")

	var key string
	if timestamp {
		key = fmt.Sprintf("%s-%d", split[0], time.Now().Unix())
	} else {
		key = split[0]
	}

	if len(split) == 2 {
		key += fmt.Sprintf(".%s", split[1])
	}

	return key
}

func buildObjectInfo(content []byte, s string) objectInfo {
	return objectInfo{
		body:        bytes.NewReader(content),
		key:         buildKey(s),
		contentType: mime.TypeByExtension(filepath.Ext(s)),
	}
}

func fetchRemoteContent(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not found: %s", url)
	}

	return ioutil.ReadAll(resp.Body)
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func isURL(str string) bool {
	prefixes := []string{"http://", "https://"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}

	return false
}

func buildOutputURL(obj objectInfo, conf config) string {
	var url string
	bucket := conf.Buckets[bucketName]

	if private {
		url = fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s", bucket.Region, bucket.Name, obj.key)
	} else {
		url = buildPublicURL(obj, conf)
	}

	return url
}

func buildPublicURL(obj objectInfo, conf config) string {
	var url string
	bucket := conf.Buckets[bucketName]

	if bucket.CNAME {
		url = fmt.Sprintf("http://%s/%s", bucket.Name, obj.key)
	} else {
		url = fmt.Sprintf("http://%s.s3.amazonaws.com/%s", bucket.Name, obj.key)
	}

	return url
}
