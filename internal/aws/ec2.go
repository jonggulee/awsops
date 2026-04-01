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

type Instance struct {
	Profile          string
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
}

type profileResult struct {
	instances []Instance
	err       error
}

// FetchAllInstances fetches EC2 instances from all profiles concurrently.
func FetchAllInstances(ctx context.Context) ([]Instance, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	results := make(chan profileResult, len(profiles))
	var wg sync.WaitGroup

	for _, profile := range profiles {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			instances, err := fetchInstances(ctx, p)
			results <- profileResult{instances: instances, err: err}
		}(profile)
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

func fetchInstances(ctx context.Context, profile string) ([]Instance, error) {
	client, err := NewProfileClient(ctx, profile)
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
			instances = append(instances, toInstance(profile, inst))
		}
	}
	return instances, nil
}

func toInstance(profile string, inst types.Instance) Instance {
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

	return Instance{
		Profile:          profile,
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
