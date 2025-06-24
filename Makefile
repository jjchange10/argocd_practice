argo-port-forward:
	kubectl port-forward --address localhost -n argocd svc/argocd-server 8080:443

argo-get-secret:
	kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

