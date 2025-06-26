usage() {
  echo ""
  echo "Usage: ${0} WORKDIR ENV(stg|prod)"
}

declare workdir="${1}"
declare target_env="${2}"
echo "$workdir"
echo "$target_env"
pwd

if [ $# -ne 2 ]; then
  usage
  exit 1
fi

# Get changed directories from git diff
changed_dirs=$(git diff --name-only HEAD~1 | xargs -I {} dirname {} | sort -u)
echo "[INFO] Changed directories: $changed_dirs"

# Extract service names from changed directories (helm only)
service_names=()
for dir in $changed_dirs; do
  # Extract service name from helm path (e.g., helm/nginx -> nginx)
  if [[ "$dir" =~ ^helm/([^/]+) ]]; then
    service_name="${BASH_REMATCH[1]}"
    service_names+=("$service_name")
  fi
done

# Remove duplicates
service_names=($(printf "%s\n" "${service_names[@]}" | sort -u))

if [ ${#service_names[@]} -eq 0 ]; then
  echo "[INFO] No service changes detected"
  exit 0
fi

echo "[INFO] Detected services: ${service_names[*]}"

# Declare allowed stages based on the target environment
case "${target_env}" in
  stg) allowed_stage_regex='^(stg)$' ;;
  prod) allowed_stage_regex='^(prod)$' ;;
  *) echo "[ERROR] Unknown env: ${target_env}" && exit 1 ;;
esac


echo "[INFO] Running 'argocd app sync' for changed dirs"

# ArgoCD CLI でログインする
echo "[INFO] Logging in to ArgoCD..."

# ArgoCD サーバーの URL を設定
ARGOCD_SERVER="${ARGOCD_SERVER:-localhost:8080}"

# ArgoCD の管理者パスワードを取得

ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# ArgoCD CLI でログイン
argocd login "${ARGOCD_SERVER}" --username admin --password "${ARGOCD_PASSWORD}" --insecure

if [ $? -ne 0 ]; then
  echo "[ERROR] Failed to login to Argo"
  exit 1
fi

# Process each detected service
for service_name in "${service_names[@]}"; do
  echo "[INFO] Processing service: ${service_name}"
  
  argocd app get ${service_name}
  if [ $? -ne 0 ]; then
    echo "[ERROR] Failed to get ArgoCD app: ${service_name}"
    continue
  fi

  echo "[INFO] Diffing ArgoCD app: ${service_name}"
  result=$(argocd app diff ${service_name} --source-positions 2 --revisions development)

  # ArgoCD app diffの終了コードを取得
  diff_exit_code=$?

  if [ $diff_exit_code -eq 1 ]; then
    echo "[INFO] ArgoCD app diff result:"
    echo "$result"
    echo "[INFO] Differences found - proceeding with sync"
    
    # 差分がある場合はsyncを実行
    echo "[INFO] Syncing ArgoCD app: ${service_name}"
    
  elif [ $diff_exit_code -eq 0 ]; then
    echo "[INFO] No differences found for ${service_name} - app is already in sync"
  else
    echo "[ERROR] Failed to diff ArgoCD app: ${service_name} (exit code: $diff_exit_code)"
  fi
done
