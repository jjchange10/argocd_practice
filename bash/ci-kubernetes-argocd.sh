source bash/lib/git.sh
source bash/lib/kustomize.sh

set -x

usage() {
  echo ""
  echo "Usage: ${0} WORKDIR ENV(stg|prod)"
}

declare workdir="helm"
declare target_env=$1
echo "$workdir"
echo "$target_env"
pwd

if [ $# -ne 1 ]; then
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

# ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# # ArgoCD CLI でログイン
# argocd login "${ARGOCD_SERVER}" --username admin --password "${ARGOCD_PASSWORD}" --insecure

# if [ $? -ne 0 ]; then
#   echo "[ERROR] Failed to login to Argo"
#   exit 1
# fi

# Git差分をチェックして変更されたディレクトリを検出
echo "[INFO] Checking git diff for changed directories"

# 変更されたファイルのリストを取得（メインブランチとの差分）
changed_files=$(get_changed_files $workdir)

if [ -z "$changed_files" ]; then
  echo "[INFO] No changes detected"
  exit 0
fi

echo "[INFO] Changed files:"
echo "$changed_files"

for file in $changed_files; do
  # ファイルのディレクトリパスを取得
  # rest="${file#helm/}"
  # app="${rest%%/*}"
  # echo "app: $app"
  # echo "file: $file"
  file_pattern="${file}"

  matching_file=$(find ./**/overlays/${target_env} -name "kustomization.yaml" -exec grep -l "valuesFile:.*${file_pattern}" {} \; 2>&1)
  echo "Find result: '${matching_file}'"
  if [ -n "$matching_file" ]; then
    matching_dirs=$(dirname ${matching_file})
  else
    echo "[ERROR] No matching file found for ${app}"
    continue
  fi

  if [ -n "$matching_dirs" ]; then 
    #(cd ${matching_dirs} && result=$(kubectl_diff) && echo "$result")
    cd ${matching_dirs}
    result=$(kubectl_diff)
    echo "$result"
    cd -
  fi
done
