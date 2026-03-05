package domain

import "errors"

var (
	ErrPlanNotFound         = errors.New("vps plan not found")
	ErrNodeNotFound         = errors.New("node not found")
	ErrNoAvailableNode      = errors.New("no node has sufficient resources for this plan")
	ErrInstanceNotFound     = errors.New("instance not found")
	ErrInstanceNotOwned     = errors.New("instance does not belong to user")
	ErrInstanceNotRunning   = errors.New("instance is not running")
	ErrInstanceNotSuspended = errors.New("instance is not suspended")
	ErrInstanceTerminated   = errors.New("instance is already terminated")
	ErrInstanceNotStoppable = errors.New("instance cannot be stopped in current status")
	ErrSnapshotNotFound     = errors.New("snapshot not found")
	ErrProvisionTimeout     = errors.New("vm provisioning timed out")
)
