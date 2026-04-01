package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ProfileClient struct {
	Profile string
	EC2     *ec2.Client
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

func NewProfileClient(ctx context.Context, profile string) (*ProfileClient, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion("ap-northeast-2"),
	}
	if profile != "default" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile [%s]: %w", profile, err)
	}

	return &ProfileClient{
		Profile: profile,
		EC2:     ec2.NewFromConfig(cfg),
	}, nil
}
