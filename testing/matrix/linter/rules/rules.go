package rules

import (
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/deckhouse/deckhouse/testing/matrix/linter/rules/errors"
	"github.com/deckhouse/deckhouse/testing/matrix/linter/rules/resources"
	"github.com/deckhouse/deckhouse/testing/matrix/linter/rules/roles"
	"github.com/deckhouse/deckhouse/testing/matrix/linter/storage"
	"github.com/deckhouse/deckhouse/testing/matrix/linter/types"
)

func applyContainerRules(lintRuleErrorsList *errors.LintRuleErrorsList, object storage.StoreObject) {
	containers, err := object.GetContainers()
	if err != nil {
		panic(err)
	}
	if len(containers) == 0 {
		return
	}

	lintRuleErrorsList.Add(containerNameDuplicates(object, containers))
	lintRuleErrorsList.Add(containerEnvVariablesDuplicates(object, containers))
	lintRuleErrorsList.Add(containerImageTagLatest(object, containers))
	lintRuleErrorsList.Add(containerImagePullPolicyIfNotPresent(object, containers))
}

func containerNameDuplicates(object storage.StoreObject, containers []v1.Container) errors.LintRuleError {
	names := make(map[string]struct{})
	for _, c := range containers {
		if _, ok := names[c.Name]; ok {
			return errors.NewLintRuleError("CONTAINER001", object.Identity(), c.Name, "Duplicate container name")
		}
		names[c.Name] = struct{}{}
	}
	return errors.EmptyRuleError
}

func containerEnvVariablesDuplicates(object storage.StoreObject, containers []v1.Container) errors.LintRuleError {
	for _, c := range containers {
		envVariables := make(map[string]struct{})
		for _, variable := range c.Env {
			if _, ok := envVariables[variable.Name]; ok {
				return errors.NewLintRuleError(
					"CONTAINER002",
					object.Identity()+"; container = "+c.Name,
					variable.Name,
					"Container has two env variables with same name",
				)
			}
			envVariables[variable.Name] = struct{}{}
		}
	}
	return errors.EmptyRuleError
}

func containerImageTagLatest(object storage.StoreObject, containers []v1.Container) errors.LintRuleError {
	for _, c := range containers {
		imageParts := strings.Split(c.Image, ":")
		if len(imageParts) != 2 {
			return errors.NewLintRuleError(
				"CONTAINER003",
				object.Identity()+"; container = "+c.Name,
				nil,
				"Can't parse an image for container",
			)
		}
		if imageParts[1] == "latest" {
			return errors.NewLintRuleError("CONTAINER003",
				object.Identity()+"; container = "+c.Name,
				nil,
				"Image tag \"latest\" used",
			)
		}
	}
	return errors.EmptyRuleError
}

func containerImagePullPolicyIfNotPresent(object storage.StoreObject, containers []v1.Container) errors.LintRuleError {
	for _, c := range containers {
		if c.ImagePullPolicy == "" || c.ImagePullPolicy == "IfNotPresent" {
			continue
		}
		return errors.NewLintRuleError(
			"CONTAINER004",
			object.Identity()+"; container = "+c.Name,
			c.ImagePullPolicy,
			"container imagePullPolicy should be unspecified or \"IfNotPresent\"",
		)
	}
	return errors.EmptyRuleError
}

func applyObjectRules(objectStore *storage.UnstructuredObjectStore, lintRuleErrorsList *errors.LintRuleErrorsList, module types.Module, object storage.StoreObject) {
	lintRuleErrorsList.Add(objectRecommendedLabels(object))
	lintRuleErrorsList.Add(objectAPIVersion(object))
	lintRuleErrorsList.Add(roles.ObjectUserAuthzClusterRolePath(module, object))
	lintRuleErrorsList.Add(roles.ObjectDeckhouseClusterRoles(module, object))
	lintRuleErrorsList.Add(roles.ObjectRBACPlacement(module, object))
	lintRuleErrorsList.Add(roles.ObjectBindingSubjectServiceAccountCheck(module, object, objectStore))
}

func objectRecommendedLabels(object storage.StoreObject) errors.LintRuleError {
	labels := object.Unstructured.GetLabels()
	if _, ok := labels["module"]; !ok {
		return errors.NewLintRuleError(
			"MANIFEST001",
			object.Identity(),
			labels,
			"Object does not have the label \"module\"",
		)
	}
	if _, ok := labels["heritage"]; !ok {
		return errors.NewLintRuleError(
			"MANIFEST001",
			object.Identity(),
			labels,
			"Object does not have the label \"heritage\"",
		)
	}
	return errors.EmptyRuleError
}

func newAPIVersionError(wanted, version, objectID string) errors.LintRuleError {
	if version != wanted {
		return errors.NewLintRuleError(
			"MANIFEST002",
			objectID,
			version,
			"Object defined using deprecated api version, wanted %q", wanted,
		)
	}
	return errors.EmptyRuleError
}

func objectAPIVersion(object storage.StoreObject) errors.LintRuleError {
	kind := object.Unstructured.GetKind()
	version := object.Unstructured.GetAPIVersion()

	switch kind {
	case "Role", "RoleBinding", "ClusterRole", "ClusterRoleBinding":
		return newAPIVersionError("rbac.authorization.k8s.io/v1", version, object.Identity())
	case "Deployment", "DaemonSet", "StatefulSet":
		return newAPIVersionError("apps/v1", version, object.Identity())
	case "Ingress":
		return newAPIVersionError("networking.k8s.io/v1beta1", version, object.Identity())
	case "PriorityClass":
		return newAPIVersionError("scheduling.k8s.io/v1", version, object.Identity())
	case "PodSecurityPolicy":
		return newAPIVersionError("policy/v1beta1", version, object.Identity())
	case "NetworkPolicy":
		return newAPIVersionError("networking.k8s.io/v1", version, object.Identity())
	default:
		return errors.EmptyRuleError
	}
}

func ApplyLintRules(m types.Module, values string, objectStore *storage.UnstructuredObjectStore) error {
	var lintRuleErrorsList errors.LintRuleErrorsList
	for _, o := range objectStore.Storage {
		applyObjectRules(objectStore, &lintRuleErrorsList, m, o)
		applyContainerRules(&lintRuleErrorsList, o)
	}

	resources.ControllerMustHasVPA(m, values, objectStore, &lintRuleErrorsList)
	return lintRuleErrorsList.ConvertToError()
}
