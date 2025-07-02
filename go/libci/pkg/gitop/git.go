package gitop

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

const (
	baseBranch = "main"
)

type Client struct {
	repo *git.Repository
}

func NewClient(repoPath string) (*Client, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}
	return &Client{repo: repo}, nil
}

func (c *Client) GetChangedFiles(targetBranch string) ([]string, error) {
	if len(targetBranch) == 0 {
		currentBranch, err := c.getCurrentBranchName()
		if err != nil {
			return nil, fmt.Errorf("failed to get current branch")
		}
		targetBranch = currentBranch
	}
	return c.getDiffFilesBetweenBranches(targetBranch)
}

func (c *Client) getCurrentBranchName() (string, error) {
	headRef, err := c.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	if headRef.Name().IsBranch() {
		return headRef.Name().Short(), nil
	}

	return "", fmt.Errorf("HEAD is not a branch (detached HEAD)")
}

func (c *Client) getDiffFilesBetweenBranches(headRef string) ([]string, error) {
	baseRef, err := c.repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get base branch ref: %w", err)
	}

	headReference, err := c.repo.Reference(plumbing.NewBranchReferenceName(headRef), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get head ref: %w", err)
	}

	baseCommit, err := c.repo.CommitObject(baseRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}
	headCommit, err := c.repo.CommitObject(headReference.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit: %w", err)
	}

	mergeBaseCommits, err := baseCommit.MergeBase(headCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}
	if len(mergeBaseCommits) == 0 {
		return nil, fmt.Errorf("no merge base found")
	}
	mergeBase := mergeBaseCommits[0]

	mergeBaseTree, err := mergeBase.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base tree: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get head tree: %w", err)
	}

	changes, err := mergeBaseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	var changedFiles []string
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue
		}
		switch action {
		case merkletrie.Insert:
			changedFiles = append(changedFiles, change.To.Name)
		case merkletrie.Delete:
			changedFiles = append(changedFiles, change.From.Name)
		case merkletrie.Modify:
			// In practice, either change.To.Name or change.From.Name is fine.
			changedFiles = append(changedFiles, change.To.Name)
		default:
			// Currently, only Insert, Delete, or Modify actions are supported,
			// so this code should not be reached.
			continue
		}
	}

	return changedFiles, nil
}
