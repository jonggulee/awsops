package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const DefaultRegion = "ap-northeast-2"

type ProfileClient struct {
	Profile      string
	Region       string
	EC2          *ec2.Client
	STS          *sts.Client
	ACM          *acm.Client
	EKS          *eks.Client
	Route53      *route53.Client
	ELBv2        *elbv2.Client
	RDS          *rds.Client
	S3           *s3.Client
	ElastiCache  *elasticache.Client
}

func LoadProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	f, err := os.Open(filepath.Join(home, ".aws", "config"))
	if err != nil {
		return nil, fmt.Errorf("failed to open ~/.aws/config: %w", err)
	}
	defer f.Close()

	var profiles []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[default]" {
			profiles = append(profiles, "default")
		} else if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			name := strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			profiles = append(profiles, name)
		}
	}
	return profiles, scanner.Err()
}

func NewProfileClient(ctx context.Context, profile, region string) (*ProfileClient, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if profile != "default" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile [%s]: %w", profile, err)
	}

	return &ProfileClient{
		Profile:     profile,
		Region:      region,
		EC2:         ec2.NewFromConfig(cfg),
		STS:         sts.NewFromConfig(cfg),
		ACM:         acm.NewFromConfig(cfg),
		EKS:         eks.NewFromConfig(cfg),
		Route53:     route53.NewFromConfig(cfg),
		ELBv2:       elbv2.NewFromConfig(cfg),
		RDS:         rds.NewFromConfig(cfg),
		S3:          s3.NewFromConfig(cfg),
		ElastiCache: elasticache.NewFromConfig(cfg),
	}, nil
}
