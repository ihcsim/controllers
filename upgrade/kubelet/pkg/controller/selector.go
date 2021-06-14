package controller

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

var (
	labelControlPlane = "node-role.kubernetes.io/control-plane"
	labelMaster       = "node-role.kubernetes.io/master"
)

func excludeRequirements() ([]labels.Requirement, error) {
	excludeControlPlane, err := labels.NewRequirement(
		labelControlPlane, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}

	excludeMaster, err := labels.NewRequirement(
		labelMaster, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}

	return []labels.Requirement{
		*excludeControlPlane, *excludeMaster,
	}, nil
}
