//go:build linux || darwin || freebsd

package rns

import (
	"fmt"
	"path/filepath"
	"plugin"
	"strings"
)

type externalInterfaceFactory func(name string, config map[string]string) (*Interface, error)

var externalInterfacePluginLoader = func(path string) (externalInterfaceFactory, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open external interface plugin %q: %w", path, err)
	}
	symbol, err := plug.Lookup("InterfaceFactory")
	if err != nil {
		return nil, fmt.Errorf("external interface plugin %q does not export InterfaceFactory: %w", path, err)
	}

	factory, ok := symbol.(func(string, map[string]string) (*Interface, error))
	if !ok {
		if typed, ok := symbol.(externalInterfaceFactory); ok {
			factory = func(n string, c map[string]string) (*Interface, error) { return typed(n, c) }
		} else {
			return nil, fmt.Errorf("external interface plugin %q has incompatible InterfaceFactory signature", path)
		}
	}
	return factory, nil
}

func loadExternalInterfacePlugin(interfacePath, ifType, name string, kv map[string]string) (*Interface, bool, error) {
	if strings.TrimSpace(interfacePath) == "" || strings.TrimSpace(ifType) == "" {
		return nil, false, nil
	}

	soPath := filepath.Join(interfacePath, ifType+".so")
	if !fileExists(soPath) {
		return nil, false, nil
	}

	factory, err := externalInterfacePluginLoader(soPath)
	if err != nil {
		return nil, true, err
	}

	ifc, err := factory(name, cloneInterfaceConfigMap(kv))
	if err != nil {
		return nil, true, err
	}
	if ifc == nil {
		return nil, true, fmt.Errorf("external interface plugin %q returned nil interface", soPath)
	}
	if strings.TrimSpace(ifc.Name) == "" {
		ifc.Name = strings.TrimSpace(name)
	}
	if strings.TrimSpace(ifc.Type) == "" {
		ifc.Type = strings.TrimSpace(ifType)
	}
	return ifc, true, nil
}
