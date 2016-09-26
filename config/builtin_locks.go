package config

import (
	"fmt"

	"github.com/gruntwork-io/terragrunt/locks"
	"github.com/gruntwork-io/terragrunt/locks/dynamodb"
)

// lockFactory provides an implementation of Lock with the provided configuration map
type lockFactory func(map[string]string) (locks.Lock, error)

// lookupLock returns the implementation for the named lock or returns an error if
// it is not found
func lookupLock(name string, conf map[string]string) (locks.Lock, error) {
	f, ok := builtinLocks[name]
	if !ok {
		return nil, fmt.Errorf("no Lock implementation found for %s", name)
	}

	return f(conf)
}

var builtinLocks = map[string]lockFactory{
	"dynamodb": dynamodb.New,
}
