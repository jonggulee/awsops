package ui

import (
	"context"
	"fmt"

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

func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
