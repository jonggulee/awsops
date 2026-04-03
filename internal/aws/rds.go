package aws

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// DBInstance represents an RDS DB instance.
type DBInstance struct {
	Profile          string
	Region           string
	DBInstanceID     string
	Name             string // Name 태그 값
	DBInstanceClass  string
	Engine           string
	EngineVersion    string
	Status           string
	Endpoint         string
	Port             int32
	MultiAZ          bool
	StorageType      string
	AllocatedStorage int32
	VpcID            string
	SubnetGroupName  string
	SubnetIDs        []string
	SecurityGroupIDs []string
	AvailabilityZone string
	CreateTime       time.Time
	Tags             map[string]string
}

type rdsResult struct {
	instances []DBInstance
	err       error
}

// FetchAllDBInstances fetches RDS DB instances from all profiles × regions concurrently.
func FetchAllDBInstances(ctx context.Context, regions []string) ([]DBInstance, []error) {
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

	results := make(chan rdsResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			instances, err := fetchDBInstances(ctx, p, r)
			results <- rdsResult{instances: instances, err: err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(results)

	var all []DBInstance
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

func fetchDBInstances(ctx context.Context, profile, region string) ([]DBInstance, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	var instances []DBInstance
	var marker *string

	for {
		out, err := client.RDS.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for _, db := range out.DBInstances {
			instances = append(instances, toDBInstance(profile, region, db))
		}

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	return instances, nil
}

func toDBInstance(profile, region string, db rdstypes.DBInstance) DBInstance {
	tags := make(map[string]string, len(db.TagList))
	name := ""
	for _, t := range db.TagList {
		k, v := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}

	endpoint := ""
	var port int32
	if db.Endpoint != nil {
		endpoint = aws.ToString(db.Endpoint.Address)
		if db.Endpoint.Port != nil {
			port = *db.Endpoint.Port
		}
	}

	vpcID := ""
	subnetGroupName := ""
	var subnetIDs []string
	if db.DBSubnetGroup != nil {
		vpcID = aws.ToString(db.DBSubnetGroup.VpcId)
		subnetGroupName = aws.ToString(db.DBSubnetGroup.DBSubnetGroupName)
		for _, sn := range db.DBSubnetGroup.Subnets {
			subnetIDs = append(subnetIDs, aws.ToString(sn.SubnetIdentifier))
		}
	}

	var sgIDs []string
	for _, sg := range db.VpcSecurityGroups {
		sgIDs = append(sgIDs, aws.ToString(sg.VpcSecurityGroupId))
	}

	createTime := time.Time{}
	if db.InstanceCreateTime != nil {
		createTime = *db.InstanceCreateTime
	}

	return DBInstance{
		Profile:          profile,
		Region:           region,
		DBInstanceID:     aws.ToString(db.DBInstanceIdentifier),
		Name:             name,
		DBInstanceClass:  aws.ToString(db.DBInstanceClass),
		Engine:           aws.ToString(db.Engine),
		EngineVersion:    aws.ToString(db.EngineVersion),
		Status:           aws.ToString(db.DBInstanceStatus),
		Endpoint:         endpoint,
		Port:             port,
		MultiAZ:          aws.ToBool(db.MultiAZ),
		StorageType:      aws.ToString(db.StorageType),
		AllocatedStorage: aws.ToInt32(db.AllocatedStorage),
		VpcID:            vpcID,
		SubnetGroupName:  subnetGroupName,
		SubnetIDs:        subnetIDs,
		SecurityGroupIDs: sgIDs,
		AvailabilityZone: aws.ToString(db.AvailabilityZone),
		CreateTime:       createTime,
		Tags:             tags,
	}
}

func (db DBInstance) CreateTimeStr() string {
	if db.CreateTime.IsZero() {
		return "-"
	}
	return db.CreateTime.In(time.Local).Format("2006-01-02 15:04:05")
}
