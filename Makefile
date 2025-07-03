argo-port-forward:
	kubectl port-forward --address localhost -n argocd svc/argocd-server 8080:443

argo-get-secret:
	argocd admin initial-password -n argocd
