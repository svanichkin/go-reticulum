//go:build !(linux || darwin || freebsd)

package rns

func loadExternalInterfacePlugin(interfacePath, ifType, name string, kv map[string]string) (*Interface, bool, error) {
	return nil, false, nil
}
