get_changed_files() {
    local basedir=$1
    git diff --name-only "HEAD^" "HEAD" ${basedir}
}

