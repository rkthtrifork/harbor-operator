package controller

import (
	"strings"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type operatorOptions struct {
	forcedHarborConnection string
	secretReader           client.Reader
}

var (
	operatorOptsMu sync.RWMutex
	operatorOpts   operatorOptions
)

func SetForcedHarborConnection(name string) {
	operatorOptsMu.Lock()
	defer operatorOptsMu.Unlock()
	operatorOpts.forcedHarborConnection = strings.TrimSpace(name)
}

func ForcedHarborConnection() string {
	operatorOptsMu.RLock()
	defer operatorOptsMu.RUnlock()
	return operatorOpts.forcedHarborConnection
}

func SetSecretReader(reader client.Reader) {
	operatorOptsMu.Lock()
	defer operatorOptsMu.Unlock()
	operatorOpts.secretReader = reader
}

func SecretReader() client.Reader {
	operatorOptsMu.RLock()
	defer operatorOptsMu.RUnlock()
	return operatorOpts.secretReader
}
