package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type VPC struct {
	Profile   string
	Region    string
	VpcID     string
	Name      string
	CidrBlock string
	State     string
	IsDefault bool
	Tags      map[string]string
}

type Subnet struct {
	Profile          string
	Region           string
	SubnetID         string
	Name             string
	VpcID            string
	CidrBlock        string
	AvailabilityZone string
	AvailableIPs     int32
	IsDefault        bool
	Tags             map[string]string
}

type vpcResult struct {
	vpcs    []VPC
	subnets []Subnet
	err     error
}

// FetchAllVPCs fetches VPCs and Subnets from all profiles × regions concurrently.
func FetchAllVPCs(ctx context.Context, regions []string) ([]VPC, []Subnet, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, nil, []error{err}
	}

	type target struct{ profile, region string }
	var targets []target
	for _, p := range profiles {
		for _, r := range regions {
			targets = append(targets, target{p, r})
		}
	}

	results := make(chan vpcResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			vpcs, subnets, err := fetchVPCs(ctx, p, r)
			results <- vpcResult{vpcs: vpcs, subnets: subnets, err: err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(results)

	var allVPCs []VPC
	var allSubnets []Subnet
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		allVPCs = append(allVPCs, r.vpcs...)
		allSubnets = append(allSubnets, r.subnets...)
	}
	return allVPCs, allSubnets, errs
}

func fetchVPCs(ctx context.Context, profile, region string) ([]VPC, []Subnet, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, nil, err
	}

	vpcOut, err := client.EC2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, nil, err
	}

	subnetOut, err := client.EC2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, nil, err
	}

	var vpcs []VPC
	for _, v := range vpcOut.Vpcs {
		vpcs = append(vpcs, toVPC(profile, region, v))
	}

	var subnets []Subnet
	for _, s := range subnetOut.Subnets {
		subnets = append(subnets, toSubnet(profile, region, s))
	}

	return vpcs, subnets, nil
}

func toVPC(profile, region string, v types.Vpc) VPC {
	tags := make(map[string]string)
	name := ""
	for _, t := range v.Tags {
		k, val := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = val
		if k == "Name" {
			name = val
		}
	}
	return VPC{
		Profile:   profile,
		Region:    region,
		VpcID:     aws.ToString(v.VpcId),
		Name:      name,
		CidrBlock: aws.ToString(v.CidrBlock),
		State:     string(v.State),
		IsDefault: aws.ToBool(v.IsDefault),
		Tags:      tags,
	}
}

func toSubnet(profile, region string, s types.Subnet) Subnet {
	tags := make(map[string]string)
	name := ""
	for _, t := range s.Tags {
		k, val := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = val
		if k == "Name" {
			name = val
		}
	}
	return Subnet{
		Profile:          profile,
		Region:           region,
		SubnetID:         aws.ToString(s.SubnetId),
		Name:             name,
		VpcID:            aws.ToString(s.VpcId),
		CidrBlock:        aws.ToString(s.CidrBlock),
		AvailabilityZone: aws.ToString(s.AvailabilityZone),
		AvailableIPs:     aws.ToInt32(s.AvailableIpAddressCount),
		IsDefault:        aws.ToBool(s.DefaultForAz),
		Tags:             tags,
	}
}
