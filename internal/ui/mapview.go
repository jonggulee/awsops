package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

var (
	mapBranchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	mapListenerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	mapTGNameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	mapPriorityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	mapActionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	mapCondStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mapSepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
)

func renderMapView(m Model) string {
	lb := m.selectedALB
	if lb == nil {
		return ""
	}

	var b strings.Builder
	sep := mapSepStyle.Render(strings.Repeat("─", 72)) + "\n"

	// 타이틀
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Resource Map  ›  %s", lb.Name)) + "\n")
	b.WriteString(row("Type", lb.TypeShort()+"  ·  "+lb.Scheme+"  ·  "+coloredALBState(lb.State)))
	b.WriteString("\n")

	// 로딩 중
	if m.mapPending > 0 {
		b.WriteString(sep)
		b.WriteString("  " + m.spinner.View() + " Loading rules and target health...\n")
		b.WriteString("\n" + helpStyle.Render("esc  back to detail"))
		return b.String()
	}

	b.WriteString(sep)

	// Listeners → Rules 섹션
	listeners := sortedListenersByPort(m.albListeners)
	tgNameMap := m.buildTGNameMap()

	for i, li := range listeners {
		isLast := i == len(listeners)-1
		prefix := "├─ "
		childPrefix := "│   "
		if isLast {
			prefix = "└─ "
			childPrefix = "    "
		}

		b.WriteString(mapBranchStyle.Render(prefix) + mapListenerStyle.Render(li.Title()) + "\n")

		rules := m.mapRules[li.ARN]
		if li.IsALB() {
			sorted := sortedRules(rules)
			for j, r := range sorted {
				isLastRule := j == len(sorted)-1
				rulePrefix := childPrefix + "├─ "
				if isLastRule {
					rulePrefix = childPrefix + "└─ "
				}
				label := mapPriorityStyle.Render(fmt.Sprintf("[%-7s]", r.Priority))
				if r.IsDefault {
					label = mapPriorityStyle.Render("[default]")
				}
				condStr := mapCondStyle.Render(ruleCondSummary(r.Conditions))
				actionStr := ruleActionStr(r, tgNameMap)
				b.WriteString(mapBranchStyle.Render(rulePrefix) + label + "  " + condStr + "  " + mapActionStyle.Render(actionStr) + "\n")
			}
		} else {
			// NLB: default action만
			for _, a := range li.DefaultActions {
				rulePrefix := childPrefix + "└─ "
				if a.Type == "forward" {
					name := tgNameMap[a.TargetGroupARN]
					if name == "" {
						name = a.TargetGroupARN
					}
					b.WriteString(mapBranchStyle.Render(rulePrefix) + mapPriorityStyle.Render("[default]") + "  " + mapActionStyle.Render("──▶  "+name) + "\n")
				}
			}
		}
		if !isLast {
			b.WriteString(mapBranchStyle.Render("│") + "\n")
		}
	}

	b.WriteString("\n" + sep)

	// Target Groups 섹션
	tgs := sortedTGsByName(m.albTargetGroups)
	for i, tg := range tgs {
		targets := m.mapTargetHealth[tg.ARN]

		healthy, unhealthy, other := 0, 0, 0
		for _, t := range targets {
			switch t.State {
			case "healthy":
				healthy++
			case "unhealthy":
				unhealthy++
			default:
				other++
			}
		}
		healthStr := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(fmt.Sprintf("●%d", healthy)) +
			" " + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("○%d", unhealthy+other))

		portStr := ""
		if tg.Port > 0 {
			portStr = fmt.Sprintf(":%d", tg.Port)
		}
		meta := mapSepStyle.Render(tg.Protocol+portStr+"  "+tg.TargetType)
		b.WriteString(mapTGNameStyle.Render(tg.Name) + "  " + meta + "  " + healthStr + "\n")

		sorted := sortedTargets(targets)
		for j, t := range sorted {
			isLastTarget := j == len(sorted)-1
			treePrefix := "  ├─ "
			if isLastTarget {
				treePrefix = "  └─ "
			}

			addr := t.ID
			if t.Port > 0 {
				addr = fmt.Sprintf("%s:%d", t.ID, t.Port)
			}

			var hint string
			switch tg.TargetType {
			case "instance":
				if name := m.lookupInstanceName(t.ID); name != "" {
					hint = nameTagStyle.Render("[" + name + "]")
				}
			case "ip":
				if instID, name := m.lookupNodeByIP(t.ID); instID != "" {
					node := instID
					if name != "" {
						node = name + " (" + instID + ")"
					}
					hint = nameTagStyle.Render("[node: " + node + "]")
				}
			}

			state := coloredTargetState(t.State)
			line := mapBranchStyle.Render(treePrefix) + fmt.Sprintf("%-38s", addr)
			if hint != "" {
				line += "  " + hint
			}
			line += "  " + state + "  " + t.AZ
			if t.Description != "" {
				line += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(t.Description)
			}
			b.WriteString(line + "\n")
		}

		if i < len(tgs)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("esc / m  back to detail    j/k  scroll"))
	return b.String()
}

// --- 정렬 헬퍼 ---

func sortedListenersByPort(listeners []awsclient.Listener) []awsclient.Listener {
	out := make([]awsclient.Listener, len(listeners))
	copy(out, listeners)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Port < out[j].Port
	})
	return out
}

func sortedRules(rules []awsclient.ListenerRule) []awsclient.ListenerRule {
	out := make([]awsclient.ListenerRule, len(rules))
	copy(out, rules)
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDefault {
			return false
		}
		if out[j].IsDefault {
			return true
		}
		// priority는 숫자 문자열이므로 길이 → 값 순 비교
		if len(out[i].Priority) != len(out[j].Priority) {
			return len(out[i].Priority) < len(out[j].Priority)
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}

func sortedTGsByName(tgs []awsclient.TargetGroup) []awsclient.TargetGroup {
	out := make([]awsclient.TargetGroup, len(tgs))
	copy(out, tgs)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func sortedTargets(targets []awsclient.TargetEntry) []awsclient.TargetEntry {
	out := make([]awsclient.TargetEntry, len(targets))
	copy(out, targets)
	sort.Slice(out, func(i, j int) bool {
		si := targetStatePriority(out[i].State)
		sj := targetStatePriority(out[j].State)
		if si != sj {
			return si < sj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func targetStatePriority(state string) int {
	switch state {
	case "healthy":
		return 0
	case "unhealthy":
		return 1
	case "draining":
		return 2
	default:
		return 3
	}
}

// --- 텍스트 헬퍼 ---

func ruleCondSummary(conditions []awsclient.RuleCondition) string {
	var parts []string
	for _, c := range conditions {
		if len(c.Values) > 0 {
			vals := strings.Join(c.Values, ", ")
			if len(vals) > 40 {
				vals = vals[:37] + "..."
			}
			parts = append(parts, c.Field+": "+vals)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	result := strings.Join(parts, "  |  ")
	return result
}

func ruleActionStr(r awsclient.ListenerRule, tgNameMap map[string]string) string {
	tgARNs := r.ForwardTGARNs()
	if len(tgARNs) > 0 {
		name := tgNameMap[tgARNs[0]]
		if name == "" {
			name = tgARNs[0]
		}
		return "──▶  " + name
	}
	for _, a := range r.Actions {
		switch a.Type {
		case "redirect":
			return fmt.Sprintf("redirect %s  ──▶  %s", a.RedirectCode, a.RedirectTarget)
		case "fixed-response":
			return "fixed-response"
		}
	}
	return ""
}
