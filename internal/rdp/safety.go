package rdp

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"wintoolbox/internal/win/syscmd"
)

//go:embed safety.ps1
var safetyScript string

type SafetyStatus struct {
	Session     Session      `json:"session"`
	Pending     bool         `json:"pending"`
	Transaction string       `json:"transaction"`
	Deadline    string       `json:"deadline"`
	Phase       string       `json:"phase"`
	Kind        string       `json:"kind"`
	Addresses   []string     `json:"addresses"`
	Port        uint32       `json:"port"`
	Listening   bool         `json:"listening"`
	Service     string       `json:"service"`
	Profiles    string       `json:"profiles"`
	Rules       []SafetyRule `json:"rules"`
	Warnings    []string     `json:"warnings"`
	CanConfirm  bool         `json:"canConfirm"`
}

type SafetyRule struct {
	Name      string `json:"name"`
	Action    string `json:"action"`
	Protocol  string `json:"protocol"`
	Ports     string `json:"ports"`
	Addresses string `json:"addresses"`
	Source    string `json:"source"`
	Profile   string `json:"profile"`
}

func NormalizeSources(input string) ([]string, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t' })
	if len(parts) == 0 || len(parts) > 64 {
		return nil, fmt.Errorf("请填写 1–64 个 IP 或 CIDR，不能使用 Any 或主机名")
	}
	result := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		if strings.Contains(part, "%") {
			return nil, fmt.Errorf("不支持带区域标识的 IPv6 地址: %s", part)
		}
		var p netip.Prefix
		var err error
		if strings.Contains(part, "/") {
			p, err = netip.ParsePrefix(part)
		} else {
			var a netip.Addr
			a, err = netip.ParseAddr(part)
			if err == nil {
				p = netip.PrefixFrom(a, a.BitLen())
			}
		}
		if err != nil || !p.IsValid() || p.Bits() == 0 || p.Addr().Is4In6() || p.Addr().Zone() != "" || p.Addr().IsUnspecified() || p.Addr().IsMulticast() {
			return nil, fmt.Errorf("无效或不安全的来源地址: %s", part)
		}
		value := p.Masked().String()
		if !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result, nil
}

// Scripts and durable state live under Program Files, never a user-writable TEMP.
// Refuse reparse points; newly created files inherit administrator/SYSTEM ACLs.
func safetyPath() (string, error) {
	base, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil {
		return "", err
	}
	root := filepath.Join(base, "WinToolBox-RdpSafety")
	if err = os.Mkdir(root, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return "", err
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return "", err
	}
	if attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return "", fmt.Errorf("拒绝使用重解析目录保存回滚任务")
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if err != nil {
		return "", err
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return "", err
	}
	if err = windows.SetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return "", err
	}
	name := fmt.Sprintf("safety-%x.ps1", sha256.Sum256([]byte(safetyScript)))
	path := filepath.Join(root, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		data, readErr := os.ReadFile(path)
		if readErr != nil || string(data) != safetyScript {
			return "", fmt.Errorf("回滚脚本完整性检查失败")
		}
		return path, nil
	}
	if err != nil {
		return "", err
	}
	_, writeErr := file.WriteString(safetyScript)
	closeErr := file.Close()
	if writeErr != nil {
		return "", writeErr
	}
	return path, closeErr
}

func psQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func runSafety(action, kind, value, transaction string) (string, error) {
	path, err := safetyPath()
	if err != nil {
		return "", err
	}
	s := CurrentSession()
	return syscmd.RunPS(fmt.Sprintf("& %s -Action %s -Kind %s -Value %s -Transaction %s -SessionID %d -RemoteSession %s", psQuote(path), psQuote(action), psQuote(kind), psQuote(value), psQuote(transaction), s.ID, strings.ToLower(fmt.Sprint(s.Known && s.Remote))))
}

func GetSafetyStatus() (SafetyStatus, error) {
	out, err := runSafety("Status", "", "", "")
	var status SafetyStatus
	if err == nil {
		err = json.Unmarshal([]byte(strings.TrimSpace(out)), &status)
	}
	status.Session = CurrentSession()
	return status, err
}

func ApplyProtected(kind, value string) error {
	s := CurrentSession()
	if !s.Known {
		return fmt.Errorf("无法识别当前会话，已中止远程配置修改")
	}
	if kind == "whitelist" {
		addresses, err := NormalizeSources(value)
		if err != nil {
			return err
		}
		value = strings.Join(addresses, ",")
	}
	_, err := runSafety("Apply", kind, value, "")
	return err
}

func ConfirmSafety(transaction string) error {
	if transaction == "" {
		return fmt.Errorf("缺少回滚事务标识")
	}
	_, err := runSafety("Confirm", "", "", transaction)
	return err
}

func RollbackSafety(transaction string) error {
	if transaction == "" {
		return fmt.Errorf("缺少回滚事务标识")
	}
	_, err := runSafety("Rollback", "", "", transaction)
	return err
}

// Existing unrelated network mutations must not race a protected transaction.
func GuardNetwork(consoleOnly bool) error {
	if consoleOnly {
		if err := RequireConsole(); err != nil {
			return err
		}
	}
	_, err := runSafety("Guard", "", "", "")
	return err
}
