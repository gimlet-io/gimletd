package kustomize

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

func ApplyPatches(patch string, templatesManifests string) (string, error) {
	bytePatch := []byte(fmt.Sprintf("%v", patch))

	fSys := filesys.MakeFsInMemory()
	err := fSys.WriteFile("manifests.yaml", []byte(templatesManifests))
	if err != nil {
		return "", err
	}

	err = fSys.WriteFile("patches.yaml", bytePatch)
	if err != nil {
		return "", err
	}

	err = fSys.WriteFile("kustomization.yaml", []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- manifests.yaml
patchesStrategicMerge:
- patches.yaml
`))
	if err != nil {
		return "", err
	}

	fmt.Println(templatesManifests)
	b := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resources, err := b.Run(fSys, ".")
	if err != nil {
		return "", err
	}

	var files []byte
	for _, res := range resources.Resources() {
		yaml, err := res.AsYAML()
		if err != nil {
			return "", err
		}
		delimiter := []byte("---\n")
		files = append(files, delimiter...)
		files = append(files, yaml...)

	}

	return string(files), err
}
