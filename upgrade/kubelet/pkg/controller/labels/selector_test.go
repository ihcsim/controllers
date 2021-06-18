package labels

import (
	"fmt"
	"testing"
)

func TestExcludedNodes(t *testing.T) {
	actualRequirements, err := ExcludedNodes()
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	expectedRequirements := []string{
		fmt.Sprintf("!%s", KeyNodeRoleControlPlane),
		fmt.Sprintf("!%s", KeyNodeRoleMaster),
	}

	for _, expected := range expectedRequirements {
		var found bool
		for _, actual := range actualRequirements {
			if expected == actual.String() {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected requirement %s to exist", expected)
		}
	}
}

func TestExcludedKubeletUpgrade(t *testing.T) {
	actualRequirements, err := ExcludedKubeletUpgrade()
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	expectedRequirements := []string{
		fmt.Sprintf("!%s", KeyKubeletUpgradeSkip),
	}

	for _, expected := range expectedRequirements {
		var found bool
		for _, actual := range actualRequirements {
			if expected == actual.String() {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected requirement %s to exist", expected)
		}
	}
}
