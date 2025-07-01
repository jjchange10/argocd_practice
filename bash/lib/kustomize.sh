kubectl_diff() {
    kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone . | kubectl diff -f -
}

argocd_diff() {
    local app_name=$1
    local revision=$2
    argocd app diff ${app_name} --revision ${revision} 
}
