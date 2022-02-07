# Home Assignment

When operating a large multi-tenant Kubernetes cluster, tenants are usually isolated by Namespaces and Role Base Acess Control (RBAC).
This approach limits the permissions tenant have on the Namespace object they use to deploy their applications.
Some tenants would like to set specific labels on their Namespace; however, they cannot edit it.
As operators, we came up with the idea of creating a Custom Resouce Definition (CRD), which will allow tenants to edit their
Namespace's labels.

`Please make sure you have a basic understanding of the following concepts before you continue to read.`
- [Controller](https://kubernetes.io/docs/concepts/architecture/controller/) 
- [Custom Resouce Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) 
- [Kubebuilder](https://book.kubebuilder.io)

## NamespaceLabel Operator

This operator should be reasonably straightforward. It should sync between the NamespaceLabel CRD and the Namespace Labels.
Various ways could achieve this functionality. Please go ahead and get creative. However, even a simple working solution will be welcomed!

We have already generated a CRD and a controller for you for your convenience.
The controller can be found at `./controllers/namespacelabel_controller.go` which you should implement,
and the CRD definition at `./api/v1alpha1/namespacelabel_types.go`.
Please feel free to change the CRD or create more controllers as you wish as long you achieve the operator's goal.

```
apiVersion: dana.io.dana.io/v1alpha1
kind: NamespaceLabel
metadata:
    name: namespacelabel-sample
    namespace: default
spec:
    labels:
        label_1: a
        label_2: b
        label_3: c
```
`YAML file could be found at ./config/smaples/dana.io_v1alpha1_namespacelabel.yaml
`

### Questions You Will Probably Will Be Asked

- Can you create/update/delete labels?
- Can you deal with more than one NamespaceLabel object per Namespace?
- Namespaces usually has labels for managment can you protect those labels?
- Tenant is not able to consum CRDs by default, what did you do to let tenant use the NamespaceLabel CRD?

## Tools you should use
This repo contains a go project you can fork it and use it as a template, also you will need:
- [Kind](https://kind.sigs.k8s.io)  for creating local cluster
- [Go](https://go.dev) your operator should be written in Go
- [Kubebuilder](https://book.kubebuilder.io) for creating the operator and crd template