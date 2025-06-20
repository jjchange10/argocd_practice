# Kindクラスタの構築

kind create cluster --config kind.yaml --name argocd-practice

## argoCDインストール
```bash
helm upgrade --install argocd argo/argo-cd --create-name=true -n argocd -f helm/argocd/values.yaml
```

```bash
kubectl port-forward service/argocd-server -n argocd 8080:443
```

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```
