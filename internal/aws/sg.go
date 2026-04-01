package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type SecurityGroup struct {
	Profile     string
	GroupID     string
	Name        string
	Description string
	VpcID       string
}

type sgProfileResult struct {
	groups []SecurityGroup
	err    error
}

// FetchAllSecurityGroups fetches security groups from all profiles concurrently.
func FetchAllSecurityGroups(ctx context.Context) ([]SecurityGroup, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	results := make(chan sgProfileResult, len(profiles))
	var wg sync.WaitGroup

	for _, profile := range profiles {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			groups, err := fetchSecurityGroups(ctx, p)
			results <- sgProfileResult{groups: groups, err: err}
		}(profile)
	}

	wg.Wait()
	close(results)

	var all []SecurityGroup
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		all = append(all, r.groups...)
	}
	return all, errs
}

func fetchSecurityGroups(ctx context.Context, profile string) ([]SecurityGroup, error) {
	client, err := NewProfileClient(ctx, profile)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	var groups []SecurityGroup
	for _, sg := range out.SecurityGroups {
		groups = append(groups, toSecurityGroup(profile, sg))
	}
	return groups, nil
}

func toSecurityGroup(profile string, sg types.SecurityGroup) SecurityGroup {
	return SecurityGroup{
		Profile:     profile,
		GroupID:     aws.ToString(sg.GroupId),
		Name:        aws.ToString(sg.GroupName),
		Description: aws.ToString(sg.Description),
		VpcID:       aws.ToString(sg.VpcId),
	}
}
