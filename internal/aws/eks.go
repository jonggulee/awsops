package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
)

type EKSNode struct {
	InstanceID       string
	Name             string
	State            string
	InstanceType     string
	PrivateIP        string
	AvailabilityZone string
	NodegroupName    string // eks:nodegroup-name 태그
}

type EKSNodegroup struct {
	Name           string
	Status         string
	Version        string
	ReleaseVersion string
	CapacityType   string // ON_DEMAND, SPOT
	InstanceTypes  []string
	AMIType        string
	DiskSize       int32
	DesiredSize    int32
	MinSize        int32
	MaxSize        int32
	NodeRoleARN    string
	CreatedAt      time.Time
}

func (n EKSNodegroup) ScalingStr() string {
	return fmt.Sprintf("%d / %d–%d", n.DesiredSize, n.MinSize, n.MaxSize)
}

func (n EKSNodegroup) CreatedAtStr() string {
	if n.CreatedAt.IsZero() {
		return "-"
	}
	return n.CreatedAt.In(time.Local).Format("2006-01-02 15:04")
}

type EKSCluster struct {
	Profile                string
	Region                 string
	Name                   string
	Status                 string
	Version                string
	PlatformVersion        string
	Endpoint               string
	RoleARN                string
	VpcID                  string
	SubnetIDs              []string
	SecurityGroupIDs       []string
	ClusterSecurityGroupID string
	PublicAccess           bool
	PrivateAccess          bool
	CreatedAt              time.Time
	Tags                   map[string]string
	Nodegroups             []EKSNodegroup
	Nodes                  []EKSNode
}

func (c EKSCluster) CreatedAtStr() string {
	if c.CreatedAt.IsZero() {
		return "-"
	}
	return c.CreatedAt.In(time.Local).Format("2006-01-02 15:04")
}

// FetchAllEKSClusters fetches EKS clusters from all profiles × regions concurrently.
func FetchAllEKSClusters(ctx context.Context, regions []string) ([]EKSCluster, []error) {
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
		clusters []EKSCluster
		err      error
	}
	ch := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			clusters, err := fetchEKSClusters(ctx, p, r)
			ch <- result{clusters, err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(ch)

	var all []EKSCluster
	var errs []error
	for res := range ch {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		all = append(all, res.clusters...)
	}
	return all, errs
}

func fetchEKSClusters(ctx context.Context, profile, region string) ([]EKSCluster, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	// 클러스터 이름 목록
	var names []string
	var nextToken *string
	for {
		out, err := client.EKS.ListClusters(ctx, &eks.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		names = append(names, out.Clusters...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	// 각 클러스터 상세 + 노드그룹 + 노드 EC2를 클러스터별로 병렬 조회
	type clusterResult struct {
		cluster EKSCluster
		ok      bool
	}
	ch := make(chan clusterResult, len(names))
	var wg sync.WaitGroup

	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			desc, err := client.EKS.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: aws.String(n),
			})
			if err != nil || desc.Cluster == nil {
				ch <- clusterResult{}
				return
			}
			c := toEKSCluster(profile, region, desc.Cluster)
			c.Nodegroups = fetchNodegroups(ctx, client, n)
			c.Nodes = fetchClusterNodes(ctx, client, n)
			ch <- clusterResult{cluster: c, ok: true}
		}(name)
	}

	wg.Wait()
	close(ch)

	var clusters []EKSCluster
	for r := range ch {
		if r.ok {
			clusters = append(clusters, r.cluster)
		}
	}
	return clusters, nil
}

// fetchNodegroups retrieves all node groups for a given cluster. Non-fatal on error.
func fetchNodegroups(ctx context.Context, client *ProfileClient, clusterName string) []EKSNodegroup {
	var ngNames []string
	var nextToken *string
	for {
		out, err := client.EKS.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil
		}
		ngNames = append(ngNames, out.Nodegroups...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var nodegroups []EKSNodegroup
	for _, ngName := range ngNames {
		desc, err := client.EKS.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(ngName),
		})
		if err != nil || desc.Nodegroup == nil {
			continue
		}
		nodegroups = append(nodegroups, toEKSNodegroup(desc.Nodegroup))
	}
	return nodegroups
}

// fetchClusterNodes returns EC2 instances tagged as members of the given EKS cluster.
func fetchClusterNodes(ctx context.Context, client *ProfileClient, clusterName string) []EKSNode {
	out, err := client.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:kubernetes.io/cluster/" + clusterName),
				Values: []string{"owned", "shared"},
			},
		},
	})
	if err != nil {
		return nil
	}

	var nodes []EKSNode
	for _, res := range out.Reservations {
		for _, inst := range res.Instances {
			nodes = append(nodes, toEKSNode(inst))
		}
	}
	return nodes
}

func toEKSNode(inst ec2types.Instance) EKSNode {
	name := ""
	nodegroupName := ""
	for _, tag := range inst.Tags {
		switch aws.ToString(tag.Key) {
		case "Name":
			name = aws.ToString(tag.Value)
		case "eks:nodegroup-name":
			nodegroupName = aws.ToString(tag.Value)
		}
	}
	az := ""
	if inst.Placement != nil {
		az = aws.ToString(inst.Placement.AvailabilityZone)
	}
	return EKSNode{
		InstanceID:       aws.ToString(inst.InstanceId),
		Name:             name,
		State:            string(inst.State.Name),
		InstanceType:     string(inst.InstanceType),
		PrivateIP:        aws.ToString(inst.PrivateIpAddress),
		AvailabilityZone: az,
		NodegroupName:    nodegroupName,
	}
}

func toEKSCluster(profile, region string, c *ekstypes.Cluster) EKSCluster {
	cluster := EKSCluster{
		Profile:         profile,
		Region:          region,
		Name:            aws.ToString(c.Name),
		Status:          string(c.Status),
		Version:         aws.ToString(c.Version),
		PlatformVersion: aws.ToString(c.PlatformVersion),
		Endpoint:        aws.ToString(c.Endpoint),
		RoleARN:         aws.ToString(c.RoleArn),
		Tags:            c.Tags,
	}
	if c.CreatedAt != nil {
		cluster.CreatedAt = *c.CreatedAt
	}
	if vpc := c.ResourcesVpcConfig; vpc != nil {
		cluster.VpcID = aws.ToString(vpc.VpcId)
		cluster.SubnetIDs = vpc.SubnetIds
		cluster.SecurityGroupIDs = vpc.SecurityGroupIds
		cluster.ClusterSecurityGroupID = aws.ToString(vpc.ClusterSecurityGroupId)
		cluster.PublicAccess = vpc.EndpointPublicAccess
		cluster.PrivateAccess = vpc.EndpointPrivateAccess
	}
	return cluster
}

func toEKSNodegroup(ng *ekstypes.Nodegroup) EKSNodegroup {
	n := EKSNodegroup{
		Name:           aws.ToString(ng.NodegroupName),
		Status:         string(ng.Status),
		Version:        aws.ToString(ng.Version),
		ReleaseVersion: aws.ToString(ng.ReleaseVersion),
		CapacityType:   string(ng.CapacityType),
		InstanceTypes:  ng.InstanceTypes,
		AMIType:        string(ng.AmiType),
		DiskSize:       aws.ToInt32(ng.DiskSize),
		NodeRoleARN:    aws.ToString(ng.NodeRole),
	}
	if ng.CreatedAt != nil {
		n.CreatedAt = *ng.CreatedAt
	}
	if sc := ng.ScalingConfig; sc != nil {
		n.DesiredSize = aws.ToInt32(sc.DesiredSize)
		n.MinSize = aws.ToInt32(sc.MinSize)
		n.MaxSize = aws.ToInt32(sc.MaxSize)
	}
	return n
}
