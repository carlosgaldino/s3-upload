# s3-upload

This is a simple tool to upload files to Amazon S3.

## Installation

```
go get -u github.com/carlosgaldino/s3-upload
```

Make sure `$GOPATH/bin` is in your `$PATH`.

## Usage

```
Usage: s3-upload [-p] <filename>
  -p	private upload
```

`filename` can be a local file or an URL.

You also need to have a `~/.aws-credentials.toml` file with the following
structure:

```toml
access_key_id = "ACCESS_KEY_ID"
secret_access_key = "SECRET_ACCESS_KEY"
bucket = "my-bucket"
region = "us-east-1"
```
