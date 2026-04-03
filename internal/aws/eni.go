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
	Name             string
	Description      string
	Status           string
	PrivateIP        string
	PrivateIPs       []string // primary + secondary IP 전체 목록 (IP 역추적용)
	VpcID            string
	SubnetID         string
	InstanceID         string // 비어있으면 EC2에 미연결
	InterfaceType      string
	AvailabilityZone   string
	SecurityGroupIDs   []string
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

// FetchENIsForRDS fetches ENIs associated with an RDS instance.
// AWS sets description="RDSNetworkInterface" on RDS-managed ENIs.
// vpcID is used as an additional filter to narrow the results.
func FetchENIsForRDS(ctx context.Context, profile, region, vpcID, dbInstanceID string) ([]ENI, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	filters := []types.Filter{
		{Name: aws.String("description"), Values: []string{"RDSNetworkInterface"}},
		{Name: aws.String("vpc-id"), Values: []string{vpcID}},
	}

	out, err := client.EC2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: filters,
	})
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

	name := ""
	for _, t := range ni.TagSet {
		if aws.ToString(t.Key) == "Name" {
			name = aws.ToString(t.Value)
			break
		}
	}

	allIPs := make([]string, 0, len(ni.PrivateIpAddresses))
	for _, pip := range ni.PrivateIpAddresses {
		if ip := aws.ToString(pip.PrivateIpAddress); ip != "" {
			allIPs = append(allIPs, ip)
		}
	}

	return ENI{
		Profile:          profile,
		Region:           region,
		ENIID:            aws.ToString(ni.NetworkInterfaceId),
		Name:             name,
		Description:      aws.ToString(ni.Description),
		Status:           string(ni.Status),
		PrivateIP:        aws.ToString(ni.PrivateIpAddress),
		PrivateIPs:       allIPs,
		VpcID:            aws.ToString(ni.VpcId),
		SubnetID:         aws.ToString(ni.SubnetId),
		InstanceID:       instanceID,
		InterfaceType:    string(ni.InterfaceType),
		AvailabilityZone: aws.ToString(ni.AvailabilityZone),
		SecurityGroupIDs: sgIDs,
	}
}
