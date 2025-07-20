resource "helm_release" "cert-manager" {
  name       = "cert-manager"
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  namespace  = "cert-manager"
  version    = "v1.18.2"
  
  create_namespace = true
  
  values = [
    file("../helm/cert-manager/values.yaml")
  ]
}

resource "helm_release" "eck-operator-crds" {
  name       = "elastic-operator-crds"
  repository = "https://helm.elastic.co"
  chart      = "eck-operator-crds"
  namespace  = "elastic-system"
  version    = "2.10.0"
}

resource "helm_release" "eck-operator" {
  name       = "elastic-operator"
  repository = "https://helm.elastic.co"
  chart      = "eck-operator"
  namespace  = "elastic-system"
  version    = "2.10.0"
  
  create_namespace = true
  
  values = [
    file("../helm/eck-operator/values.yaml")
  ]
}
