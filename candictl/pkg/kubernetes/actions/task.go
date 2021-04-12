package actions

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/deckhouse/deckhouse/candictl/pkg/log"
)

type ManifestTask struct {
	Name       string
	CreateFunc func(manifest interface{}) error
	UpdateFunc func(manifest interface{}) error
	Manifest   func() interface{}
	PatchData  func() interface{}
	PatchFunc  func(patchData []byte) error
}

// CreateOrUpdate tries to create resource with the CreateFunc. If resource is already
// exists, it updates the resource with the UpdateFunc.
func (task *ManifestTask) CreateOrUpdate() error {
	log.InfoF("Manifest for %s\n", task.Name)
	manifest := task.Manifest()

	err := task.CreateFunc(manifest)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create resource: %v", err)
		}
		log.InfoF("%s already exists. Trying to update ... ", task.Name)
		err = task.UpdateFunc(manifest)
		if err != nil {
			log.ErrorLn("ERROR!")
			return fmt.Errorf("update resource: %v", err)
		}
		log.InfoLn("OK!")
	}
	return nil
}

func (task *ManifestTask) Patch() error {
	log.DebugF("Patch for %s\n", task.Name)
	patchData := task.PatchData()

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("marshal patch data: %v", err)
	}

	err = task.PatchFunc(patchBytes)
	if err != nil {
		return fmt.Errorf("Apply patch: %v", err)
	}

	return nil
}

func (task *ManifestTask) PatchOrCreate() error {
	log.DebugF("Patch or create for %s\n", task.Name)
	patchData := task.PatchData()

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("marshal patch data: %v", err)
	}

	err = task.PatchFunc(patchBytes)
	if err == nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("Apply patch for '%s': %v", task.Name, err)
	}

	log.DebugF("%s is not found. Trying to create ... \n", task.Name)
	manifest := task.Manifest()
	err = task.CreateFunc(manifest)
	if err != nil {
		return fmt.Errorf("Create '%s': %v", task.Name, err)
	}
	return nil
}
