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
	mapPriorityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	mapActionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	mapCondStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mapSepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
)

// TG마다 고유 색상을 순환 배정하기 위한 팔레트
var tgColorPalette = []lipgloss.Color{
	"205", // 핑크
	"220", // 노랑
	"208", // 오렌지
	"141", // 라벤더
	"51",  // 밝은 시안
	"118", // 밝은 초록
	"214", // 황금색
	"147", // 연보라
}

func renderMapView(m Model) string {
	lb := m.selectedALB
	if lb == nil {
		return ""
	}

	var b strings.Builder
	sep := mapSepStyle.Render(strings.Repeat("─", 72)) + "\n"

	// 타이틀
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("ALB  ›  %s  ›  Resource Map", lb.Name)) + "\n")
	b.WriteString(row("Type", lb.TypeShort()+"  ·  "+lb.Scheme+"  ·  "+coloredALBState(lb.State)))
	b.WriteString("\n")

	// 로딩 중
	if m.mapPending > 0 {
		b.WriteString(sep)
		b.WriteString("  " + m.spinner.View() + " Loading rules and target health...\n")
		b.WriteString(detailHintBar(m.width, hintItem("esc", "Back to detail")))
		return b.String()
	}

	b.WriteString(sep)

	// TG ARN → 색상 맵 (이름 정렬 순서 기준으로 색상 배정)
	tgs := sortedTGsByName(m.albTargetGroups)
	tgColorMap := make(map[string]lipgloss.Color, len(tgs))
	for i, tg := range tgs {
		tgColorMap[tg.ARN] = tgColorPalette[i%len(tgColorPalette)]
	}

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
				actionStr := ruleActionStr(r, tgNameMap, tgColorMap)
				b.WriteString(mapBranchStyle.Render(rulePrefix) + label + "  " + condStr + "  " + actionStr + "\n")
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
					arrow := mapActionStyle.Render("──▶  ")
					nameStr := tgNameStr(a.TargetGroupARN, name, tgColorMap)
					b.WriteString(mapBranchStyle.Render(rulePrefix) + mapPriorityStyle.Render("[default]") + "  " + arrow + nameStr + "\n")
				}
			}
		}
		if !isLast {
			b.WriteString(mapBranchStyle.Render("│") + "\n")
		}
	}

	b.WriteString("\n" + sep)

	// Target Groups 섹션
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
		tgHeader := tgNameStr(tg.ARN, tg.Name, tgColorMap)
		b.WriteString(tgHeader + "  " + meta + "  " + healthStr + "\n")

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

	b.WriteString(detailHintBar(m.width, hintItem("esc/m", "Back to detail"), hintItem("j/k", "Scroll")))
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

// tgNameStr은 TG ARN에 배정된 색상으로 TG 이름을 굵게 렌더링한다.
func tgNameStr(arn, name string, tgColorMap map[string]lipgloss.Color) string {
	if color, ok := tgColorMap[arn]; ok {
		return lipgloss.NewStyle().Foreground(color).Bold(true).Render(name)
	}
	return mapActionStyle.Render(name)
}

func ruleActionStr(r awsclient.ListenerRule, tgNameMap map[string]string, tgColorMap map[string]lipgloss.Color) string {
	tgARNs := r.ForwardTGARNs()
	if len(tgARNs) > 0 {
		arn := tgARNs[0]
		name := tgNameMap[arn]
		if name == "" {
			name = arn
		}
		return mapActionStyle.Render("──▶  ") + tgNameStr(arn, name, tgColorMap)
	}
	for _, a := range r.Actions {
		switch a.Type {
		case "redirect":
			return mapActionStyle.Render(fmt.Sprintf("redirect %s  ──▶  %s", a.RedirectCode, a.RedirectTarget))
		case "fixed-response":
			return mapActionStyle.Render("fixed-response")
		}
	}
	return ""
}
