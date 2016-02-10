package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
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
	Region         string
	Bucket         string
}

type objectInfo struct {
	body        io.ReadSeeker
	key         string
	contentType string
}

func main() {
	if len(os.Args) != 2 {
		exit(fmt.Errorf("usage: %s filename", os.Args[0]))
	}

	user, err := user.Current()
	if err != nil {
		exit(err)
	}

	var conf config
	credentialsPath := user.HomeDir + "/.aws-credentials.toml"
	if _, err := toml.DecodeFile(credentialsPath, &conf); err != nil {
		exit(fmt.Errorf("invalid config file: %v", err))
	}

	awsConfig := aws.NewConfig().WithCredentials(credentials.NewStaticCredentials(conf.AccessKey, conf.SecretAcessKey, "")).WithRegion(conf.Region)
	svc := s3.New(session.New(awsConfig))

	obj, err := newObjectInfo(os.Args[1])
	if err != nil {
		exit(fmt.Errorf("unable to get contents for file: %v", err))
	}

	params := &s3.PutObjectInput{
		Bucket:      &conf.Bucket,
		Key:         &obj.key,
		Body:        obj.body,
		ContentType: &obj.contentType,
		ACL:         aws.String("public-read"),
	}

	_, err = svc.PutObject(params)
	if err != nil {
		exit(fmt.Errorf("failed to upload object: %v", err))
	}

	url := fmt.Sprintf("http://%s/%s", conf.Bucket, obj.key)
	fmt.Printf("uploaded %s to %s\n", url, conf.Bucket)
}

func newObjectInfo(s string) (objectInfo, error) {
	_, err := os.Stat(s)

	if err == nil {
		content, err := fetchLocalContent(s)

		if err != nil {
			exit(err)
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	} else if os.IsNotExist(err) {
		fmt.Printf("%s is not a local file, will attempt to fetch it as an URL\n", s)

		content, err := fetchRemoteContent(s)

		if err != nil {
			exit(err)
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	}

	return objectInfo{}, err
}

func buildKey(s string) string {
	return fmt.Sprintf("%d%s", time.Now().Unix(), filepath.Base(s))
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
	defer resp.Body.Close()

	if err != nil {
		exit(fmt.Errorf("failed to get: %s", url))
	}

	return ioutil.ReadAll(resp.Body)
}

func fetchLocalContent(fpath string) ([]byte, error) {
	f, err := os.Open(fpath)
	defer f.Close()

	if err != nil {
		exit(err)
	}

	return ioutil.ReadAll(f)
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
