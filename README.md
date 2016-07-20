# s3-upload

This is a simple tool to upload files to Amazon S3.

## Installation

```
go get -u github.com/carlosgaldino/s3-upload
```

Make sure `$GOPATH/bin` is in your `$PATH`.

## Usage

```
Usage: s3-upload [-p] [-t] [-path <pathPrefix>] [-bucket <bucketName>] <filename>...
  -bucket string
        bucket to upload (default "default")
  -p	private upload
  -path string
        path prefix
  -t	add timestamp
```

`filename` can be a local file or an URL. And you can pass multiple filenames as
well.

You also need to have a `~/.aws-credentials.toml` file with the following
structure:

```toml
access_key_id = "ACCESS_KEY_ID"
secret_access_key = "SECRET_ACCESS_KEY"

[buckets]
    [buckets.default]
        bucket = "my-bucket"
        region = "us-east-1" # Of course the region may be different.
        cname = true # If omitted or `false` the URL won't be customized.

    [buckets.gifs]
        bucket = "my-gifs-bucket"
        region = "us-east-1"
```

For more information about CNAME customization take a look at: http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html#VirtualHostingCustomURLs
