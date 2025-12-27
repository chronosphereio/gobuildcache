# Go Build Cache Server

TODO: Why not S3 (too slow)
TODO: Clear command
TODO: Assumes ephemeral storage
TODO: Link depot blog post
TODO: Lifecycle policy

`gobuildcache` implements the [gocacheprog](TODO: LINK) interface defined by the Go compiler over a variety of storage backends, the most important of which is S3 Express One Zone (henceforth referred to as S3OZ). Its primary purpose is to accelerate CI (both compilation and tests) for large Go repositories.

Effectively, `gobuildcache` leverages S3OZ as a distributed build cache for concurrent `go build` or `go test` processes regardless of whether they're running on a single machine or distributed across a fleet of CI VMs. This dramatically improves the performance of CI for large Go repositories because every CI process will behave as if it is running with an almost completely pre-populated build cache, even if the CI process was started on a completely ephemeral VM that has never compiled code or executed tests for the repository before.

This is similar in spirit to the common pattern of restoring a shared go build cache at the beginning of the CI run, and then saving the freshly updated go build cache at the beginning of the CI run so it can be restored by subsequent CI jobs. However, the approach taken by `gobuildcache` is much more efficient resulting in dramatically lower CI times (and bills) with significantly less "CI engineering" required. For more details on why the approach taken by `gobuildcache` is better, see the "Why Should I Use gobuildcache" section.

# Quick Start

## Installation

```bash
go install github.com/richardartoul/gobuildcache@latest
```

## Usage

```bash
export GOCACHEPROG=gobuildcache
go build ./...
go test ./...
```

By default, `gobuildcache` uses an on-disk cache stored in your operating system's default temporary directory. This is useful testing and experimentation with `gobuildcache`, but provides no benefits over the Go compiler's built in cache which also stores cached data on locally on disk.

For "production" use-cases in CI, you'll want to configure `gobuildcache` to use S3 Express One Zone, or a similarly low latency distributed backend.

```bash
export BACKEND_TYPE=s3
export S3_BUCKET=$BUCKET_NAME
```

You'll also have to provide AWS credentials. `gobuildcaceh` embeds the AWS V2 S3 SDK so any method of providing credentials to that library will work, but the simplest is to us environment variables as demonstrated below.

```bash
export GOCACHEPROG=gobuildcache
export BACKEND_TYPE=s3
export S3_BUCKET=$BUCKET_NAME
export AWS_REGION=$BUCKE_REGION
export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
go build ./...
go test ./...
```

Your credentials must have the following permissions:

TODO: S3express permission
TODO: Confirm these

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:HeadBucket",
        "s3:HeadObject"
      ],
      "Resource": [
        "arn:aws:s3:::$BUCKET_NAME",
        "arn:aws:s3:::$BUCKET_NAME/*"
      ]
    }
  ]
}
```

In normal circumstances you should never have to run the `gobuildcache` binary directly, it will be instead be invoked by the go compiler (hence why configuration is managed via environment variables instead of command line flags). However, the `gobuildcache` binary ships with a `clear` command that can be used to 

# Configuration

`gobuildcache` ships with reasonable defaults, but this section provides a complete overview of flags / environment variables that can be used to override behavior.

# Why should I use gobuildcache?

First, the local on-disk cache of the CI VM doesn't have to be pre-populated at once. `gobuildcache` populates it by loading the cache on the fly as the Go compiler compiles code and runs test. This makes it so you don't have to waste several precious minutes of CI time waiting for gigabytes of data to be downloaded and decompressed while CI cores sit idle. This is why S3OZ's low latency is crucial to `gobuildcache`'s design.

Second, `gobuildcache` is never stale. A big problem with the common caching pattern described above is that if the P.R under test differs "significantly" from the main branch (say, because a package that many other packages depend on has been modified) then the Go toolchain will be required to compile almost every file from scratch, as well as run almost every test in the repo. Contrast that with the `gobuildcache` approach where the first commit that is pushed will incur the penalty described above, but all subsequent commits will experience extremely high cache hit ratios. One way to think about this benefit is that with the common approach, only one "branch" of the repository can be cached at any given time (usually the `main` branch), and as a result all P.Rs experience CI delays that are roughly proportional to how much they "differ" from `main`. With the `gobuildcache` approach, the cache stored in S3OZ can store a hot cache for all of the different branches and PRs in the repository at the same time. This makes cache misses significantly less likely, and reduces average CI times dramatically.

Third, the `gobuildcache` approach completely obviates the need to determine how frequently to "rebuild" the shared cache tarball. This is important, because rebuilding the shared cache is expensive as it usually has to be built from a CI process running with no pre-built cache to avoid infinite cache bloat, but if its run too infrequently then CI for PRs will be slow (because they "differ" too much from the stale cached tarball).

Fourth, `gobuildcache` makes parallelizing CI using commonly supported "matrix" strategies much easier and efficient. For example, consider the common pattern where unit tests are split across 4 concurrent CI jobs using Github actions matrix functionality. In this approach, each CI job runs ~ 1/4th of the unit tests in the repostitory and each CI job determines which tests its responsible for running by hashing the unit tests name and then moduloing it by the index assigned to the CI job by Github actions matrix functionality. This works great for parallelizing test execution across multiple VMs, but it presesents a huge problem for build caching. The Go build cache doesn't just cache package compilation, it also cache test execution. This is a hugely important optimization for CI because it means that if you can populate the the CI job's build cache efficiently, P.Rs that modify packages that not many other packages depend on will only have to run a small fraction of the total tests in the repository. However, generating this cache is difficult now because each CI job is only executing a fraction of the test suite, so the build cache generated by CI job 1 will result in 0 cache hits for job 2 and vice versa. As a result, CI job matrix unit now has to restore and save a build cache that is unique to its specific matrix index. This is doable, but it's annoying and requires solving a bunch of other incidental engineering challenges like making sure the cache is only ever saved from CI jobs running on the main branch, and using consistent hashing instead of modulo hashing to assign tests to CI job matrix units (because otherwise add a single test will completely shuffle the assignment of tests to CI jobs and the cache hit ratio will be terrible). All of these problems just dissapear when using the `gobuildcache` because the CI jobs behave much more like stateless, ephemeral compute while still benefitting from extremely high cache hit ratios due to the shared / distributed cache backend.

## TODOs

1. Actually use HandleRequestWithRetries instead of HandleRequest.
2. Move async_backend.go to backends package

## Features

- **Multiple Storage Backends**: Choose between local disk storage or S3 cloud storage
- **Go Build Cache Protocol**: Compatible with Go's remote cache protocol (`GOCACHEPROG`)
- **Flexible Configuration**: Use command-line flags or environment variables (or both!)
- **Debug Mode**: Optional debug logging for troubleshooting

## Configuration

All configuration options can be set via **command-line flags** or **environment variables**. Command-line flags take precedence over environment variables.

### Available Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-backend` | `BACKEND_TYPE` | `disk` | Backend type: `disk` or `s3` |
| `-cache-dir` | `CACHE_DIR` | `/tmp/gobuildcache` | Cache directory for disk backend |
| `-s3-bucket` | `S3_BUCKET` | (none) | S3 bucket name (required for S3) |
| `-s3-prefix` | `S3_PREFIX` | (empty) | S3 key prefix |
| `-debug` | `DEBUG` | `false` | Enable debug logging |
| `-stats` | `PRINT_STATS` | `false` | Print cache statistics on exit |

## Storage Backends

### Disk Backend (Default)

Stores cache files on the local filesystem.

**Using Flags:**
```bash
gobuildcache -cache-dir=/path/to/cache
```

**Using Environment Variables:**
```bash
export CACHE_DIR=/path/to/cache
gobuildcache
```

**Mixed (flags override env vars):**
```bash
export CACHE_DIR=/default/path
gobuildcache -cache-dir=/override/path -debug
```

### S3 Backend

Stores cache files in Amazon S3 (or S3-compatible storage).

**Using Flags:**
```bash
gobuildcache -backend=s3 -s3-bucket=my-bucket
```

**Using Environment Variables:**
```bash
export BACKEND_TYPE=s3
export S3_BUCKET=my-bucket
export S3_PREFIX=cache/
gobuildcache
```

**Mixed (flags override env vars):**
```bash
export BACKEND_TYPE=s3
export S3_BUCKET=default-bucket
gobuildcache -s3-bucket=override-bucket -debug
```

**AWS Credentials:**
AWS credentials are always configured via standard AWS environment variables or `~/.aws/credentials`:
```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
# Or use AWS profiles:
export AWS_PROFILE=your-profile
```

**How S3 Backend Works:**
1. Cache objects are stored in S3 with metadata (outputID, size, timestamp)
2. On cache hits, files are downloaded to a local temp directory for Go to access
3. The local temp directory acts as a secondary cache to avoid repeated S3 downloads
4. The `diskPath` returned to Go points to the locally cached file

## Usage

### Getting Help

View available commands and flags:
```bash
gobuildcache help
gobuildcache -h
gobuildcache clear -h
```

### Running the Server

Start the cache server:
```bash
# With disk backend (default)
gobuildcache

# With custom cache directory
gobuildcache -cache-dir=/var/cache/go

# With S3 backend
gobuildcache -backend=s3 -s3-bucket=my-cache-bucket

# With debug logging
gobuildcache -debug

# With statistics (prints stats on exit)
gobuildcache -stats

# Both debug and statistics
gobuildcache -debug -stats
```

The server will:
1. Read from stdin (Go sends cache requests)
2. Write responses to stdout
3. Log debug information to stderr (if `-debug` flag is set)
4. Print cache statistics to stderr on exit (if `-stats` flag is set)

### Configuring Go to Use the Cache

Set the `GOCACHEPROG` environment variable to point to the cache server:

```bash
export GOCACHEPROG=/path/to/gobuildcache/builds/gobuildcache
go build ./...
```

### Clearing the Cache

Clear all cache entries:
```bash
# Clear disk cache
gobuildcache clear -cache-dir=/var/cache/go

# Clear S3 cache
gobuildcache clear -backend=s3 -s3-bucket=my-cache-bucket

# Clear with debug logging
gobuildcache clear -debug
```

The clear command uses the same backend flags as the server command.

## Building

Build the cache server:
```bash
make build
```

Or manually:
```bash
go build -o builds/gobuildcache
```

## Testing

Run all tests:
```bash
go test ./...
```

Run with race detector:
```bash
go test -race ./...
```

### S3 Integration Tests

To run S3 integration tests, set AWS credentials and bucket name as environment variables:

```bash
# Set credentials
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
export TEST_S3_BUCKET=your-bucket-name

# Run S3 integration tests
go test -v -run TestCacheIntegrationS3 -timeout 5m

# Or in one line
AWS_ACCESS_KEY_ID=xxx AWS_SECRET_ACCESS_KEY=yyy AWS_REGION=us-east-1 TEST_S3_BUCKET=my-bucket go test -v -run TestCacheIntegrationS3 -timeout 5m
```

**Note:** S3 tests will fail if credentials or bucket name are not set. To skip S3 tests, use the `-short` flag:

```bash
go test -short ./...  # Skips S3 tests
```

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Architecture

### CacheBackend Interface

All backends implement the `CacheBackend` interface:

```go
type CacheBackend interface {
    // Put stores an object in the cache
    Put(actionID, outputID []byte, body io.Reader, bodySize int64) (diskPath string, err error)
    
    // Get retrieves an object from the cache
    Get(actionID []byte) (outputID []byte, diskPath string, size int64, putTime *time.Time, miss bool, err error)
    
    // Close performs cleanup operations
    Close() error
    
    // Clear removes all cache entries
    Clear() error
}
```

### Available Backends

- **DiskBackend** (`disk_backend.go`): Local filesystem storage
- **S3Backend** (`s3_backend.go`): AWS S3 storage with local caching

## AWS Configuration

The S3 backend uses the AWS SDK for Go v2 and supports all standard AWS credential sources:

1. **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
2. **Shared Credentials File**: `~/.aws/credentials`
3. **IAM Roles**: For EC2 instances, ECS tasks, Lambda functions
4. **SSO**: AWS IAM Identity Center (formerly AWS SSO)

### Required IAM Permissions

The IAM user/role needs the following S3 permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:HeadBucket",
        "s3:HeadObject"
      ],
      "Resource": [
        "arn:aws:s3:::your-bucket-name",
        "arn:aws:s3:::your-bucket-name/*"
      ]
    }
  ]
}
```

## Examples

### Local Development with Disk Backend

```bash
# Using flags
gobuildcache -debug -cache-dir=/tmp/my-go-cache

# Or using environment variables
export DEBUG=true
export CACHE_DIR=/tmp/my-go-cache
gobuildcache
```

### Team Build Cache with S3

```bash
# Option 1: Using environment variables (easier for team consistency)
export BACKEND_TYPE=s3
export S3_BUCKET=team-build-cache
export S3_PREFIX=go/
export AWS_REGION=us-east-1
export GOCACHEPROG=/usr/local/bin/gobuildcache
go build ./...

# Option 2: Using flags
gobuildcache -backend=s3 -s3-bucket=team-build-cache -s3-prefix=go/
```

### CI/CD Pipeline with S3

```yaml
# Example GitHub Actions workflow
env:
  # Use environment variables for cleaner configuration
  BACKEND_TYPE: s3
  S3_BUCKET: ci-build-cache
  S3_PREFIX: ${{ github.repository }}/
  AWS_REGION: us-east-1
  GOCACHEPROG: ./gobuildcache

steps:
  - name: Download cache server
    run: |
      curl -L -o gobuildcache https://example.com/gobuildcache
      chmod +x gobuildcache
  
  - name: Configure AWS credentials
    uses: aws-actions/configure-aws-credentials@v4
    with:
      role-to-assume: arn:aws:iam::123456789012:role/GithubActionsRole
      aws-region: us-east-1
  
  - name: Build with remote cache
    run: go build ./...
```

**Alternative using flags:**
```yaml
steps:
  # ... (download and AWS setup same as above)
  
  - name: Start cache server
    run: |
      ./gobuildcache -backend=s3 \
        -s3-bucket=ci-build-cache \
        -s3-prefix=${{ github.repository }}/ &
  
  - name: Build with remote cache
    run: go build ./...
```

## Troubleshooting

### Enable Debug Logging

Using flags:
```bash
gobuildcache -debug
```

Using environment variable:
```bash
export DEBUG=true
gobuildcache
```

### Check S3 Connectivity

Using flags:
```bash
gobuildcache clear -backend=s3 -s3-bucket=your-bucket -debug
```

Using environment variables:
```bash
export BACKEND_TYPE=s3
export S3_BUCKET=your-bucket
export DEBUG=true
gobuildcache clear
```

### Verify Cache is Being Used

Look for cache hits in your Go build output:
```bash
go build -x ./...  # Shows detailed build steps including cache usage
```

## Performance Considerations

### Disk Backend
- **Pros**: Very fast, no network latency
- **Cons**: Not shared across machines, limited by disk space

### S3 Backend
- **Pros**: Shared across team/CI, scalable, durable
- **Cons**: Network latency for downloads, S3 API costs
- **Optimization**: Local temp cache reduces repeated S3 downloads

## License

MIT License (or your chosen license)

