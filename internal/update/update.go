package update

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"

	"wintoolbox/internal/win/snapshot"
	"wintoolbox/internal/win/syscmd"
)

var updateServices = []string{
	"wuauserv",
	"UsoSvc",
	"WaaSMedicSvc",
	"uhssvc",
	"DoSvc",
}

// defaultStartType is used when restoring Windows Update services without a snapshot.
var defaultStartType = map[string]string{
	"wuauserv":     "auto",
	"UsoSvc":       "auto",
	"WaaSMedicSvc": "demand",
	"uhssvc":       "demand",
	"DoSvc":        "auto",
}

var updateTaskNames = []string{
	`\Microsoft\Windows\WindowsUpdate\Scheduled Start`,
	`\Microsoft\Windows\WindowsUpdate\sih`,
	`\Microsoft\Windows\UpdateOrchestrator\Schedule Scan`,
	`\Microsoft\Windows\UpdateOrchestrator\UpdateModelTask`,
	`\Microsoft\Windows\UpdateOrchestrator\USO_UxBroker`,
	`\Microsoft\Windows\UpdateOrchestrator\Backup Scan`,
}

const (
	auKeyPath     = `SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`
	wuKeyPath     = `SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`
	backupKeyPath = `SOFTWARE\WinToolbox\UpdateBackup`
)

// Status describes whether automatic updates appear disabled.
type Status struct {
	Disabled  bool
	Unknown   bool
	HasBackup bool
	Detail    string
}

// GetStatus inspects policy registry and key update services.
func GetStatus() Status { return GetStatusContext(context.Background()) }

func GetStatusContext(ctx context.Context) Status {
	noAuto, hasPolicy := readNoAutoUpdate()
	types := make([]string, len(updateServices))
	var wg sync.WaitGroup
	wg.Add(len(updateServices))
	for i, name := range updateServices {
		i, name := i, name
		go func() {
			defer wg.Done()
			types[i] = serviceStartTypeContext(ctx, name)
		}()
	}
	wg.Wait()

	disabledCount := 0
	presentCount := 0
	parts := make([]string, 0, len(updateServices))
	for i, name := range updateServices {
		t := types[i]
		if t == "n/a" {
			continue
		}
		presentCount++
		parts = append(parts, fmt.Sprintf("%s=%s", name, t))
		if t == "disabled" {
			disabledCount++
		}
	}

	policyOff := hasPolicy && noAuto
	// Treat as disabled when policy is off, or core update services are disabled
	// (so the UI "恢复" button stays available even if only services were flipped).
	coreOff := false
	for i, name := range updateServices {
		if name != "wuauserv" && name != "UsoSvc" {
			continue
		}
		if types[i] == "disabled" {
			coreOff = true
			break
		}
	}
	majorityOff := presentCount > 0 && disabledCount*2 >= presentCount
	disabled := policyOff || coreOff || majorityOff

	detail := fmt.Sprintf("策略关闭=%v · 相关服务已禁用 %d/%d", policyOff, disabledCount, presentCount)
	if len(parts) > 0 {
		detail += "（" + strings.Join(parts, ", ") + "）"
	}
	unknown := false
	for _, t := range types {
		if t == "unknown" {
			unknown = true
		}
	}
	if _, _, err := snapshot.ReadOptional(auKeyPath, "NoAutoUpdate"); err != nil {
		unknown = true
	}
	return Status{Disabled: disabled, Unknown: unknown, HasBackup: backupExists(), Detail: detail}
}

// Disable turns off Windows automatic updates via policy, then best-effort service stops.
// Previous policy/service start values are snapshotted once for Enable() restore.
func Disable() error {
	if err := saveUpdateSnapshotIfNeeded(); err != nil {
		return fmt.Errorf("保存更新配置备份失败，已中止关闭: %w", err)
	}

	if err := setUpdatePolicyDisabled(); err != nil {
		return fmt.Errorf("写入更新策略失败: %w", err)
	}

	for _, name := range updateServices {
		_, _ = syscmd.Run("sc", "stop", name)
		out, err := syscmd.Run("sc", "config", name, "start=", "disabled")
		if err != nil {
			msg := strings.TrimSpace(out)
			if msg == "" {
				msg = err.Error()
			}
			if serviceMissing(msg) || accessDenied(msg) {
				_ = setServiceStartDWORD(name, 4)
				continue
			}
			_ = setServiceStartDWORD(name, 4)
		}
	}

	setUpdateTasks(false)
	st := GetStatus()
	if st.Unknown || !st.Disabled {
		return fmt.Errorf("已提交关闭更新，但复查仍显示运行中: %s", st.Detail)
	}
	return nil
}

// Enable restores Windows Update from the Disable() snapshot when available;
// otherwise clears WinToolbox policy keys and applies default service start types.
func Enable() error {
	restored, err := restoreUpdateSnapshot()
	if err != nil {
		return fmt.Errorf("按关闭前配置备份恢复更新失败: %w", err)
	}
	if !restored {
		if err := clearUpdatePolicy(); err != nil {
			return fmt.Errorf("清除更新限制策略失败: %w", err)
		}
		for _, name := range updateServices {
			if err := applyServiceStart(name, defaultStartType[name]); err != nil {
				return fmt.Errorf("恢复服务 %s 失败: %w", name, err)
			}
		}
		setUpdateTasks(true)
	}

	if !restored {
		for _, name := range []string{"wuauserv", "UsoSvc", "DoSvc"} {
			_, _ = syscmd.Run("sc", "start", name)
		}
	}

	st := GetStatus()
	if !restored && (st.Disabled || st.Unknown) {
		return fmt.Errorf("已提交恢复更新，但复查仍显示已关闭: %s", st.Detail)
	}
	if restored {
		return clearBackup()
	}
	return nil
}

func saveUpdateSnapshotIfNeeded() error {
	if backupExists() {
		return nil
	}
	values := map[string]uint32{}
	for _, name := range []string{"NoAutoUpdate", "AUOptions"} {
		v, present, err := snapshot.ReadOptional(auKeyPath, name)
		if err != nil {
			return fmt.Errorf("读取 %s: %w", name, err)
		}
		if present {
			values["Had"+name] = 1
		} else {
			values["Had"+name] = 0
		}
		values[name] = v
	}
	for _, name := range updateServices {
		v, present, err := snapshot.ReadOptional(`SYSTEM\CurrentControlSet\Services\`+name, "Start")
		if err != nil {
			return err
		}
		if present {
			values["HadSvc_"+name] = 1
		} else {
			values["HadSvc_"+name] = 0
		}
		values["Svc_"+name] = v
	}
	for i, tn := range updateTaskNames {
		state, err := readTaskState(tn)
		if err != nil {
			return fmt.Errorf("读取任务 %s 失败，已中止关闭: %w", tn, err)
		}
		values["Task_"+strconv.Itoa(i)] = state
	}

	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, backupKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return snapshot.Commit(k, values)
}

func backupExists() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, backupKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("Saved")
	return err == nil && v == 1
}

func restoreUpdateSnapshot() (bool, error) {
	fields := []string{"HadNoAutoUpdate", "HadAUOptions"}
	for _, name := range updateServices {
		fields = append(fields, "HadSvc_"+name)
	}
	for i := range updateTaskNames {
		fields = append(fields, "Task_"+strconv.Itoa(i))
	}
	values, exists, err := snapshot.Load(backupKeyPath, fields)
	if err != nil || !exists {
		return false, err
	}
	// Validate conditional fields as well; legacy snapshots omit absent values.
	extra := append([]string{}, fields...)
	for _, name := range []string{"NoAutoUpdate", "AUOptions"} {
		if values["Had"+name] == 1 {
			extra = append(extra, name)
		}
	}
	for _, name := range updateServices {
		if values["HadSvc_"+name] == 1 {
			extra = append(extra, "Svc_"+name)
		}
	}
	values, _, err = snapshot.Load(backupKeyPath, extra)
	if err != nil {
		return false, err
	}
	if err := validateSnapshot(values); err != nil {
		return false, err
	}
	for _, name := range []string{"NoAutoUpdate", "AUOptions"} {
		if values["Had"+name] == 1 {
			err = writeAUDword(name, values[name])
		} else {
			err = deleteAUValue(name)
		}
		if err != nil {
			return false, err
		}
		v, present, err := snapshot.ReadOptional(auKeyPath, name)
		if err != nil || present != (values["Had"+name] == 1) || (present && v != values[name]) {
			return false, fmt.Errorf("恢复策略 %s 后复查不一致: %v", name, err)
		}
	}
	for _, name := range updateServices {
		if values["HadSvc_"+name] != 1 {
			continue
		}
		start := values["Svc_"+name]
		if start > 4 {
			return false, fmt.Errorf("服务 %s 备份启动值无效", name)
		}
		if err := applyServiceStart(name, dwordToScStart(start)); err != nil {
			return false, err
		}
		actual, present, err := snapshot.ReadOptional(`SYSTEM\CurrentControlSet\Services\`+name, "Start")
		if err != nil || !present || actual != start {
			return false, fmt.Errorf("服务 %s 恢复后复查不一致", name)
		}
	}
	for i, tn := range updateTaskNames {
		v := values["Task_"+strconv.Itoa(i)]
		if v == 2 {
			continue
		} // Preserve unknown/missing tasks rather than inventing a state.
		if v > 2 {
			return false, fmt.Errorf("任务备份状态无效: %s", tn)
		}
		flag := "/Disable"
		if v == 1 {
			flag = "/Enable"
		}
		if _, err := syscmd.Run("schtasks", "/Change", "/TN", tn, flag); err != nil {
			return false, err
		}
		enabled, ok := taskEnabled(tn)
		if !ok || enabled != (v == 1) {
			return false, fmt.Errorf("任务恢复后复查不一致: %s", tn)
		}
	}
	return true, nil
}

func applyServiceStart(name, startType string) error {
	if startType == "" {
		startType = "demand"
	}
	out, err := syscmd.Run("sc", "config", name, "start=", startType)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		if serviceMissing(msg) {
			return nil
		}
		dword := uint32(3)
		switch startType {
		case "auto":
			dword = 2
		case "disabled":
			dword = 4
		}
		if err2 := setServiceStartDWORD(name, dword); err2 != nil {
			return fmt.Errorf("%s; 注册表回退: %v", msg, err2)
		}
	}
	return nil
}

func clearBackup() error {
	return registry.DeleteKey(registry.LOCAL_MACHINE, backupKeyPath)
}

func dwordToScStart(v uint32) string {
	switch v {
	case 2:
		return "auto"
	case 3:
		return "demand"
	case 4:
		return "disabled"
	default:
		return "demand"
	}
}

func writeAUDword(name string, value uint32) error {
	wuKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, wuKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	wuKey.Close()
	auKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, auKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer auKey.Close()
	return auKey.SetDWordValue(name, value)
}

func deleteAUValue(name string) error {
	auKey, err := registry.OpenKey(registry.LOCAL_MACHINE, auKeyPath, registry.SET_VALUE)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return err
	}
	defer auKey.Close()
	if err := auKey.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

func readAUOptions() (uint32, bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, auKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AUOptions")
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func readServiceStartDWORD(service string) (uint32, bool) {
	path := `SYSTEM\CurrentControlSet\Services\` + service
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("Start")
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func setServiceStartDWORD(service string, start uint32) error {
	path := `SYSTEM\CurrentControlSet\Services\` + service
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetDWordValue("Start", start)
}

func setUpdatePolicyDisabled() error {
	wuKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, wuKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	wuKey.Close()

	auKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, auKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer auKey.Close()

	if err := auKey.SetDWordValue("NoAutoUpdate", 1); err != nil {
		return err
	}
	if err := auKey.SetDWordValue("AUOptions", 1); err != nil {
		return err
	}
	return nil
}

func clearUpdatePolicy() error {
	auKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, auKeyPath, registry.SET_VALUE)
	if err != nil {
		if _, err2 := registry.OpenKey(registry.LOCAL_MACHINE, auKeyPath, registry.QUERY_VALUE); err2 != nil {
			return nil
		}
		return err
	}
	defer auKey.Close()
	if err := auKey.DeleteValue("NoAutoUpdate"); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("清除 NoAutoUpdate 失败: %w", err)
	}
	if err := auKey.DeleteValue("AUOptions"); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("清除 AUOptions 失败: %w", err)
	}
	return nil
}

func setUpdateTasks(enable bool) {
	flag := "/Disable"
	if enable {
		flag = "/Enable"
	}
	for _, task := range updateTaskNames {
		_, _ = syscmd.Run("schtasks", "/Change", "/TN", task, flag)
	}
}

func taskEnabled(tn string) (enabled bool, ok bool) {
	state, err := readTaskState(tn)
	return state == 1, err == nil && state != 2
}

// Query the scheduler COM API, independent of localized schtasks output.
// 2 is a confirmed missing task; permission/transport failures return an error.
func readTaskState(tn string) (uint32, error) {
	index := strings.LastIndex(tn, `\`)
	if index < 0 || index == len(tn)-1 {
		return 0, fmt.Errorf("任务名称无效")
	}
	folder, name := tn[:index], tn[index+1:]
	if folder == "" {
		folder = `\`
	}
	quote := func(v string) string { return "'" + strings.ReplaceAll(v, "'", "''") + "'" }
	script := fmt.Sprintf(`
$ErrorActionPreference='Stop'
try {
 $scheduler=New-Object -ComObject 'Schedule.Service'
 $scheduler.Connect()
 $task=$scheduler.GetFolder(%s).GetTask(%s)
 if ($task.Enabled) { Write-Output 'TASK=1' } else { Write-Output 'TASK=0' }
} catch {
 $exception=$_.Exception
 while ($exception.InnerException) { $exception=$exception.InnerException }
 if ($exception.HResult -eq -2147024894 -or $exception.HResult -eq -2147024893) { Write-Output 'TASK=2' } else { throw }
}
`, quote(folder), quote(name))
	out, err := syscmd.RunPSQuick(script)
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		switch strings.TrimSpace(line) {
		case "TASK=0":
			return 0, nil
		case "TASK=1":
			return 1, nil
		case "TASK=2":
			return 2, nil
		}
	}
	return 0, fmt.Errorf("无法解析任务状态: %s", tn)
}

func serviceMissing(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "1060") ||
		strings.Contains(msg, "指定的服务未安装") ||
		strings.Contains(lower, "does not exist")
}

func accessDenied(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "access is denied") ||
		strings.Contains(msg, "拒绝访问") ||
		strings.Contains(msg, "FAILED 5") ||
		strings.Contains(lower, "openService failed 5")
}

func readNoAutoUpdate() (value bool, ok bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, auKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("NoAutoUpdate")
	if err != nil {
		return false, false
	}
	return v == 1, true
}

func serviceStartType(name string) string { return serviceStartTypeContext(context.Background(), name) }

func serviceStartTypeContext(ctx context.Context, name string) string {
	if ctx.Err() != nil {
		return "unknown"
	}
	value, present, err := snapshot.ReadOptional(`SYSTEM\CurrentControlSet\Services\`+name, "Start")
	if err != nil {
		return "unknown"
	}
	if !present {
		return "n/a"
	}
	switch value {
	case 2:
		return "auto"
	case 3:
		return "manual"
	case 4:
		return "disabled"
	case 0:
		return "boot"
	case 1:
		return "system"
	default:
		return "unknown"
	}
}

func validateSnapshot(values map[string]uint32) error {
	for key, value := range values {
		switch {
		case strings.HasPrefix(key, "Had") && value > 1:
			return fmt.Errorf("备份存在标志无效: %s", key)
		case strings.HasPrefix(key, "Svc_") && value > 4:
			return fmt.Errorf("备份服务启动值无效: %s", key)
		case strings.HasPrefix(key, "Task_") && value > 2:
			return fmt.Errorf("备份任务状态无效: %s", key)
		}
	}
	return nil
}
