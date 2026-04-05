package ui

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

func fetchInstances() tea.Cmd {
	return fetchInstancesWithRegions([]string{awsclient.DefaultRegion})
}

func fetchSecurityGroups() tea.Cmd {
	return fetchSGWithRegions([]string{awsclient.DefaultRegion})
}

func fetchVPCsWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		vpcs, subnets, errs := awsclient.FetchAllVPCs(context.Background(), regions)
		return vpcsLoadedMsg{vpcs: vpcs, subnets: subnets, errs: errs}
	}
}

func fetchTGWsWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		gws, atts, rts, routes, assocs, errs := awsclient.FetchAllTGWs(context.Background(), regions)
		return tgwLoadedMsg{
			gateways: gws, attachments: atts,
			routeTables: rts, routes: routes, associations: assocs,
			errs: errs,
		}
	}
}

func fetchRouteTablesWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		tables, errs := awsclient.FetchAllRouteTables(context.Background(), regions)
		return routeTablesLoadedMsg{tables: tables, errs: errs}
	}
}

func fetchAccountIDs() tea.Cmd {
	return func() tea.Msg {
		profileToAccount, accountToProfile, _ := awsclient.FetchAccountIDs(context.Background())
		return accountIDsLoadedMsg{profileToAccount: profileToAccount, accountToProfile: accountToProfile}
	}
}

func fetchInstancesWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		instances, errs := awsclient.FetchAllInstances(context.Background(), regions)
		return instancesLoadedMsg{instances: instances, errs: errs}
	}
}

func fetchSGWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		groups, errs := awsclient.FetchAllSecurityGroups(context.Background(), regions)
		return sgLoadedMsg{groups: groups, errs: errs}
	}
}

func fetchENIsWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		enis, errs := awsclient.FetchAllENIs(context.Background(), regions)
		return enisLoadedMsg{enis: enis, errs: errs}
	}
}

func fetchCertificatesWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		certs, errs := awsclient.FetchAllCertificates(context.Background(), regions)
		return certsLoadedMsg{certs: certs, errs: errs}
	}
}

func fetchInstanceTypeSpecs(instances []awsclient.Instance) tea.Cmd {
	// 사용 중인 타입만 unique하게 추출
	seen := map[string]struct{}{}
	for _, inst := range instances {
		if inst.Type != "" {
			seen[inst.Type] = struct{}{}
		}
	}
	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}
	return func() tea.Msg {
		specs, err := awsclient.FetchInstanceTypeSpecs(context.Background(), types)
		return instanceTypeSpecsLoadedMsg{specs: specs, err: err}
	}
}

func fetchEKSWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		clusters, errs := awsclient.FetchAllEKSClusters(context.Background(), regions)
		return eksLoadedMsg{clusters: clusters, errs: errs}
	}
}

func fetchRoute53() tea.Cmd {
	return func() tea.Msg {
		records, errs := awsclient.FetchAllRoute53Records(context.Background())
		return route53LoadedMsg{records: records, errs: errs}
	}
}

func fetchALBWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		lbs, errs := awsclient.FetchAllLoadBalancers(context.Background(), regions)
		return albLoadedMsg{lbs: lbs, errs: errs}
	}
}

func fetchRDSWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		instances, errs := awsclient.FetchAllDBInstances(context.Background(), regions)
		return rdsLoadedMsg{instances: instances, errs: errs}
	}
}

func fetchElastiCacheWithRegions(regions []string) tea.Cmd {
	return func() tea.Msg {
		clusters, errs := awsclient.FetchAllElastiCacheClusters(context.Background(), regions)
		return elastiCacheLoadedMsg{clusters: clusters, errs: errs}
	}
}

// --- S3 ---

func fetchS3Buckets() tea.Cmd {
	return func() tea.Msg {
		buckets, errs := awsclient.FetchAllS3Buckets(context.Background())
		return s3LoadedMsg{buckets: buckets, errs: errs}
	}
}

func fetchS3Tags(profile, bucketName string) tea.Cmd {
	return func() tea.Msg {
		tags, err := awsclient.FetchS3BucketTags(context.Background(), profile, bucketName)
		return s3TagsLoadedMsg{tags: tags, err: err}
	}
}

// --- RDS lazy fetch ---

// fetchENIsForRDS finds primary and standby ENIs for an RDS instance.
//
// Primary: resolved by DNS lookup of db.Endpoint hostname → matched by private IP.
// Standby: searched from allENIs where description="RDSNetworkInterface",
//
//	subnetID is in dbSubnetIDs, SG set exactly matches primary ENI's SGs,
//	and it is not the primary ENI. Exact SG match avoids false positives
//	from other RDS instances that share the same subnet group.
func fetchENIsForRDS(endpoint string, dbSubnetIDs []string, allENIs []awsclient.ENI) tea.Cmd {
	return func() tea.Msg {
		if endpoint == "" {
			return rdsENIsLoadedMsg{enis: []awsclient.ENI{}}
		}

		// 1. DNS resolve → primary IP
		addrs, err := net.LookupHost(endpoint)
		if err != nil {
			return rdsENIsLoadedMsg{enis: []awsclient.ENI{}, err: err}
		}
		primaryIPSet := make(map[string]bool, len(addrs))
		for _, a := range addrs {
			primaryIPSet[a] = true
		}

		subnetSet := make(map[string]bool, len(dbSubnetIDs))
		for _, s := range dbSubnetIDs {
			subnetSet[s] = true
		}

		// 2. Primary ENI 탐색
		var primary *awsclient.ENI
		for i := range allENIs {
			e := &allENIs[i]
			for _, ip := range e.PrivateIPs {
				if primaryIPSet[ip] {
					primary = e
					break
				}
			}
			if primary != nil {
				break
			}
		}

		if primary == nil {
			return rdsENIsLoadedMsg{enis: []awsclient.ENI{}}
		}

		// primary SG 세트 (exact match용)
		primarySGSet := make(map[string]bool, len(primary.SecurityGroupIDs))
		for _, sg := range primary.SecurityGroupIDs {
			primarySGSet[sg] = true
		}

		result := []awsclient.ENI{*primary}

		// 3. Standby ENI 탐색: RDSNetworkInterface + DB 서브넷 + SG 완전 일치 + primary 제외
		for _, e := range allENIs {
			if e.ENIID == primary.ENIID {
				continue
			}
			if e.Description != "RDSNetworkInterface" {
				continue
			}
			if !subnetSet[e.SubnetID] {
				continue
			}
			// SG 세트 완전 일치 확인
			if len(e.SecurityGroupIDs) != len(primary.SecurityGroupIDs) {
				continue
			}
			match := true
			for _, sg := range e.SecurityGroupIDs {
				if !primarySGSet[sg] {
					match = false
					break
				}
			}
			if match {
				result = append(result, e)
			}
		}

		return rdsENIsLoadedMsg{enis: result, primaryENIID: primary.ENIID}
	}
}

// --- ELB lazy fetch (진입 시점에 특정 LB ARN 기준으로 조회) ---

func fetchListenersForLB(profile, region, lbARN string) tea.Cmd {
	return func() tea.Msg {
		listeners, err := awsclient.FetchListenersForLB(context.Background(), profile, region, lbARN)
		return listenersLoadedMsg{listeners: listeners, err: err}
	}
}

func fetchTargetGroupsForLB(profile, region, lbARN string) tea.Cmd {
	return func() tea.Msg {
		tgs, err := awsclient.FetchTargetGroupsForLB(context.Background(), profile, region, lbARN)
		return tgListLoadedMsg{tgs: tgs, err: err}
	}
}

func fetchRulesForListener(profile, region, listenerARN string) tea.Cmd {
	return func() tea.Msg {
		rules, err := awsclient.FetchRulesForListener(context.Background(), profile, region, listenerARN)
		return rulesLoadedMsg{rules: rules, err: err}
	}
}

func fetchTargetHealthForTG(profile, region, tgARN string) tea.Cmd {
	return func() tea.Msg {
		targets, err := awsclient.FetchTargetHealthForTG(context.Background(), profile, region, tgARN)
		return targetHealthLoadedMsg{targets: targets, err: err}
	}
}

// --- Resource Map 전용 fetch (ARN을 메시지에 포함해서 맵에 저장) ---

func fetchMapRulesForListener(profile, region, listenerARN string) tea.Cmd {
	return func() tea.Msg {
		rules, _ := awsclient.FetchRulesForListener(context.Background(), profile, region, listenerARN)
		if rules == nil {
			rules = []awsclient.ListenerRule{}
		}
		return mapRulesLoadedMsg{listenerARN: listenerARN, rules: rules}
	}
}

func fetchMapTargetHealthForTG(profile, region, tgARN string) tea.Cmd {
	return func() tea.Msg {
		targets, _ := awsclient.FetchTargetHealthForTG(context.Background(), profile, region, tgARN)
		if targets == nil {
			targets = []awsclient.TargetEntry{}
		}
		return mapTargetHealthLoadedMsg{tgARN: tgARN, targets: targets}
	}
}

func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
