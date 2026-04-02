package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SGRef is a lightweight security group reference (ID + name) attached to an instance.
type SGRef struct {
	ID   string
	Name string
}

type Instance struct {
	Profile          string
	Region           string
	InstanceID       string
	Name             string
	State            string
	Type             string
	PublicIP         string
	PrivateIP        string
	AMIID            string
	VpcID            string
	SubnetID         string
	AvailabilityZone string
	KeyName          string
	LaunchTime       time.Time
	Tags             map[string]string
	SecurityGroups   []SGRef
}

type profileResult struct {
	instances []Instance
	err       error
}

// FetchAllInstances fetches EC2 instances from all profiles × regions concurrently.
func FetchAllInstances(ctx context.Context, regions []string) ([]Instance, []error) {
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

	results := make(chan profileResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			instances, err := fetchInstances(ctx, p, r)
			results <- profileResult{instances: instances, err: err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(results)

	var all []Instance
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		all = append(all, r.instances...)
	}
	return all, errs
}

func fetchInstances(ctx context.Context, profile, region string) ([]Instance, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	var instances []Instance
	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			instances = append(instances, toInstance(profile, region, inst))
		}
	}
	return instances, nil
}

func toInstance(profile, region string, inst types.Instance) Instance {
	tags := make(map[string]string)
	name := ""
	for _, tag := range inst.Tags {
		k, v := aws.ToString(tag.Key), aws.ToString(tag.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}

	launchTime := time.Time{}
	if inst.LaunchTime != nil {
		launchTime = *inst.LaunchTime
	}

	az := ""
	if inst.Placement != nil {
		az = aws.ToString(inst.Placement.AvailabilityZone)
	}

	var sgs []SGRef
	for _, sg := range inst.SecurityGroups {
		sgs = append(sgs, SGRef{
			ID:   aws.ToString(sg.GroupId),
			Name: aws.ToString(sg.GroupName),
		})
	}

	return Instance{
		Profile:          profile,
		Region:           region,
		InstanceID:       aws.ToString(inst.InstanceId),
		Name:             name,
		State:            string(inst.State.Name),
		Type:             string(inst.InstanceType),
		PublicIP:         aws.ToString(inst.PublicIpAddress),
		PrivateIP:        aws.ToString(inst.PrivateIpAddress),
		AMIID:            aws.ToString(inst.ImageId),
		VpcID:            aws.ToString(inst.VpcId),
		SubnetID:         aws.ToString(inst.SubnetId),
		AvailabilityZone: az,
		KeyName:          aws.ToString(inst.KeyName),
		LaunchTime:       launchTime,
		Tags:             tags,
		SecurityGroups:   sgs,
	}
}

func (inst Instance) LaunchTimeStr() string {
	if inst.LaunchTime.IsZero() {
		return "-"
	}
	return inst.LaunchTime.In(time.Local).Format("2006-01-02 15:04:05")
}

func (inst Instance) TagsStr() string {
	if len(inst.Tags) == 0 {
		return "-"
	}
	result := ""
	for k, v := range inst.Tags {
		if k == "Name" {
			continue
		}
		result += fmt.Sprintf("%s=%s  ", k, v)
	}
	if result == "" {
		return "-"
	}
	return result
}
