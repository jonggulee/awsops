package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type SGRule struct {
	Protocol  string
	FromPort  int32
	ToPort    int32
	Source    string // CIDR or SG ID
	Direction string // inbound / outbound
}

func (r SGRule) PortRange() string {
	if r.Protocol == "-1" {
		return "All"
	}
	if r.FromPort == r.ToPort {
		return fmt.Sprintf("%d", r.FromPort)
	}
	return fmt.Sprintf("%d-%d", r.FromPort, r.ToPort)
}

func (r SGRule) ProtocolStr() string {
	switch r.Protocol {
	case "-1":
		return "All"
	case "tcp":
		return "TCP"
	case "udp":
		return "UDP"
	case "icmp":
		return "ICMP"
	default:
		return strings.ToUpper(r.Protocol)
	}
}

type SecurityGroup struct {
	Profile     string
	Region      string
	GroupID     string
	Name        string
	Description string
	VpcID       string
	Rules       []SGRule
}

type sgProfileResult struct {
	groups []SecurityGroup
	err    error
}

// FetchAllSecurityGroups fetches security groups from all profiles × regions concurrently.
func FetchAllSecurityGroups(ctx context.Context, regions []string) ([]SecurityGroup, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type target struct {
		profile string
		region  string
	}

	var targets []target
	for _, p := range profiles {
		for _, r := range regions {
			targets = append(targets, target{p, r})
		}
	}

	results := make(chan sgProfileResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			groups, err := fetchSecurityGroups(ctx, p, r)
			results <- sgProfileResult{groups: groups, err: err}
		}(t.profile, t.region)
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

func fetchSecurityGroups(ctx context.Context, profile, region string) ([]SecurityGroup, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	var groups []SecurityGroup
	for _, sg := range out.SecurityGroups {
		groups = append(groups, toSecurityGroup(profile, region, sg))
	}
	return groups, nil
}

func toSecurityGroup(profile, region string, sg types.SecurityGroup) SecurityGroup {
	var rules []SGRule

	for _, p := range sg.IpPermissions {
		sources := ipSources(p)
		for _, src := range sources {
			rules = append(rules, SGRule{
				Protocol:  aws.ToString(p.IpProtocol),
				FromPort:  aws.ToInt32(p.FromPort),
				ToPort:    aws.ToInt32(p.ToPort),
				Source:    src,
				Direction: "inbound",
			})
		}
	}

	for _, p := range sg.IpPermissionsEgress {
		sources := ipSources(p)
		for _, src := range sources {
			rules = append(rules, SGRule{
				Protocol:  aws.ToString(p.IpProtocol),
				FromPort:  aws.ToInt32(p.FromPort),
				ToPort:    aws.ToInt32(p.ToPort),
				Source:    src,
				Direction: "outbound",
			})
		}
	}

	return SecurityGroup{
		Profile:     profile,
		Region:      region,
		GroupID:     aws.ToString(sg.GroupId),
		Name:        aws.ToString(sg.GroupName),
		Description: aws.ToString(sg.Description),
		VpcID:       aws.ToString(sg.VpcId),
		Rules:       rules,
	}
}

func ipSources(p types.IpPermission) []string {
	var sources []string
	for _, r := range p.IpRanges {
		sources = append(sources, aws.ToString(r.CidrIp))
	}
	for _, r := range p.Ipv6Ranges {
		sources = append(sources, aws.ToString(r.CidrIpv6))
	}
	for _, r := range p.UserIdGroupPairs {
		sources = append(sources, aws.ToString(r.GroupId))
	}
	if len(sources) == 0 {
		sources = append(sources, "-")
	}
	return sources
}
