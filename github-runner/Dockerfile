# Dockerfile.runner
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    jq \
    ca-certificates \
    gnupg \
    lsb-release \
    sudo

# Runnerバイナリのダウンロード（最新版を確認してください）
RUN curl -o actions-runner-linux-x64-2.325.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.325.0/actions-runner-linux-x64-2.325.0.tar.gz
RUN echo "5020da7139d85c776059f351e0de8fdec753affc9c558e892472d43ebeb518f4  actions-runner-linux-x64-2.325.0.tar.gz" | shasum -a 256 -c
RUN tar xzf ./actions-runner-linux-x64-2.325.0.tar.gz

RUN sudo ./config.sh --url https://github.com/jjchange10/argocd_practice --token BCX7RC2KOT6NL5XLVSNZUIDIKVSBO
CMD ["./run.sh"]