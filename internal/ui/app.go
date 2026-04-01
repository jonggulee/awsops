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

func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
