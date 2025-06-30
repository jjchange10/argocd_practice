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

# Git差分をチェックして変更されたディレクトリを検出
echo "[INFO] Checking git diff for changed directories"

# 変更されたファイルのリストを取得（メインブランチとの差分）
changed_files=$(git diff --name-only HEAD~1 HEAD $workdir)

if [ -z "$changed_files" ]; then
  echo "[INFO] No changes detected"
  exit 0
fi

echo "[INFO] Changed files:"
echo "$changed_files"

# 変更されたディレクトリから対応するアプリケーションファイルを検出
app_files=()

for file in $changed_files; do
  # ファイルのディレクトリパスを取得
  dir=$(dirname "$file")
  
  # applicationsディレクトリ配下でvaluesFile参照を持つファイルを検索し、ディレクトリのみ取得
  matching_dirs=$(grep -rl "valuesFile:.*${dir}" applications/ 2>/dev/null | xargs dirname | sort -u)

  result=$(kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone ${matching_dirs} | kubectl diff -f -)
  echo "$result"
done
