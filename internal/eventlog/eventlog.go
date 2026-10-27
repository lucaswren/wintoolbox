package eventlog

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"wintoolbox/internal/win/syscmd"
)

//go:embed collect.ps1
var collector string

type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Event struct {
	RecordID string  `json:"recordId"`
	Time     string  `json:"time"`
	Provider string  `json:"provider"`
	ID       int     `json:"id"`
	Level    int     `json:"level"`
	Computer string  `json:"computer"`
	Message  string  `json:"message"`
	XML      string  `json:"xml"`
	Data     []Field `json:"data"`
	Category string  `json:"category"`
	Meaning  string  `json:"meaning"`
	Limited  bool    `json:"limited"`
}
type Finding struct {
	StopCode   string   `json:"stopCode"`
	RecordID   string   `json:"recordId"`
	Time       string   `json:"time"`
	Kind       string   `json:"kind"`
	Conclusion string   `json:"conclusion"`
	NextSteps  []string `json:"nextSteps"`
	Evidence   []string `json:"evidence"`
}
type Summary struct {
	Total      int `json:"total"`
	Boot       int `json:"boot"`
	Requests   int `json:"requests"`
	Unexpected int `json:"unexpected"`
	Bugchecks  int `json:"bugchecks"`
}
type Report struct {
	Events      []Event   `json:"events"`
	Findings    []Finding `json:"findings"`
	Warnings    []string  `json:"warnings"`
	Truncated   bool      `json:"truncated"`
	Summary     Summary   `json:"summary"`
	CollectedAt string    `json:"collectedAt"`
	StartTime   string    `json:"startTime"`
}

func Read(ctx context.Context, days, limit int) (Report, error) {
	if days < 1 || days > 90 || limit < 50 || limit > 1000 {
		return Report{}, fmt.Errorf("日志范围须为 1–90 天，条数须为 50–1000")
	}
	out, err := syscmd.RunPSContext(ctx, fmt.Sprintf("$days=%d; $limit=%d\n%s", days, limit, collector))
	if err != nil {
		return Report{}, fmt.Errorf("读取系统日志失败: %w", err)
	}
	var report Report
	if err = json.Unmarshal([]byte(strings.TrimSpace(out)), &report); err != nil {
		return Report{}, fmt.Errorf("解析系统日志失败: %w", err)
	}
	return Analyze(report), nil
}

func classify(e Event) (string, string) {
	p := strings.ToLower(e.Provider)
	switch {
	case p == "microsoft-windows-kernel-power" && e.ID == 41:
		return "unexpected", "检测到未正常关机后的启动；此事件本身不能确定是断电、宿主机重置、蓝屏还是系统卡死。"
	case p == "eventlog" && e.ID == 6008:
		return "unexpected", "系统报告上次关机异常；记录时间通常在随后启动时，不能当作实际故障发生时间。"
	case (p == "microsoft-windows-wer-systemerrorreporting" || p == "bugcheck") && e.ID == 1001:
		return "bugcheck", "系统记录了停止错误或错误检查报告，请结合停止代码与内存转储分析。"
	case p == "user32" && e.ID == 1074:
		return "request", "已记录进程或用户发起的关机/重启请求；这不能单独证明操作正常完成。"
	case p == "microsoft-windows-kernel-general" && e.ID == 12:
		return "boot", "内核记录系统启动。"
	case p == "microsoft-windows-kernel-general" && e.ID == 13:
		return "shutdown", "内核记录系统正在关闭，不能单独证明关机完整完成。"
	case p == "eventlog" && e.ID == 6005:
		return "boot", "事件日志服务启动；不一定等同于操作系统启动。"
	case p == "eventlog" && e.ID == 6006:
		return "shutdown", "事件日志服务停止；需结合其他记录判断关机过程。"
	case p == "eventlog" && e.ID == 6009:
		return "boot", "启动附近的系统版本记录。"
	case p == "microsoft-windows-windowsupdateclient" && e.ID == 19:
		return "update", "更新安装记录；时间接近重启并不证明更新导致了异常。"
	case p == "microsoft-windows-whea-logger":
		return "hardware", "硬件错误报告；在 VPS 上还需结合宿主机和服务商记录排查。"
	case p == "volmgr" && (e.ID == 46 || e.ID == 161):
		return "dump", "内存转储初始化或写入异常，可能影响停止错误取证。"
	case p == "disk" || p == "ntfs" || p == "storahci" || p == "stornvme" || p == "storport":
		return "storage", "存储相关告警或错误；需结合磁盘、驱动及云存储状态检查。"
	case p == "service control manager" && (e.ID == 7031 || e.ID == 7034):
		return "service", "服务意外终止；应确认具体服务和恢复动作，不能直接认定为系统重启原因。"
	case p == "microsoft-windows-eventlog" && e.ID == 104:
		return "logclear", "日志清除记录，历史证据可能不完整。"
	case p == "microsoft-windows-kernel-general" && e.ID == 1:
		return "clock", "系统时间调整记录，可能影响事件时间顺序和关联。"
	default:
		return "other", "其他系统事件；需结合原始内容判断。"
	}
}
func field(e Event, name string) string {
	for _, f := range e.Data {
		if strings.EqualFold(f.Name, name) {
			return f.Value
		}
	}
	return ""
}

// Analyze observations, not an asserted root cause. Provider + ID are matched
// together; localized message text is never used as a classification rule.
func Analyze(r Report) Report {
	if r.Events == nil {
		r.Events = []Event{}
	}
	if r.Warnings == nil {
		r.Warnings = []string{}
	}
	r.Findings = []Finding{}
	r.Summary = Summary{Total: len(r.Events)}
	sort.SliceStable(r.Events, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, r.Events[i].Time)
		b, _ := time.Parse(time.RFC3339Nano, r.Events[j].Time)
		return a.After(b)
	})
	for i := range r.Events {
		e := &r.Events[i]
		e.Category, e.Meaning = classify(*e)
		switch e.Category {
		case "boot":
			r.Summary.Boot++
		case "request":
			r.Summary.Requests++
		case "unexpected":
			r.Summary.Unexpected++
		case "bugcheck":
			r.Summary.Bugchecks++
		}
		if e.Category == "logclear" || e.Category == "clock" {
			r.Warnings = append(r.Warnings, e.Meaning)
		}
		if e.Category != "unexpected" && e.Category != "bugcheck" && e.Category != "request" {
			continue
		}
		f := Finding{RecordID: e.RecordID, Time: e.Time, Kind: e.Category, Conclusion: e.Meaning, Evidence: []string{e.RecordID}, NextSteps: []string{}}
		if e.Category == "unexpected" {
			f.NextSteps = []string{"核对故障前后的服务商维护、宿主机重置、监控与控制台记录。", "检查同一时间段的停止错误、WHEA、存储与转储失败事件；缺少记录不能排除相应故障。"}
			code := field(*e, "BugcheckCode")
			base := 10
			if strings.HasPrefix(strings.ToLower(code), "0x") {
				base = 0
			}
			if value, err := strconv.ParseUint(code, base, 32); err == nil && value != 0 {
				f.StopCode = fmt.Sprintf("0x%08X", value)
			}
			if value := field(*e, "PowerButtonTimestamp"); value != "" && value != "0" {
				f.NextSteps = append(f.NextSteps, "记录包含电源按钮信息，需要核对控制台操作；不能仅据此确认断电原因。")
			}
		} else if e.Category == "bugcheck" {
			f.NextSteps = []string{"查看原始事件中的停止代码与转储路径，在 WinDbg 中分析转储。", "若缺少转储，检查 volmgr 事件、分页文件、磁盘空间和转储配置。"}
		} else {
			f.NextSteps = []string{"查看原始字段中的进程、用户、原因代码与注释，再核对后续关机和启动记录。"}
		}
		stamp, err := time.Parse(time.RFC3339Nano, e.Time)
		if err == nil {
			for _, other := range r.Events {
				if other.RecordID == e.RecordID {
					continue
				}
				at, err := time.Parse(time.RFC3339Nano, other.Time)
				if err == nil && at.Sub(stamp).Abs() <= 10*time.Minute && len(f.Evidence) < 16 {
					f.Evidence = append(f.Evidence, other.RecordID)
				}
			}
		}
		r.Findings = append(r.Findings, f)
	}
	r.Warnings = append(r.Warnings, "统计为事件条数，不是重启次数；41、6008、1001 可能描述同一次故障。前后十分钟的关联仅为排查线索。")
	if r.Truncated {
		r.Warnings = append(r.Warnings, "已达到条数上限，当前时间范围的记录可能未完整显示；请缩小范围或提高上限。")
	}
	// Repeated clock changes or log clears need one explanation, while every
	// original event remains available in the timeline and diagnostic export.
	seen := make(map[string]bool, len(r.Warnings))
	warnings := make([]string, 0, len(r.Warnings))
	for _, warning := range r.Warnings {
		if !seen[warning] {
			warnings = append(warnings, warning)
			seen[warning] = true
		}
	}
	r.Warnings = warnings
	return r
}
