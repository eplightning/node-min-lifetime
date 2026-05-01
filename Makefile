.PHONY: k8smanifests
k8smanifests:
	kubectl kustomize config/default > manifests/default.yaml