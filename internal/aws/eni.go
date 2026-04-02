package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ENI struct {
	Profile          string
	Region           string
	ENIID            string
	Description      string
	Status           string
	PrivateIP        string
	VpcID            string
	SubnetID         string
	InstanceID       string // 비어있으면 EC2에 미연결
	InterfaceType    string
	SecurityGroupIDs []string
}

// SecurityGroupIDs returns true if the ENI is associated with the given SG ID.
func (e ENI) HasSG(sgID string) bool {
	for _, id := range e.SecurityGroupIDs {
		if id == sgID {
			return true
		}
	}
	return false
}

// FetchAllENIs fetches ENIs from all profiles × regions concurrently.
func FetchAllENIs(ctx context.Context, regions []string) ([]ENI, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type target struct{ profile, region string }
	var targets []target
	for _, p := range profiles {
		for _, r := range regions {
			targets = append(targets, target{p, r})
		}
	}

	type result struct {
		enis []ENI
		err  error
	}
	ch := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			enis, err := fetchENIs(ctx, p, r)
			ch <- result{enis, err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(ch)

	var all []ENI
	var errs []error
	for res := range ch {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		all = append(all, res.enis...)
	}
	return all, errs
}

func fetchENIs(ctx context.Context, profile, region string) ([]ENI, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{})
	if err != nil {
		return nil, err
	}

	enis := make([]ENI, 0, len(out.NetworkInterfaces))
	for _, ni := range out.NetworkInterfaces {
		enis = append(enis, toENI(profile, region, ni))
	}
	return enis, nil
}

func toENI(profile, region string, ni types.NetworkInterface) ENI {
	instanceID := ""
	if ni.Attachment != nil {
		instanceID = aws.ToString(ni.Attachment.InstanceId)
	}

	sgIDs := make([]string, 0, len(ni.Groups))
	for _, g := range ni.Groups {
		sgIDs = append(sgIDs, aws.ToString(g.GroupId))
	}

	return ENI{
		Profile:          profile,
		Region:           region,
		ENIID:            aws.ToString(ni.NetworkInterfaceId),
		Description:      aws.ToString(ni.Description),
		Status:           string(ni.Status),
		PrivateIP:        aws.ToString(ni.PrivateIpAddress),
		VpcID:            aws.ToString(ni.VpcId),
		SubnetID:         aws.ToString(ni.SubnetId),
		InstanceID:       instanceID,
		InterfaceType:    string(ni.InterfaceType),
		SecurityGroupIDs: sgIDs,
	}
}
