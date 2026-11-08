package rdp

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// IsNLAEnabled reports whether Network Level Authentication is required.
func IsNLAEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, rdpKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("读取 NLA 失败: %w", err)
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("UserAuthentication")
	if err != nil {
		return false, fmt.Errorf("读取 UserAuthentication 失败: %w", err)
	}
	return v == 1, nil
}
