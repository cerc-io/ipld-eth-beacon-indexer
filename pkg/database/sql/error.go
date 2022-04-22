package sql

import (
	"fmt"
)

const (
	DbConnectionFailedMsg = "db connection failed"
	SettingNodeFailedMsg  = "unable to set db node"
)

func ErrDBConnectionFailed(connectErr error) error {
	return formatError(DbConnectionFailedMsg, connectErr.Error())
}

func ErrUnableToSetNode(setErr error) error {
	return formatError(SettingNodeFailedMsg, setErr.Error())
}

func formatError(msg, err string) error {
	return fmt.Errorf("%s: %s", msg, err)
}
