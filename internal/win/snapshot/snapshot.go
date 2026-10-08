// Package snapshot provides strict, complete registry configuration snapshots.
package snapshot

import (
	"fmt"
	"sort"

	"golang.org/x/sys/windows/registry"
)

type Writer interface{ SetDWordValue(string, uint32) error }

// Commit never publishes Saved unless every field was persisted successfully.
func Commit(w Writer, values map[string]uint32) error {
	if err := w.SetDWordValue("Saved", 0); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := w.SetDWordValue(key, values[key]); err != nil {
			return fmt.Errorf("备份字段 %s: %w", key, err)
		}
	}
	return w.SetDWordValue("Saved", 1)
}

// ReadOptional distinguishes an absent value from an unreadable value.
func ReadOptional(path, name string) (uint32, bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err == registry.ErrNotExist {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(name)
	if err == registry.ErrNotExist {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uint32(v), true, nil
}

// Load reads every expected field before the caller may restore any setting.
func Load(path string, fields []string) (map[string]uint32, bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err == registry.ErrNotExist {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer k.Close()
	saved, _, err := k.GetIntegerValue("Saved")
	if err != nil {
		return nil, false, fmt.Errorf("备份完整性标记缺失: %w", err)
	}
	if saved != 1 {
		return nil, false, fmt.Errorf("备份未完整保存，请勿恢复")
	}
	values := make(map[string]uint32, len(fields))
	for _, field := range fields {
		v, _, err := k.GetIntegerValue(field)
		if err != nil {
			return nil, false, fmt.Errorf("备份字段 %s 缺失或不可读: %w", field, err)
		}
		values[field] = uint32(v)
	}
	return values, true, nil
}
