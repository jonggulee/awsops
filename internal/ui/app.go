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

func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
