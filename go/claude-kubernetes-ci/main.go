package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"gopkg.in/yaml.v2"
)

type KustomizationConfig struct {
	Resources []string `yaml:"resources"`
	Patches   []struct {
		Target struct {
			Kind string `yaml:"kind"`
			Name string `yaml:"name"`
		} `yaml:"target"`
		Patch string `yaml:"patch"`
	} `yaml:"patches"`
	HelmCharts []struct {
		Name         string `yaml:"name"`
		Repo         string `yaml:"repo"`
		Version      string `yaml:"version"`
		ValuesFile   string `yaml:"valuesFile"`
		ReleaseName  string `yaml:"releaseName"`
		Namespace    string `yaml:"namespace"`
	} `yaml:"helmCharts"`
}

func main() {
	workdir := "helm"
	
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	
	targetEnv := os.Args[1]
	fmt.Printf("ワーキングディレクトリ: %s\n", workdir)
	fmt.Printf("対象環境: %s\n", targetEnv)
	
	pwd, _ := os.Getwd()
	fmt.Printf("現在のディレクトリ: %s\n", pwd)
	
	// 許可された環境をチェック
	switch targetEnv {
	case "stg", "prod":
		// OK
	default:
		fmt.Printf("[ERROR] 不明な環境: %s\n", targetEnv)
		os.Exit(1)
	}
	
	fmt.Println("[INFO] 変更されたディレクトリに対して kubectl diff を実行中")
	
	// ArgoCD サーバーの設定
	argoCdServer := os.Getenv("ARGOCD_SERVER")
	if argoCdServer == "" {
		argoCdServer = "localhost:8080"
	}
	fmt.Printf("ArgoCD サーバー: %s\n", argoCdServer)
	
	// Git差分をチェックして変更されたファイルを取得
	fmt.Println("[INFO] git diff で変更されたディレクトリをチェック中")
	
	changedFiles, err := getChangedFiles(workdir)
	if err != nil {
		fmt.Printf("[ERROR] 変更されたファイルの取得に失敗: %v\n", err)
		os.Exit(1)
	}
	
	if len(changedFiles) == 0 {
		fmt.Println("[INFO] 変更が検出されませんでした")
		os.Exit(0)
	}
	
	fmt.Println("[INFO] 変更されたファイル:")
	for _, file := range changedFiles {
		fmt.Printf("  %s\n", file)
	}
	
	// Kubernetesクライアントを初期化
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		fmt.Printf("[ERROR] Kubernetesクライアントの設定に失敗: %v\n", err)
		os.Exit(1)
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("[ERROR] Kubernetesクライアントの作成に失敗: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Kubernetesクライアント接続成功\n")
	
	// 各変更されたファイルを処理
	for _, file := range changedFiles {
		fmt.Printf("\n処理中のファイル: %s\n", file)
		
		// kustomization.yamlファイルを検索
		matchingFiles, err := findMatchingKustomizationFiles(targetEnv, file)
		if err != nil {
			fmt.Printf("[ERROR] マッチするファイルの検索に失敗: %v\n", err)
			continue
		}
		
		if len(matchingFiles) == 0 {
			fmt.Printf("[ERROR] %s にマッチするファイルが見つかりませんでした\n", file)
			continue
		}
		
		for _, matchingFile := range matchingFiles {
			fmt.Printf("見つかったファイル: '%s'\n", matchingFile)
			matchingDir := filepath.Dir(matchingFile)
			
			err := runKustomizeDiff(matchingDir, clientset)
			if err != nil {
				fmt.Printf("[ERROR] kustomize diff の実行に失敗: %v\n", err)
			}
		}
	}
}

func usage() {
	fmt.Println("")
	fmt.Printf("使用方法: %s ENV(stg|prod)\n", os.Args[0])
}

// getChangedFiles はGitライブラリを使用して変更されたファイルを取得
func getChangedFiles(basedir string) ([]string, error) {
	// 現在のディレクトリからGitリポジトリのルートを見つける
	repoPath := "."
	for i := 0; i < 10; i++ { // 最大10レベルまで親ディレクトリを探索
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
			break
		}
		repoPath = filepath.Join(repoPath, "..")
	}
	
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("Gitリポジトリのオープンに失敗: %v", err)
	}
	
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("HEADの取得に失敗: %v", err)
	}
	
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("コミットオブジェクトの取得に失敗: %v", err)
	}
	
	parentCommit, err := commit.Parent(0)
	if err != nil {
		return nil, fmt.Errorf("親コミットの取得に失敗: %v", err)
	}
	
	parentTree, err := parentCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("親コミットのツリーの取得に失敗: %v", err)
	}
	
	currentTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("現在のコミットのツリーの取得に失敗: %v", err)
	}
	
	changes, err := object.DiffTree(parentTree, currentTree)
	if err != nil {
		return nil, fmt.Errorf("差分の取得に失敗: %v", err)
	}
	
	var changedFiles []string
	for _, change := range changes {
		if change.From.Name != "" && strings.HasPrefix(change.From.Name, basedir) {
			changedFiles = append(changedFiles, change.From.Name)
		}
		if change.To.Name != "" && strings.HasPrefix(change.To.Name, basedir) {
			changedFiles = append(changedFiles, change.To.Name)
		}
	}
	
	// 重複を除去
	uniqueFiles := make(map[string]bool)
	var result []string
	for _, file := range changedFiles {
		if !uniqueFiles[file] {
			uniqueFiles[file] = true
			result = append(result, file)
		}
	}
	
	return result, nil
}

// findMatchingKustomizationFiles は指定されたファイルにマッチするkustomization.yamlファイルを検索
func findMatchingKustomizationFiles(targetEnv, file string) ([]string, error) {
	var matchingFiles []string
	
	// Gitリポジトリのルートを見つける
	repoPath := "."
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
			break
		}
		repoPath = filepath.Join(repoPath, "..")
	}
	
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.Name() == "kustomization.yaml" && strings.Contains(path, fmt.Sprintf("overlays/%s", targetEnv)) {
			// kustomization.yamlファイルの内容を読み取り
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			var config KustomizationConfig
			err = yaml.Unmarshal(content, &config)
			if err != nil {
				return err
			}
			
			// valuesFileにマッチするものを検索
			for _, chart := range config.HelmCharts {
				if strings.Contains(chart.ValuesFile, file) {
					matchingFiles = append(matchingFiles, path)
					break
				}
			}
		}
		
		return nil
	})
	
	return matchingFiles, err
}

// runKustomizeDiff はbashスクリプトと同じくkubectl diffを実行
func runKustomizeDiff(dir string, clientset *kubernetes.Clientset) error {
	fmt.Printf("ディレクトリ %s で kubectl diff を実行中\n", dir)
	
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("現在のディレクトリの取得に失敗: %v", err)
	}
	
	err = os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("ディレクトリの変更に失敗: %v", err)
	}
	defer os.Chdir(originalDir)
	
	// kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone . | kubectl diff -f -
	kustomizeCmd := exec.Command("kustomize", "build", "--enable-helm", "--load-restrictor=LoadRestrictionsNone", ".")
	kubectlCmd := exec.Command("kubectl", "diff", "-f", "-")
	
	// パイプラインを設定
	kubectlCmd.Stdin, err = kustomizeCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("パイプの設定に失敗: %v", err)
	}
	
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	
	err = kubectlCmd.Start()
	if err != nil {
		return fmt.Errorf("kubectl コマンドの開始に失敗: %v", err)
	}
	
	err = kustomizeCmd.Run()
	if err != nil {
		return fmt.Errorf("kustomize コマンドの実行に失敗: %v", err)
	}
	
	err = kubectlCmd.Wait()
	if err != nil {
		// kubectl diff は変更がある場合にexitコード1を返すので、これは正常
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return nil
		}
		return fmt.Errorf("kubectl diff コマンドの実行に失敗: %v", err)
	}
	
	return nil
}

// checkResourceExists はリソースがクラスター内に存在するかチェック
func checkResourceExists(clientset *kubernetes.Clientset, obj *unstructured.Unstructured) error {
	// 簡単な存在確認（実際のdiffは複雑なので、ここでは簡略化）
	gvk := obj.GroupVersionKind()
	
	switch gvk.Kind {
	case "Deployment":
		_, err := clientset.AppsV1().Deployments(obj.GetNamespace()).Get(
			context.TODO(), obj.GetName(), metav1.GetOptions{})
		return err
	case "Service":
		_, err := clientset.CoreV1().Services(obj.GetNamespace()).Get(
			context.TODO(), obj.GetName(), metav1.GetOptions{})
		return err
	case "ConfigMap":
		_, err := clientset.CoreV1().ConfigMaps(obj.GetNamespace()).Get(
			context.TODO(), obj.GetName(), metav1.GetOptions{})
		return err
	default:
		return fmt.Errorf("未サポートのリソース種別: %s", gvk.Kind)
	}
}