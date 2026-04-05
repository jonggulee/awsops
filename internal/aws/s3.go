package aws

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Bucket struct {
	Profile          string
	Name             string
	Region           string
	CreationDate     time.Time
	VersioningStatus string // "Enabled" | "Suspended" | "Disabled"
	PublicAccess     string // "Blocked" | "Public" | "Unknown"
	Tags             map[string]string
}

func (b S3Bucket) CreationDateStr() string {
	if b.CreationDate.IsZero() {
		return "-"
	}
	return b.CreationDate.Local().Format("2006-01-02")
}

// FetchAllS3Buckets fetches S3 buckets for all profiles.
// S3 is a global service so we use a single fixed region per profile.
func FetchAllS3Buckets(ctx context.Context) ([]S3Bucket, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type result struct {
		buckets []S3Bucket
		errs    []error
	}

	ch := make(chan result, len(profiles))
	for _, p := range profiles {
		p := p
		go func() {
			buckets, errs := fetchS3BucketsForProfile(ctx, p)
			ch <- result{buckets, errs}
		}()
	}

	var all []S3Bucket
	var allErrs []error
	for range profiles {
		r := <-ch
		all = append(all, r.buckets...)
		allErrs = append(allErrs, r.errs...)
	}
	return all, allErrs
}

func fetchS3BucketsForProfile(ctx context.Context, profile string) ([]S3Bucket, []error) {
	client, err := NewProfileClient(ctx, profile, DefaultRegion)
	if err != nil {
		return nil, []error{err}
	}

	out, err := client.S3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, []error{err}
	}

	type bucketResult struct {
		bucket S3Bucket
		err    error
	}

	ch := make(chan bucketResult, len(out.Buckets))
	for _, b := range out.Buckets {
		b := b
		go func() {
			bucket, err := enrichBucket(ctx, client, profile, b)
			ch <- bucketResult{bucket, err}
		}()
	}

	var buckets []S3Bucket
	var errs []error
	for range out.Buckets {
		r := <-ch
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			buckets = append(buckets, r.bucket)
		}
	}
	return buckets, errs
}

// enrichBucket fetches region, versioning, and public access block concurrently.
func enrichBucket(ctx context.Context, client *ProfileClient, profile string, b s3types.Bucket) (S3Bucket, error) {
	bucket := S3Bucket{
		Profile:      profile,
		Name:         aws.ToString(b.Name),
		CreationDate: aws.ToTime(b.CreationDate),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	// 1. Region
	go func() {
		defer wg.Done()
		loc, err := client.S3.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: b.Name,
		})
		if err == nil {
			region := string(loc.LocationConstraint)
			if region == "" {
				region = "us-east-1" // 빈 값은 us-east-1 의미
			}
			mu.Lock()
			bucket.Region = region
			mu.Unlock()
		}
	}()

	// 2. Versioning
	go func() {
		defer wg.Done()
		ver, err := client.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: b.Name,
		})
		mu.Lock()
		if err != nil {
			bucket.VersioningStatus = "Unknown"
		} else {
			switch ver.Status {
			case s3types.BucketVersioningStatusEnabled:
				bucket.VersioningStatus = "Enabled"
			case s3types.BucketVersioningStatusSuspended:
				bucket.VersioningStatus = "Suspended"
			default:
				bucket.VersioningStatus = "Disabled"
			}
		}
		mu.Unlock()
	}()

	// 3. Public Access Block
	go func() {
		defer wg.Done()
		pub, err := client.S3.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
			Bucket: b.Name,
		})
		mu.Lock()
		if err != nil {
			// 설정이 없으면 퍼블릭 접근 허용 상태
			bucket.PublicAccess = "Public"
		} else {
			cfg := pub.PublicAccessBlockConfiguration
			if cfg != nil &&
				aws.ToBool(cfg.BlockPublicAcls) &&
				aws.ToBool(cfg.BlockPublicPolicy) &&
				aws.ToBool(cfg.IgnorePublicAcls) &&
				aws.ToBool(cfg.RestrictPublicBuckets) {
				bucket.PublicAccess = "Blocked"
			} else {
				bucket.PublicAccess = "Public"
			}
		}
		mu.Unlock()
	}()

	wg.Wait()
	return bucket, nil
}

// FetchS3BucketTags fetches tags for a single bucket. Used for lazy loading in detail view.
func FetchS3BucketTags(ctx context.Context, profile, bucketName string) (map[string]string, error) {
	client, err := NewProfileClient(ctx, profile, DefaultRegion)
	if err != nil {
		return nil, err
	}
	out, err := client.S3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return map[string]string{}, nil // 태그 없음으로 처리
	}
	tags := make(map[string]string, len(out.TagSet))
	for _, t := range out.TagSet {
		tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return tags, nil
}
