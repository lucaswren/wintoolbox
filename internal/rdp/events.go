package rdp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"wintoolbox/internal/win/syscmd"
)

//go:embed events.ps1
var eventsScript string

const sessionChannel = "Microsoft-Windows-TerminalServices-LocalSessionManager/Operational"
const authChannel = "Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational"

type EventField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type ServerEvent struct {
	Time      string       `json:"time"`
	ID        int          `json:"id"`
	RecordID  string       `json:"recordId"`
	Provider  string       `json:"provider"`
	Channel   string       `json:"channel"`
	User      string       `json:"user"`
	Address   string       `json:"address"`
	Session   string       `json:"session"`
	LogonType string       `json:"logonType"`
	Status    string       `json:"status"`
	SubStatus string       `json:"subStatus"`
	Detail    string       `json:"detail"`
	XML       string       `json:"xml"`
	Data      []EventField `json:"data"`
	Limited   bool         `json:"limited"`
	Category  string       `json:"category"`
	Meaning   string       `json:"meaning"`
	Reason    string       `json:"reason"`
}
type LoginSummary struct {
	Total           int `json:"total"`
	Logins          int `json:"logins"`
	Failures        int `json:"failures"`
	NetworkFailures int `json:"networkFailures"`
	Disconnects     int `json:"disconnects"`
	Reconnects      int `json:"reconnects"`
}
type LoginSource struct {
	Address         string `json:"address"`
	Logins          int    `json:"logins"`
	Failures        int    `json:"failures"`
	NetworkFailures int    `json:"networkFailures"`
	LastSeen        string `json:"lastSeen"`
}
type LoginFinding struct {
	Kind    string `json:"kind"`
	Address string `json:"address"`
	User    string `json:"user"`
	Count   int    `json:"count"`
	Start   string `json:"start"`
	End     string `json:"end"`
	Meaning string `json:"meaning"`
}
type ServerEvents struct {
	Events      []ServerEvent  `json:"events"`
	Warnings    []string       `json:"warnings"`
	Summary     LoginSummary   `json:"summary"`
	Sources     []LoginSource  `json:"sources"`
	Findings    []LoginFinding `json:"findings"`
	Truncated   bool           `json:"truncated"`
	Partial     bool           `json:"partial"`
	CollectedAt string         `json:"collectedAt"`
	StartTime   string         `json:"startTime"`
}

func GetServerEvents() (ServerEvents, error) { return ReadLoginReport(context.Background(), 7, 300) }

func ReadLoginReport(ctx context.Context, days, limit int) (ServerEvents, error) {
	if days < 1 || days > 90 || limit < 50 || limit > 1000 {
		return ServerEvents{}, fmt.Errorf("日志范围须为 1–90 天，条数须为 50–1000")
	}
	// Log reads never install recovery scripts, adjust ACLs, or acquire the network mutex.
	out, err := syscmd.RunPSContext(ctx, fmt.Sprintf("$days=%d; $limit=%d\n%s", days, limit, eventsScript))
	if err != nil {
		return ServerEvents{}, fmt.Errorf("读取远程登录日志失败: %w", err)
	}
	var report ServerEvents
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &report); err != nil {
		return report, fmt.Errorf("解析远程登录日志失败: %w", err)
	}
	return AnalyzeLogins(report), nil
}

func loginCategory(e ServerEvent) (string, string) {
	if e.Channel == sessionChannel && strings.EqualFold(e.Provider, "Microsoft-Windows-TerminalServices-LocalSessionManager") {
		if strings.EqualFold(e.Address, "LOCAL") {
			return "other", "本地会话记录，不能当作远程登录"
		}
		if (e.ID == 21 || e.ID == 25) && sourceKey(e.Address) == "" {
			return "session", "会话记录未提供有效来源地址，不能单独确认是远程连接"
		}
		switch e.ID {
		case 21:
			return "login", "远程桌面会话登录记录"
		case 22:
			return "desktop", "远程桌面外壳启动记录"
		case 23:
			return "logoff", "会话注销记录"
		case 24, 39, 40:
			return "disconnect", "会话断开记录；断开不等于注销"
		case 25:
			return "reconnect", "远程桌面重新连接记录"
		}
	}
	if e.Channel == authChannel && strings.EqualFold(e.Provider, "Microsoft-Windows-TerminalServices-RemoteConnectionManager") && e.ID == 1149 {
		return "auth", "身份验证成功；不能单独证明已进入远程桌面"
	}
	if e.Channel == "Security" && strings.EqualFold(e.Provider, "Microsoft-Windows-Security-Auditing") {
		switch e.ID {
		case 4624:
			if e.LogonType == "10" {
				return "login", "远程交互式登录记录（类型 10）"
			}
		case 4625:
			if e.LogonType == "10" {
				return "failure", "远程交互式登录失败"
			}
			if e.LogonType == "3" {
				return "network-failure", "网络认证失败（类型 3）；不能单独确认是 RDP"
			}
		case 4634:
			if e.LogonType == "10" {
				return "logoff", "远程交互式会话注销记录"
			}
		case 4778, 4779:
			if strings.HasPrefix(strings.ToLower(e.Session), "rdp-") {
				if e.ID == 4778 {
					return "reconnect", "远程桌面重新连接记录"
				}
				return "disconnect", "会话断开记录；断开不等于注销"
			}
			return "other", "会话连接变化；未确认是 RDP 会话"
		case 1102:
			return "logclear", "安全审计日志被清除，历史证据可能不完整"
		}
	}
	return "other", "其他事件；结合来源和原文判断"
}

func failureReason(e ServerEvent) string {
	code := strings.ToLower(e.SubStatus)
	if code == "" || code == "0x0" || code == "0x00000000" {
		code = strings.ToLower(e.Status)
	}
	switch code {
	case "0xc0000064":
		return "账户不存在"
	case "0xc000006a":
		return "密码不正确"
	case "0xc0000234":
		return "账户已锁定"
	case "0xc0000072":
		return "账户已禁用"
	case "0xc0000193":
		return "账户已过期"
	case "0xc0000071":
		return "密码已过期"
	case "0xc000015b":
		return "未授予所需登录权限"
	case "0xc000006d":
		return "用户名或身份验证信息不正确"
	default:
		return "请结合状态码和原始事件判断失败原因"
	}
}

func sourceKey(value string) string {
	ip, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil || ip.IsUnspecified() {
		return ""
	}
	return ip.Unmap().String()
}

func AnalyzeLogins(report ServerEvents) ServerEvents {
	if report.Events == nil {
		report.Events = []ServerEvent{}
	}
	if report.Warnings == nil {
		report.Warnings = []string{}
	}
	report.Summary = LoginSummary{}
	report.Sources = []LoginSource{}
	report.Findings = []LoginFinding{}
	sort.SliceStable(report.Events, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, report.Events[i].Time)
		b, _ := time.Parse(time.RFC3339Nano, report.Events[j].Time)
		return a.Before(b)
	})
	sources := map[string]*LoginSource{}
	failed := map[string][]ServerEvent{}
	seen := map[string]bool{}
	unique := make([]ServerEvent, 0, len(report.Events))
	for _, e := range report.Events {
		key := e.Channel + ":" + e.RecordID
		if e.RecordID != "" && seen[key] {
			continue
		}
		seen[key] = true
		e.Category, e.Meaning = loginCategory(e)
		e.Reason = ""
		if e.Category == "failure" || e.Category == "network-failure" {
			e.Reason = failureReason(e)
		}
		unique = append(unique, e)
		switch e.Category {
		case "login":
			report.Summary.Logins++
		case "failure":
			report.Summary.Failures++
		case "network-failure":
			report.Summary.NetworkFailures++
		case "disconnect":
			report.Summary.Disconnects++
		case "reconnect":
			report.Summary.Reconnects++
		case "logclear":
			report.Warnings = append(report.Warnings, e.Meaning)
		}
		address := sourceKey(e.Address)
		if address == "" {
			continue
		}
		if sources[address] == nil {
			sources[address] = &LoginSource{Address: address}
		}
		s := sources[address]
		s.LastSeen = e.Time
		if e.Category == "login" {
			s.Logins++
		}
		if e.Category == "failure" {
			s.Failures++
		}
		if e.Category == "network-failure" {
			s.NetworkFailures++
		}
		if _, err := time.Parse(time.RFC3339Nano, e.Time); err == nil && (e.Category == "failure" || e.Category == "network-failure") {
			failed[address] = append(failed[address], e)
		}
	}
	for address, events := range failed {
		left, bestStart, bestEnd := 0, 0, 0
		for right, event := range events {
			at, _ := time.Parse(time.RFC3339Nano, event.Time)
			for left < right {
				earlier, _ := time.Parse(time.RFC3339Nano, events[left].Time)
				if at.Sub(earlier) <= 5*time.Minute {
					break
				}
				left++
			}
			if right-left > bestEnd-bestStart {
				bestStart, bestEnd = left, right
			}
		}
		count := bestEnd - bestStart + 1
		if count >= 5 {
			report.Findings = append(report.Findings, LoginFinding{Kind: "repeated-failure", Address: address, Count: count, Start: events[bestStart].Time, End: events[bestEnd].Time, Meaning: "五分钟内同一来源多次认证失败；可能包含非 RDP 流量，也可能是密码配置错误，请核对来源和账户"})
		}
	}
	for _, source := range sources {
		report.Sources = append(report.Sources, *source)
	}
	sort.Slice(report.Sources, func(i, j int) bool {
		a, b := report.Sources[i], report.Sources[j]
		if a.Failures+a.NetworkFailures != b.Failures+b.NetworkFailures {
			return a.Failures+a.NetworkFailures > b.Failures+b.NetworkFailures
		}
		return a.Address < b.Address
	})
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Count != report.Findings[j].Count {
			return report.Findings[i].Count > report.Findings[j].Count
		}
		return report.Findings[i].Address < report.Findings[j].Address
	})
	report.Events = unique
	sort.SliceStable(report.Events, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, report.Events[i].Time)
		b, _ := time.Parse(time.RFC3339Nano, report.Events[j].Time)
		return a.After(b)
	})
	report.Summary.Total = len(unique)
	report.Warnings = append(report.Warnings, "统计为事件条数，不是独立登录人数或攻击次数；跨通道事件可能描述同一次连接。缺少记录不能证明没有登录。")
	if report.Truncated {
		report.Warnings = append(report.Warnings, "已达到条数上限，当前时间范围的记录可能未完整显示；请缩小范围或提高上限。")
	}
	dedup := map[string]bool{}
	warnings := []string{}
	for _, warning := range report.Warnings {
		if !dedup[warning] {
			warnings = append(warnings, warning)
			dedup[warning] = true
		}
	}
	report.Warnings = warnings
	return report
}
