package labels

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

var (
	KeyNodeRoleControlPlane = "node-role.kubernetes.io/control-plane"
	KeyNodeRoleMaster       = "node-role.kubernetes.io/master"
	KeyKubeletUpgradeSkip   = "clusterop.isim.dev/skip"
)

func ExcludedNodes() (labels.Requirements, error) {
	excludeControlPlane, err := labels.NewRequirement(
		KeyNodeRoleControlPlane, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}

	excludeMaster, err := labels.NewRequirement(
		KeyNodeRoleMaster, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}

	return labels.Requirements{
		*excludeControlPlane,
		*excludeMaster,
	}, nil
}

func ExcludedKubeletUpgrade() (labels.Requirements, error) {
	skip, err := labels.NewRequirement(
		KeyKubeletUpgradeSkip, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}

	return labels.Requirements{*skip}, nil
}
