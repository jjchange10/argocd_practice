usage() {
  echo ""
  echo "Usage: ${0} WORKDIR ENV(stg|prod)"
}

declare basedir="${1}"
declare target_env="${2}"
echo "$basedir"
echo "$target_env"
pwd

if [ $# -ne 2 ]; then
  usage
  exit 1
fi

if ! ls "${basedir}" 1>/dev/null 2>/dev/null; then
  echo "[ERROR] No such directory: ${basedir}"
  exit 1
fi

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






