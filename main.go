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
		fmt.Fprintf(os.Stderr, "usage: %s filename\n", os.Args[0])
		os.Exit(1)
	}

	user, err := user.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var conf config
	credentialsPath := user.HomeDir + "/.aws-credentials.toml"
	if _, err := toml.DecodeFile(credentialsPath, &conf); err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding config file: %v\n", err)
		os.Exit(1)
	}

	awsConfig := aws.NewConfig().WithCredentials(credentials.NewStaticCredentials(conf.AccessKey, conf.SecretAcessKey, "")).WithRegion(conf.Region)
	svc := s3.New(session.New(awsConfig))

	obj, err := newObjectInfo(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while opening file: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "failed to upload object: %v\n", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s/%s", conf.Bucket, obj.key)
	fmt.Printf("uploaded %s to %s\n", url, conf.Bucket)
}

func newObjectInfo(s string) (*objectInfo, error) {
	_, err := os.Stat(s)

	if os.IsNotExist(err) {
		fmt.Printf("%s is not a file, will attempt to fetch it as an URL\n", s)

		resp, err := http.Get(s)
		defer resp.Body.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get: %s", s)
			os.Exit(1)
		}

		content, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	} else if err == nil {
		f, err := os.Open(s)
		defer f.Close()

		content, err := ioutil.ReadAll(f)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		obj := buildObjectInfo(content, s)

		return obj, nil
	}

	return nil, err
}

func buildKey(s string) string {
	return fmt.Sprintf("%d%s", time.Now().Unix(), filepath.Base(s))
}

func buildObjectInfo(content []byte, s string) *objectInfo {
	return &objectInfo{
		body:        bytes.NewReader(content),
		key:         buildKey(s),
		contentType: mime.TypeByExtension(filepath.Ext(s)),
	}
}
