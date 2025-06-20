argo-port-forward:
	kubectl port-forward service/argocd-server -n argocd 8080:443

argo-get-secret:
	kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# ESO Application commands
eso-validate:
	helm repo add external-secrets https://charts.external-secrets.io
	helm repo update
	helm template eso-test external-secrets/external-secrets \
		--version 0.9.0 \
		-f helm/argocd/eso/values.yaml \
		--dry-run

eso-dry-run:
	kubectl apply --dry-run=client -f applications/eso.yaml

eso-deploy:
	kubectl apply -f applications/eso.yaml

eso-status:
	kubectl get application eso -n argocd -o yaml

eso-logs:
	kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=100 -f

# CI commands
ci-local:
	@echo "Running local CI checks..."
	@make eso-validate
	@make eso-dry-run
	@echo "✅ Local CI checks passed"
