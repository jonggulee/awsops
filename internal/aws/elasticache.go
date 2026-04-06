package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

type ElastiCacheCluster struct {
	Profile          string
	ID               string // replication group ID 또는 cache cluster ID
	Engine           string // "Redis" | "Valkey" | "Memcached"
	EngineVersion    string
	NodeType         string // e.g., "cache.r6g.large"
	Status           string
	NumNodes         int
	Endpoint         string // primary endpoint (Redis RG), config endpoint (Memcached), node endpoint (standalone)
	Port             int32
	SubnetGroupName  string
	SecurityGroupIDs []string
	MultiAZ          string // "enabled" | "disabled" | "-"
	Region           string
}

func FetchAllElastiCacheClusters(ctx context.Context, regions []string) ([]ElastiCacheCluster, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type result struct {
		clusters []ElastiCacheCluster
		errs     []error
	}

	ch := make(chan result, len(profiles)*len(regions))
	for _, p := range profiles {
		for _, r := range regions {
			p, r := p, r
			go func() {
				clusters, errs := fetchElastiCacheForProfileRegion(ctx, p, r)
				ch <- result{clusters, errs}
			}()
		}
	}

	var all []ElastiCacheCluster
	var allErrs []error
	for range profiles {
		for range regions {
			res := <-ch
			all = append(all, res.clusters...)
			allErrs = append(allErrs, res.errs...)
		}
	}
	return all, allErrs
}

func fetchElastiCacheForProfileRegion(ctx context.Context, profile, region string) ([]ElastiCacheCluster, []error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, []error{err}
	}

	type ccResult struct {
		clusters []ectypes.CacheCluster
		err      error
	}
	type rgResult struct {
		groups []ectypes.ReplicationGroup
		err    error
	}

	ccCh := make(chan ccResult, 1)
	rgCh := make(chan rgResult, 1)

	// Cache Clusters 조회 (ShowCacheNodeInfo: true → 노드 엔드포인트 포함)
	go func() {
		var clusters []ectypes.CacheCluster
		var marker *string
		for {
			out, err := client.ElastiCache.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
				ShowCacheNodeInfo: aws.Bool(true),
				Marker:            marker,
			})
			if err != nil {
				ccCh <- ccResult{err: err}
				return
			}
			clusters = append(clusters, out.CacheClusters...)
			if out.Marker == nil {
				break
			}
			marker = out.Marker
		}
		ccCh <- ccResult{clusters: clusters}
	}()

	// Replication Groups 조회 (primary endpoint, multi-AZ, 전체 노드 수)
	go func() {
		var groups []ectypes.ReplicationGroup
		var marker *string
		for {
			out, err := client.ElastiCache.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{
				Marker: marker,
			})
			if err != nil {
				// 권한 없어도 무시 — Cache Clusters만으로 처리
				rgCh <- rgResult{err: err}
				return
			}
			groups = append(groups, out.ReplicationGroups...)
			if out.Marker == nil {
				break
			}
			marker = out.Marker
		}
		rgCh <- rgResult{groups: groups}
	}()

	ccRes := <-ccCh
	rgRes := <-rgCh

	if ccRes.err != nil {
		return nil, []error{fmt.Errorf("[%s/%s] elasticache: %w", profile, region, ccRes.err)}
	}

	// Replication Group 정보 맵 구성
	type rgInfo struct {
		endpoint string
		port     int32
		multiAZ  string
		status   string
		numNodes int
	}
	rgMap := make(map[string]rgInfo)
	if rgRes.err == nil {
		for _, rg := range rgRes.groups {
			info := rgInfo{
				status:  aws.ToString(rg.Status),
				multiAZ: string(rg.MultiAZ),
			}
			for _, ng := range rg.NodeGroups {
				// Cluster Mode Disabled: 각 NodeGroup에 PrimaryEndpoint 존재
				if ng.PrimaryEndpoint != nil && info.endpoint == "" {
					info.endpoint = aws.ToString(ng.PrimaryEndpoint.Address)
					info.port = aws.ToInt32(ng.PrimaryEndpoint.Port)
				}
				info.numNodes += len(ng.NodeGroupMembers)
			}
			// Cluster Mode Enabled: NodeGroup에 PrimaryEndpoint 없고 RG 레벨에 ConfigurationEndpoint 존재
			if info.endpoint == "" && rg.ConfigurationEndpoint != nil {
				info.endpoint = aws.ToString(rg.ConfigurationEndpoint.Address)
				info.port = aws.ToInt32(rg.ConfigurationEndpoint.Port)
			}
			rgMap[aws.ToString(rg.ReplicationGroupId)] = info
		}
	}

	seenRG := make(map[string]bool)
	var result []ElastiCacheCluster

	for _, cc := range ccRes.clusters {
		engine := aws.ToString(cc.Engine)
		rgID := aws.ToString(cc.ReplicationGroupId)

		engineDisplay := engineName(engine)

		if rgID != "" {
			// Replication Group 소속 → RG 기준으로 중복 제거
			if seenRG[rgID] {
				continue
			}
			seenRG[rgID] = true

			info := rgMap[rgID]
			numNodes := info.numNodes
			if numNodes == 0 {
				numNodes = int(aws.ToInt32(cc.NumCacheNodes))
			}
			status := info.status
			if status == "" {
				status = aws.ToString(cc.CacheClusterStatus)
			}
			multiAZ := info.multiAZ
			if multiAZ == "" {
				multiAZ = "-"
			}

			ec := ElastiCacheCluster{
				Profile:         profile,
				ID:              rgID,
				Engine:          engineDisplay,
				EngineVersion:   aws.ToString(cc.EngineVersion),
				NodeType:        aws.ToString(cc.CacheNodeType),
				Status:          status,
				NumNodes:        numNodes,
				Endpoint:        info.endpoint,
				Port:            info.port,
				SubnetGroupName: aws.ToString(cc.CacheSubnetGroupName),
				MultiAZ:         multiAZ,
				Region:          region,
			}
			for _, sg := range cc.SecurityGroups {
				ec.SecurityGroupIDs = append(ec.SecurityGroupIDs, aws.ToString(sg.SecurityGroupId))
			}
			result = append(result, ec)
		} else {
			// 독립 클러스터 (Memcached or standalone Redis)
			var endpoint string
			var port int32
			if cc.ConfigurationEndpoint != nil {
				endpoint = aws.ToString(cc.ConfigurationEndpoint.Address)
				port = aws.ToInt32(cc.ConfigurationEndpoint.Port)
			} else if len(cc.CacheNodes) > 0 && cc.CacheNodes[0].Endpoint != nil {
				endpoint = aws.ToString(cc.CacheNodes[0].Endpoint.Address)
				port = aws.ToInt32(cc.CacheNodes[0].Endpoint.Port)
			}

			ec := ElastiCacheCluster{
				Profile:         profile,
				ID:              aws.ToString(cc.CacheClusterId),
				Engine:          engineDisplay,
				EngineVersion:   aws.ToString(cc.EngineVersion),
				NodeType:        aws.ToString(cc.CacheNodeType),
				Status:          aws.ToString(cc.CacheClusterStatus),
				NumNodes:        int(aws.ToInt32(cc.NumCacheNodes)),
				Endpoint:        endpoint,
				Port:            port,
				SubnetGroupName: aws.ToString(cc.CacheSubnetGroupName),
				MultiAZ:         "-",
				Region:          region,
			}
			for _, sg := range cc.SecurityGroups {
				ec.SecurityGroupIDs = append(ec.SecurityGroupIDs, aws.ToString(sg.SecurityGroupId))
			}
			result = append(result, ec)
		}
	}

	return result, nil
}

// FetchSubnetGroupForElastiCache fetches the subnet IDs in a given cache subnet group.
func FetchSubnetGroupForElastiCache(ctx context.Context, profile, region, subnetGroupName string) ([]string, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	out, err := client.ElastiCache.DescribeCacheSubnetGroups(ctx, &elasticache.DescribeCacheSubnetGroupsInput{
		CacheSubnetGroupName: aws.String(subnetGroupName),
	})
	if err != nil {
		return nil, err
	}
	if len(out.CacheSubnetGroups) == 0 {
		return []string{}, nil
	}
	sg := out.CacheSubnetGroups[0]
	ids := make([]string, 0, len(sg.Subnets))
	for _, s := range sg.Subnets {
		ids = append(ids, aws.ToString(s.SubnetIdentifier))
	}
	return ids, nil
}

func engineName(e string) string {
	switch e {
	case "redis":
		return "Redis"
	case "valkey":
		return "Valkey"
	case "memcached":
		return "Memcached"
	default:
		return e
	}
}
